/**
 * Energy-based Voice Activity Detection (VAD) with adaptive noise floor.
 *
 * State machine (correct):
 *
 *   IDLE ──[above threshold for onsetDebounceS]──→ SPEAKING
 *   SPEAKING ──[below threshold for silenceDurationS]──→ IDLE
 *   SPEAKING ──[exceeds maxSpeechDurationS]──→ IDLE (hard cutoff, optional)
 *
 * Key: speech ends when signal GOES BELOW threshold for N seconds,
 *      NOT when speech has lasted N seconds.
 *
 * Design principles:
 *   1. Sample-counting timeline — NOT wall-clock timestamps.
 *      Duration = totalSamplesProcessed / sampleRate.
 *      Immune to GC pauses, event loop lag, and chunk-size variance.
 *   2. Adaptive noise floor — exponential moving average tracks ambient noise.
 *      Threshold = noiseFloor × multiplier. Self-adjusts to quiet rooms, fans, AC.
 *   3. Aligned buffer reads — always copy to aligned Int16Array.
 *      Prevents RangeError from odd byteOffset and avoids shared pool memory mirroring.
 *   4. Bounded event listeners — setMaxListeners in initialize(), expose once().
 *
 * Architecture role: sits between Transport and STT.
 *   Transport → VAD → STT (only processes audio when speech is active)
 *
 * Lifecycle: initialize() → isSpeaking()/getState() → destroy()
 */

import { EventEmitter } from 'events';
import { logger } from '../utils/logger.js';
import type {
  VoiceActivityDetector,
  AudioChunk,
  AudioEventMap,
  SpeechState,
  VADConfig,
} from './types.js';

// ----- Defaults -----

const DEFAULT_SILENCE_DURATION_S = 0.7;
const DEFAULT_ONSET_DEBOUNCE_S = 0.2;
const DEFAULT_MAX_SPEECH_DURATION_S = 30;
const DEFAULT_NOISE_ADAPT_RATE = 0.05;
const DEFAULT_NOISE_FLOOR_MIN = 0.005;
const DEFAULT_SPEECH_MULTIPLIER = 2.5;
const DEFAULT_MAX_LISTENERS = 20;

/**
 * Creates an energy-based Voice Activity Detector with adaptive noise floor.
 *
 * @param initialConfig - VAD sensitivity configuration
 */
