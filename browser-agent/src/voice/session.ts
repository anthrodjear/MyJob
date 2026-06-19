/**
 * Interview Session — pure orchestrator.
 *
 * Receives ALL dependencies via injection. Creates nothing.
 * Owns only: state machine, event routing, audio pipeline, lifecycle.
 *
 * Zero business logic. Zero config reads. Zero concrete implementations.
 *
 * Dependencies (injected via SessionDeps):
 *   - transport: LiveKit audio transport
 *   - brain: interview intelligence (planner+responder+memory+retriever)
 *   - audioQueue: sequential audio segment processing
 *   - vad: voice activity detection (separate STT/TTS path)
 *   - stt: speech-to-text (separate STT/TTS path)
 *   - tts: text-to-speech (separate STT/TTS path)
 *   - realtime: combined STT+TTS provider (realtime path)
 *
 * State machine:
 *   idle → connecting → listening → thinking → speaking → listening → ... → ended
 */

import { EventEmitter } from 'events';
import { logger } from '../utils/logger.js';
import type {
  InterviewSession,
  InterviewMode,
  SessionState,
  TranscriptSegment,
  AudioChunk,
  BrainResponse,
  Transport,
  Brain,
  AudioSegmentQueue,
  VoiceActivityDetector,
  STTProvider,
  TTSProvider,
  RealtimeProvider,
  SessionEventMap,
  VoiceSessionConfig,
} from './types.js';
import { createAudioSegmentQueue } from './queue.js';

const log = logger.child({ component: 'InterviewSession' });

/** Dependencies injected into the session. All optional except transport + brain. */
export interface SessionDeps {
  transport: Transport;
  brain: Brain;
  /** VAD for separate STT/TTS path. Required when realtime is not provided. */
  vad?: VoiceActivityDetector;
  /** STT provider for separate path. Required when realtime is not provided. */
  stt?: STTProvider;
  /** TTS provider for separate path. Required when realtime is not provided. */
  tts?: TTSProvider;
  /** Realtime provider for combined STT+TTS path. Required when stt/tts are not provided. */
  realtime?: RealtimeProvider;
}

/**
 * Creates an InterviewSession orchestrator.
 *
 * All dependencies are injected — the session creates nothing.
 * The factory (createInterviewSession) handles all wiring.
 */
