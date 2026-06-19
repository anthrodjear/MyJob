/**
 * Audio Segment Queue — sequential processing of transcribed audio.
 *
 * Fixes the race condition: when audio arrives while the previous segment
 * is being transcribed, it's queued instead of dropped.
 *
 * FIFO order guarantees: segment N completes before segment N+1 starts.
 *
 * Usage:
 *   const queue = createAudioSegmentQueue(processFn);
 *   queue.enqueue(buffer1, 16000, 1); // starts immediately
 *   queue.enqueue(buffer2, 16000, 1); // queued, runs after #1
 *   queue.clear(); // drop pending on session stop
 */

import { logger } from '../utils/logger.js';
import type { AudioSegmentQueue, TranscriptSegment } from './types.js';

interface QueuedSegment {
  audio: Buffer;
  sampleRate: number;
  channels: number;
  resolve: () => void;
  reject: (err: Error) => void;
}

/**
 * Creates an audio segment queue that processes segments sequentially.
 *
 * @param processFn - Function to process each segment (e.g., STT → Brain → TTS).
 *                    Should return the TranscriptSegment if STT was performed,
 *                    or null if no transcription was needed.
 * @param onTranscript - Callback invoked with transcript after STT completes.
 *                       Required to forward transcripts to the brain pipeline.
 */
export function createAudioSegmentQueue(
  processFn: (audio: Buffer, sampleRate: number, channels: number) => Promise<TranscriptSegment | null>,
  onTranscript: (segment: TranscriptSegment) => Promise<void>,
): AudioSegmentQueue {
  const queue: QueuedSegment[] = [];
  let _processing = false;

  function processNext(): void {
    if (queue.length === 0) {
      _processing = false;
      return;
    }

    _processing = true;
    const segment = queue.shift()!;

    processFn(segment.audio, segment.sampleRate, segment.channels)
      .then(async (transcript) => {
        // Forward transcript to session
        if (transcript) {
          await onTranscript(transcript);
        }
        segment.resolve();
        processNext();
      })
      .catch((err) => {
        const error = err instanceof Error ? err : new Error(String(err));
        logger.error({ message: 'Audio segment processing failed', error });
        segment.reject(error);
        processNext();
      });
  }

  function enqueue(
    audio: Buffer,
    sampleRate: number,
    channels: number,
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      queue.push({ audio, sampleRate, channels, resolve, reject });

      logger.debug({
        message: 'Audio segment enqueued',
        queueLength: queue.length,
        processing: _processing,
      });

      if (!_processing) {
        processNext();
      }
    });
  }

  function clear(): void {
    // Reject all pending segments
    for (const segment of queue) {
      segment.reject(new Error('Queue cleared'));
    }
    queue.length = 0;
    _processing = false;
  }

  return {
    enqueue,
    clear,
    get processing() {
      return _processing;
    },
    get pending() {
      return queue.length;
    },
  };
}