export function createVAD(initialConfig: VADConfig = {}): VoiceActivityDetector {
  // ----- Thresholds (mutable via updateConfig) -----

  /** Seconds of silence below threshold before speech_ended fires. */
  let silenceDurationS = (initialConfig.silenceThresholdMs ?? 700) / 1000;
  /** Seconds of sustained energy above threshold before speech_started fires. */
  let onsetDebounceS = (initialConfig.speechThresholdMs ?? 200) / 1000;
  /** Hard maximum speech duration — forces speech_ended even if still talking. 0 = disabled. */
  let maxSpeechDurationS = DEFAULT_MAX_SPEECH_DURATION_S;

  // ----- Adaptive noise floor -----

  let noiseFloor = DEFAULT_NOISE_FLOOR_MIN;
  const speechMultiplier = DEFAULT_SPEECH_MULTIPLIER;
  const noiseAdaptRate = DEFAULT_NOISE_ADAPT_RATE;
  const noiseFloorMin = DEFAULT_NOISE_FLOOR_MIN;

  /** Sample rate of incoming audio — set from first chunk if not configured. */
  let sampleRate = initialConfig.sampleRate ?? 16000;

  const emitter = new EventEmitter();

  // ----- Sample-counting timeline (NOT wall-clock) -----

  /** Total samples processed since initialize(). */
  let totalSamplesProcessed = 0;
  /** Samples at the start of the current speech segment (when speech_started fired). */
  let speechStartSamples = 0;

  // ----- Counters for sustained-state detection -----

  /** Samples accumulated above threshold while in IDLE (onset debounce). */
  let aboveThresholdSamples = 0;
  /** Samples accumulated below threshold while in SPEAKING (silence detection). */
  let belowThresholdSamples = 0;

  // ----- State flags -----

  let _isSpeaking = false;
  let initialized = false;
  /** Track whether silence was already detected to avoid repeated events per chunk. */
  let silenceAlreadyDetected = false;

  // ----- Last RMS for UI meters -----

  let _lastRMS = 0;

  // ----- Derived helpers (seconds ↔ samples) -----

  function samplesToSeconds(samples: number): number {
    return samples / sampleRate;
  }

  /** Current adaptive speech threshold (noise floor × multiplier). */
  function currentThreshold(): number {
    return Math.max(noiseFloor * speechMultiplier, noiseFloorMin * speechMultiplier);
  }

  // ----- Core Methods -----

  function initialize(): Promise<void> {
    if (initialized) return Promise.resolve();
    initialized = true;

    // Bound listeners to prevent memory leak warnings
    emitter.setMaxListeners(DEFAULT_MAX_LISTENERS);

    logger.debug({
      message: 'VAD initialized',
      noiseFloor,
      speechMultiplier,
      noiseAdaptRate,
      silenceDurationS,
      onsetDebounceS,
      maxSpeechDurationS,
      sampleRate,
    });
    return Promise.resolve();
  }

  function isSpeaking(audio: AudioChunk): boolean {
    const samples = alignedSamples(audio);
    if (samples.length === 0) return _isSpeaking;

    // Detect sample rate from first chunk (override config if present)
    if (totalSamplesProcessed === 0 && audio.sampleRate > 0) {
      sampleRate = audio.sampleRate;
    }

    const rms = computeRMS(samples);
    _lastRMS = rms;
    const nowSamples = totalSamplesProcessed + samples.length;

    const threshold = currentThreshold();
    const aboveThreshold = rms >= threshold;

    // ----- Adaptive noise floor -----
    // Track ambient noise ONLY when below threshold and not speaking.
    // Prevents user's voice from raising the noise floor.
    if (!aboveThreshold && !_isSpeaking) {
      noiseFloor = noiseFloor * (1 - noiseAdaptRate) + rms * noiseAdaptRate;
      if (noiseFloor < noiseFloorMin) {
        noiseFloor = noiseFloorMin;
      }
    }

    // ----- State machine -----

    if (_isSpeaking) {
      // === CURRENTLY SPEAKING ===

      if (aboveThreshold) {
        // Still speaking — reset silence counter
        belowThresholdSamples = 0;
        silenceAlreadyDetected = false;

        // Hard cutoff: force speech_ended if segment exceeds max duration
        if (maxSpeechDurationS > 0) {
          const segmentSamples = nowSamples - speechStartSamples;
          const segmentDurationS = samplesToSeconds(segmentSamples);
          if (segmentDurationS >= maxSpeechDurationS) {
            _isSpeaking = false;
            const ts = new Date().toISOString();
            emitEvent('speech_ended', ts, segmentDurationS * 1000);
            totalSamplesProcessed = nowSamples;
            return false;
          }
        }
      } else {
        // Below threshold — accumulate silence samples
        belowThresholdSamples += samples.length;
        const silenceSec = samplesToSeconds(belowThresholdSamples);

        if (silenceSec >= silenceDurationS && !silenceAlreadyDetected) {
          // Silence confirmed — end speech
          _isSpeaking = false;
          silenceAlreadyDetected = true;
          const segmentSamples = nowSamples - belowThresholdSamples - speechStartSamples;
          const segmentDurationS = samplesToSeconds(segmentSamples);
          const ts = new Date().toISOString();
          emitEvent('speech_ended', ts, segmentDurationS * 1000);
          totalSamplesProcessed = nowSamples;
          return false;
        }
      }
    } else {
      // === CURRENTLY IDLE ===

      silenceAlreadyDetected = false;

      if (aboveThreshold) {
        // Above threshold — accumulate onset samples
        aboveThresholdSamples += samples.length;
        const onsetSec = samplesToSeconds(aboveThresholdSamples);

        if (onsetSec >= onsetDebounceS) {
          // Onset confirmed — start speech
          _isSpeaking = true;
          speechStartSamples = nowSamples - aboveThresholdSamples; // backdate to when energy started
          aboveThresholdSamples = 0;
          belowThresholdSamples = 0;
          const ts = new Date().toISOString();
          emitEvent('speech_started', ts);
        }
      } else {
        // Still silent — reset onset counter
        aboveThresholdSamples = 0;
      }
    }

    totalSamplesProcessed = nowSamples;
    return _isSpeaking;
  }

  function getState(): SpeechState | null {
    if (!initialized) return null;

    const currentSegmentMs = _isSpeaking
      ? samplesToSeconds(totalSamplesProcessed - speechStartSamples) * 1000
      : 0;

    const silenceGapMs = (!_isSpeaking && belowThresholdSamples > 0)
      ? samplesToSeconds(belowThresholdSamples) * 1000
      : 0;

    return {
      isSpeaking: _isSpeaking,
      currentSegmentMs,
      silenceGapMs,
      energyLevel: _lastRMS,
    };
  }

  function on<K extends keyof AudioEventMap>(event: K, handler: AudioEventMap[K]): void {
    emitter.on(event, handler);
  }

  function once<K extends keyof AudioEventMap>(event: K, handler: AudioEventMap[K]): void {
    emitter.once(event, handler);
  }

  function off<K extends keyof AudioEventMap>(event: K, handler: AudioEventMap[K]): void {
    emitter.off(event, handler);
  }

  function updateConfig(newConfig: VADConfig): void {
    if (newConfig.silenceThresholdMs !== undefined) {
      silenceDurationS = newConfig.silenceThresholdMs / 1000;
    }
    if (newConfig.speechThresholdMs !== undefined) {
      onsetDebounceS = newConfig.speechThresholdMs / 1000;
    }
    if (newConfig.sampleRate !== undefined && newConfig.sampleRate > 0) {
      sampleRate = newConfig.sampleRate;
    }
    logger.debug({
      message: 'VAD config updated',
      silenceDurationS,
      onsetDebounceS,
      maxSpeechDurationS,
      sampleRate,
      noiseFloor,
      threshold: currentThreshold(),
    });
  }

  function destroy(): Promise<void> {
    emitter.removeAllListeners();
    _isSpeaking = false;
    totalSamplesProcessed = 0;
    speechStartSamples = 0;
    aboveThresholdSamples = 0;
    belowThresholdSamples = 0;
    initialized = false;
    silenceAlreadyDetected = false;
    _lastRMS = 0;
    noiseFloor = DEFAULT_NOISE_FLOOR_MIN;
    return Promise.resolve();
  }

  // ----- Helpers -----

  /**
   * Extract aligned Int16Array samples from AudioChunk.
   * Always copies to a new ArrayBuffer to avoid shared pool memory mirroring
   * and odd-byteOffset RangeError crashes.
   */
  function alignedSamples(audio: AudioChunk): Int16Array {
    const data = audio.data;
    if (!data || data.byteLength === 0) return new Int16Array(0);

    const byteLength = data.byteLength;
    const sampleCount = Math.floor(byteLength / 2);

    if (sampleCount === 0) return new Int16Array(0);

    // Allocate a fresh aligned buffer — no shared pool, no offset issues
    const aligned = new ArrayBuffer(sampleCount * 2);
    const view = new Int16Array(aligned);

    if (data instanceof ArrayBuffer) {
      view.set(new Int16Array(data));
    } else {
      // Buffer — use readInt16LE for safe, aligned reads
      for (let i = 0; i < sampleCount; i++) {
        view[i] = data.readInt16LE(data.byteOffset + i * 2);
      }
    }

    return view;
  }

  /**
   * Compute Root Mean Square energy of aligned Int16Array (normalized 0-1).
   * Runs synchronously — acceptable for single-interview local-first tool.
   * For 50+ concurrent streams, offload to worker thread.
   */
  function computeRMS(samples: Int16Array): number {
    if (samples.length === 0) return 0;

    let sumSquares = 0;
    for (let i = 0; i < samples.length; i++) {
      const normalized = samples[i] / 32768;
      sumSquares += normalized * normalized;
    }

    return Math.sqrt(sumSquares / samples.length);
  }

  /** Safe emit — don't throw if no listener attached. Type-safe event names. */
  function emitEvent<K extends keyof AudioEventMap>(
    event: K,
    ...args: Parameters<AudioEventMap[K]>
  ): void {
    if (emitter.listenerCount(event) > 0) {
      emitter.emit(event, ...args);
    }
  }

  return {
    name: 'adaptive-energy-vad',
    initialize,
    isSpeaking,
    getState,
    on,
    once,
    off,
    updateConfig,
    destroy,
  };
}