export function createInterviewSession(deps: SessionDeps): InterviewSession {
  // ----- State -----

  let _state: SessionState = 'idle';
  let _mode: InterviewMode = 'autonomous';
  let _roomName = '';
  let _applicationId = '';
  let _stopping = false;

  const emitter = new EventEmitter();

  // ----- Audio pipeline state -----

  /** Per-participant audio buffers accumulating raw audio during speech segments. */
  const speechAudioBuffers = new Map<string, Buffer[]>();

  /** Participant identity of the interviewer (set on first non-local participant). */
  let interviewerParticipantId: string | null = null;

  /** Local participant identity (our agent). */
  let localParticipantId: string | null = null;

  // ----- Accessors -----

  function getState(): SessionState {
    return _state;
  }

  function getMode(): InterviewMode {
    return _mode;
  }

  // ----- Event handlers (type-safe) -----

  function on<K extends keyof SessionEventMap>(event: K, handler: SessionEventMap[K]): void {
    emitter.on(event, handler);
  }

  function off<K extends keyof SessionEventMap>(event: K, handler: SessionEventMap[K]): void {
    emitter.off(event, handler);
  }

  // ----- State management -----

  /** Allowed transitions — enforces valid state machine paths. */
  const validTransitions: Record<SessionState, SessionState[]> = {
    idle: ['connecting'],
    connecting: ['listening', 'error', 'ended'],
    listening: ['thinking', 'speaking', 'error', 'ended'],
    thinking: ['speaking', 'listening', 'error', 'ended'],
    speaking: ['listening', 'error', 'ended'],
    ended: [],
    error: ['ended'],
    reconnecting: ['listening', 'error', 'ended'],
  };

  function setState(state: SessionState, context?: { reason?: string; error?: Error }): void {
    if (_state === state) return;

    // Validate transition
    const allowed = validTransitions[_state];
    if (!allowed?.includes(state)) {
      log.warn({
        message: 'Invalid state transition blocked',
        from: _state,
        to: state,
        reason: context?.reason,
      });
      return;
    }

    const previous = _state;
    _state = state;

    log.info({ message: 'State changed', from: previous, to: state, reason: context?.reason });

    if (context?.error) {
      log.error({ message: 'Session error', phase: state, error: context.error });
      emitError(context.error, state as 'connecting' | 'listening' | 'thinking' | 'speaking' | 'cleanup');
    }

    emitEvent('stateChanged', state);
  }

  // ----- Lifecycle: start -----

  async function start(config: VoiceSessionConfig): Promise<void> {
    if (_state !== 'idle') {
      throw new Error(`Cannot start session in state '${_state}' — expected 'idle'`);
    }

    _mode = config.mode;
    _roomName = config.roomName;
    _applicationId = config.applicationId;
    _stopping = false;

    setState('connecting');

    try {
      // 1. Connect transport
      await deps.transport.connect(config.roomName, config.token);
      wireTransportEvents();

      // 2. Initialize brain
      const context = config.context ?? { resume: '', jobDescription: '' };
      await deps.brain.initialize(context);

      // 3. Connect providers
      if (deps.realtime) {
        wireRealtimeEvents();
        await deps.realtime.connect();
      } else {
        if (deps.stt) await deps.stt.initialize();
        if (deps.tts) await deps.tts.initialize();
        if (deps.vad) await deps.vad.initialize();

        // Create audio queue with transcript callback to brain pipeline
        const audioQueue = createAudioSegmentQueue(
          async (audioBuffer: Buffer, sampleRate: number, channels: number) => {
            if (!deps.stt) return null;

            const audioChunk: AudioChunk = { data: audioBuffer, sampleRate, channels };
            return deps.stt.transcribe(audioChunk);
          },
          handleTranscript,
        );

        // Store queue reference for cleanup
        (deps as any)._audioQueue = audioQueue;
      }

      // 4. Start listening
      setState('listening');
      emitEvent('started', _mode, _roomName);

      log.info({
        message: 'Session started',
        mode: _mode,
        room: _roomName,
        provider: deps.realtime ? 'realtime' : 'separate',
      });
    } catch (err) {
      const error = err instanceof Error ? err : new Error(String(err));
      setState('error', { reason: 'Failed to start session', error });
      await cleanup();
      throw error;
    }
  }

  // ----- Lifecycle: stop -----

  async function stop(reason: string): Promise<TranscriptSegment[]> {
    if (_state === 'ended' || _stopping) {
      return deps.brain.memory.state.recentTranscript;
    }

    _stopping = true;
    log.info({ message: 'Stopping session', reason });

    const transcript = [...deps.brain.memory.state.recentTranscript];

    setState('ended', { reason });
    await cleanup();

    emitEvent('ended', reason, transcript);
    return transcript;
  }

  // ----- Cleanup -----

  async function cleanup(): Promise<void> {
    const errors: unknown[] = [];

    const audioQueue = (deps as any)._audioQueue;
    if (audioQueue) {
      audioQueue.clear();
    }

    try { deps.brain.destroy(); } catch (e) { errors.push(e); }
    if (deps.vad) { try { await deps.vad.destroy(); } catch (e) { errors.push(e); } }
    if (deps.tts) { try { await deps.tts.destroy(); } catch (e) { errors.push(e); } }
    if (deps.stt) { try { await deps.stt.destroy(); } catch (e) { errors.push(e); } }
    if (deps.realtime) {
      try { await deps.realtime.disconnect(); } catch (e) { errors.push(e); }
      try { await deps.realtime.destroy(); } catch (e) { errors.push(e); }
    }
    try { await deps.transport.disconnect(); } catch (e) { errors.push(e); }

    speechAudioBuffers.clear();
    interviewerParticipantId = null;

    if (errors.length > 0) {
      log.error({
        message: 'Cleanup errors',
        errors: errors.map((e) => (e instanceof Error ? e.message : String(e))),
      });
    }
  }

  // ----- Transport event wiring -----

  function wireTransportEvents(): void {
    deps.transport.on('connected', (roomName) => {
      log.info({ message: 'Transport connected', roomName });
    });

    deps.transport.on('disconnected', (reason) => {
      log.info({ message: 'Transport disconnected', reason });
      if (_state !== 'ended' && _state !== 'error') {
        setState('error', { reason: `Transport disconnected: ${reason ?? 'unknown'}` });
      }
    });

    deps.transport.on('audioReceived', (audio, participantId) => {
      // Identify interviewer: first non-local participant to join
      // We track local participant via participantJoined with kind='local'
      if (interviewerParticipantId === null && participantId !== localParticipantId) {
        interviewerParticipantId = participantId;
        log.info({ message: 'Identified interviewer', participantId });
      }

      // Only process interviewer's audio (skip local/agent audio)
      if (interviewerParticipantId !== null && participantId === interviewerParticipantId) {
        handleIncomingAudio(audio, participantId);
      }
    });

    deps.transport.on('participantJoined', (identity, kind) => {
      if (kind === 'local') {
        localParticipantId = identity;
        log.info({ message: 'Local participant identified', identity });
      }
    });

    deps.transport.on('error', (error) => {
      log.error({ message: 'Transport error', error });
      setState('error', { reason: 'Transport error', error });
    });
  }

  // ----- Realtime provider event wiring -----

  function wireRealtimeEvents(): void {
    if (!deps.realtime) return;

    deps.realtime.on('transcript', (segment) => {
      handleTranscript(segment);
    });

    deps.realtime.on('audio', (audio) => {
      if (deps.transport.connected) {
        deps.transport.publishAudio(audio);
      }
    });

    deps.realtime.on('error', (error) => {
      log.error({ message: 'Realtime provider error', error });
      setState('error', { reason: 'Realtime provider error', error });
    });
  }

  // ----- Audio pipeline -----

  function handleIncomingAudio(audio: AudioChunk, participantId: string): void {
    if (_state !== 'listening' && _state !== 'speaking') return;

    // Realtime path: forward directly to provider
    if (deps.realtime) {
      deps.realtime.sendAudio(audio);
      return;
    }

    // Separate STT/TTS path: VAD → queue → STT → Brain → TTS
    if (!deps.vad) return;

    const isSpeaking = deps.vad.isSpeaking(audio);

    if (isSpeaking) {
      // Accumulate audio for STT processing (per participant)
      const buffer = speechAudioBuffers.get(participantId) ?? [];
      const data = audio.data instanceof Buffer
        ? audio.data
        : Buffer.from(new Uint8Array(audio.data));
      buffer.push(data);
      speechAudioBuffers.set(participantId, buffer);
    } else {
      // Speech ended — enqueue accumulated audio for processing
      const buffer = speechAudioBuffers.get(participantId);
      if (buffer && buffer.length > 0) {
        const fullAudio = Buffer.concat(buffer);
        speechAudioBuffers.set(participantId, []);

        const audioQueue = (deps as any)._audioQueue;
        if (audioQueue) {
          audioQueue.enqueue(fullAudio, audio.sampleRate, audio.channels)
            .catch((err: Error) => {
              log.error({ message: 'Audio queue enqueue failed', error: err });
            });
        }
      }
    }
  }

  // ----- Brain pipeline -----

  /**
   * Called by audio queue's processFn (via deps.onTranscript) after STT.
   * Handles the transcript through the brain pipeline.
   */
  async function handleTranscript(segment: TranscriptSegment): Promise<void> {
    if (_state === 'ended' || _state === 'error') return;

    emitEvent('transcript', segment);

    // Let brain handle everything: plan → retrieve → generate → memory
    try {
      const response = await deps.brain.respond(segment);
      await handleBrainResponse(response);
    } catch (err) {
      const error = err instanceof Error ? err : new Error(String(err));
      log.error({ message: 'Brain processing failed', error });
      setState('error', { reason: 'Brain processing failed', error });
    }
  }

  async function handleBrainResponse(response: BrainResponse): Promise<void> {
    if (_state === 'ended' || _state === 'error') return;

    if (response.strategy === 'silent') {
      setState('listening');
      return;
    }

    if (response.speech) {
      emitEvent('agentSpeech', response.speech, response.confidence);

      // Capture state before async speak — prevent race where disconnect
      // changes state during speakText(), then we incorrectly overwrite it.
      setState('speaking');
      await speakText(response.speech);

      // Only transition back to listening if still in 'speaking' state.
      // If an error/disconnect occurred, state would be 'error' or 'ended'.
      if (_state === 'speaking') {
        setState('listening');
      }
    }
  }

  async function speakText(text: string): Promise<void> {
    if (!text) return;

    // Realtime path
    if (deps.realtime) {
      try {
        await deps.realtime.speak(text);
      } catch (err) {
        log.error({ message: 'Realtime speak failed', error: err });
        setState('error', { reason: 'Realtime speak failed', error: err instanceof Error ? err : new Error(String(err)) });
      }
      return;
    }

    // Separate TTS path
    if (!deps.tts || !deps.transport) {
      log.warn({ message: 'Cannot speak: no TTS provider or transport' });
      return;
    }

    // LiveKit requires 48kHz audio. Resample TTS output if needed.
    const TARGET_SAMPLE_RATE = 48000;

    function resampleAudio(chunk: AudioChunk): AudioChunk {
      if (chunk.sampleRate === TARGET_SAMPLE_RATE) return chunk;

      const ratio = TARGET_SAMPLE_RATE / chunk.sampleRate;
      const input = chunk.data instanceof Buffer
        ? new Int16Array(chunk.data.buffer, chunk.data.byteOffset, chunk.data.byteLength / 2)
        : new Int16Array(chunk.data);
      const outputLength = Math.round(input.length * ratio * chunk.channels);
      const output = new Int16Array(outputLength);

      for (let i = 0; i < outputLength; i++) {
        const srcIdx = i / ratio;
        const srcIdx0 = Math.floor(srcIdx);
        const frac = srcIdx - srcIdx0;
        const val0 = input[srcIdx0] ?? 0;
        const val1 = input[srcIdx0 + 1] ?? 0;
        output[i] = Math.round(val0 + frac * (val1 - val0));
      }

      return {
        data: Buffer.from(output.buffer),
        sampleRate: TARGET_SAMPLE_RATE,
        channels: chunk.channels,
      };
    }

    try {
      for await (const audioChunk of deps.tts.synthesize(text)) {
        if (!deps.transport.connected) break;
        const resampled = resampleAudio(audioChunk);
        await deps.transport.publishAudio(resampled);
      }
    } catch (err) {
      log.error({ message: 'TTS synthesis failed', error: err });
    }
  }

  // ----- Event emission helpers -----

  function emitEvent<K extends keyof SessionEventMap>(
    event: K,
    ...args: Parameters<SessionEventMap[K]>
  ): void {
    if (emitter.listenerCount(event) > 0) {
      emitter.emit(event, ...args);
    }
  }

  function emitError(error: Error, phase: 'connecting' | 'listening' | 'thinking' | 'speaking' | 'cleanup'): void {
    emitEvent('error', error, phase);
  }

  // ----- Public interface -----

  return {
    get state() {
      return getState();
    },
    get mode() {
      return getMode();
    },
    start,
    stop,
    setState,
    on,
    off,
  };
}
