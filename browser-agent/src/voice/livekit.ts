/**
 * LiveKit transport layer — Node.js server-side implementation.
 *
 * Uses @livekit/rtc-node for raw audio frame access (no browser APIs).
 * Architecture: persistent publish track + AudioStream receive.
 *
 * Responsibilities:
 *   - Connect/disconnect from LiveKit rooms
 *   - Publish audio via a persistent AudioSource (write frames continuously)
 *   - Receive raw PCM frames from remote participants via AudioStream
 *   - Emit type-safe transport events with raw payloads
 *   - Filter own audio to prevent feedback loops
 *
 * NOT responsible for: STT, TTS, interview logic, brain processing, VAD.
 *
 * ⚠️ This runs in Node.js (Docker), NOT in a browser.
 *    No AudioContext, no HTMLAudioElement, no requestAnimationFrame.
 */

import {
  Room,
  RoomEvent,
  RemoteAudioTrack,
  LocalAudioTrack,
  AudioSource,
  AudioFrame,
  AudioStream,
  TrackPublishOptions,
  TrackSource,
} from '@livekit/rtc-node';
import { EventEmitter } from 'events';
import { logger } from '../utils/logger.js';
import type {
  LiveKitTransport as ILiveKitTransport,
  LiveKitConfig,
  AudioChunk,
  TransportEventMap,
} from './types.js';

/** Default audio format for the transport's publish source. */
const DEFAULT_SAMPLE_RATE = 48000;
const DEFAULT_CHANNELS = 1;

/**
 * LiveKit transport implementation for Node.js.
 *
 * Uses @livekit/rtc-node for raw audio frame access.
 * Publishes one persistent audio track — TTS output is written via captureFrame().
 * Receives incoming audio as raw Int16Array PCM via AudioStream events.
 *
 * Handler signatures match TransportEventMap — raw payloads, not wrapped events:
 *   transport.on('audioReceived', (audio, participantId) => { ... })
 *   transport.on('connected', (roomName) => { ... })
 */
export class LiveKitTransportImpl implements ILiveKitTransport {
  readonly name = 'livekit';

  private room: Room | null = null;
  private emitter = new EventEmitter();
  private _connected = false;
  private _connecting = false;
  private _disconnecting = false;
  private _localIdentity: string | null = null;

  // Persistent publish infrastructure — created once, lives for the session.
  private publishSource: AudioSource | null = null;
  private publishTrack: LocalAudioTrack | null = null;

  // Active receive streams — keyed by participant identity.
  private activeStreams = new Map<string, AudioStream>();

  get connected(): boolean {
    return this._connected;
  }

  /**
   * Connect to a LiveKit room.
   * Creates Room, wires events, establishes connection.
   * Automatically initializes the persistent publish track after connecting.
   */
  async connect(config: LiveKitConfig, roomName: string, token: string): Promise<void> {
    if (this._connected || this._connecting) {
      logger.warn({ message: 'LiveKit already connected or connecting, ignoring', roomName });
      return;
    }

    this._connecting = true;

    logger.info({ message: 'Connecting to LiveKit room', roomName });

    const room = new Room();

    // Wire events BEFORE connect — prevents missing immediate events after connect resolves
    this.room = room;
    this.wireRoomEvents();

    try {
      await room.connect(config.url, token, {
        autoSubscribe: true,
        dynacast: true,
      });

      this._connected = true;

      // Capture local identity for self-audio filtering
      if (room.localParticipant) {
        this._localIdentity = room.localParticipant.identity;
      }

      // Initialize persistent publish track
      await this.initPublish();

      logger.info({ message: 'Connected to LiveKit room', roomName });
      this.emitConnected(roomName);
    } catch (err) {
      // Clean up the room — remove any handlers we wired
      room.removeAllListeners();
      this.room = null;
      this._localIdentity = null;
      this._connected = false; // Reset — transport is not usable after failed initPublish

      logger.error({ message: 'Failed to connect to LiveKit room', err, roomName });
      this.emitError(err instanceof Error ? err : new Error(String(err)));
      throw err;
    } finally {
      this._connecting = false;
    }
  }

  /**
   * Disconnect from the current room.
   * Stops all tracks, cleans up streams, closes publish source.
   */
  async disconnect(): Promise<void> {
    if (!this._connected || !this.room) {
      logger.warn({ message: 'LiveKit disconnect called but not connected' });
      return;
    }

    this._disconnecting = true;

    const roomName = this.room.name;
    logger.info({ message: 'Disconnecting from LiveKit room', roomName });

    // Close all active receive streams
    try {
      this.closeAllStreams();
    } catch (err) {
      logger.error({ message: 'Error closing audio streams', err });
    }

    // Close persistent publish infrastructure
    if (this.publishTrack && this.room.localParticipant) {
      try {
        await this.room.localParticipant.unpublishTrack(this.publishTrack.sid);
      } catch (err) {
        logger.error({ message: 'Error unpublishing track', err });
      }
    }
    this.publishTrack = null;
    this.publishSource = null;

    try {
      await this.room.disconnect();
    } catch (err) {
      logger.error({ message: 'Error during LiveKit disconnect', err, roomName });
    }

    this.room = null;
    this._localIdentity = null;
    this._connected = false;
    this._disconnecting = false;
    this.emitDisconnected('manual disconnect');
  }

  /**
   * Publish audio to the room via the persistent track.
   *
   * Converts AudioChunk (Buffer | ArrayBuffer) to AudioFrame and writes to AudioSource.
   * The track is published once at init — this just pushes frames into the pipeline.
   *
   * ⚠️ Sample rate must match the AudioSource (48kHz).
   *    If TTS outputs 16k/24k, resample before calling publishAudio().
   *    Passing mismatched rates causes pitch/speed distortion.
   *
   * Fire-and-forget: errors are emitted via 'error' event, not thrown.
   */
  async publishAudio(audio: AudioChunk): Promise<void> {
    if (!this._connected || this._disconnecting || !this.publishSource) {
      logger.warn({ message: 'Cannot publish audio: not connected, disconnecting, or publish not initialized' });
      return;
    }

    // Validate sample rate — mismatch causes pitch/speed distortion
    if (audio.sampleRate !== DEFAULT_SAMPLE_RATE) {
      logger.error({
        message: 'Sample rate mismatch — audio will be distorted',
        expected: DEFAULT_SAMPLE_RATE,
        received: audio.sampleRate,
      });
      this.emitError(new Error(
        `Sample rate mismatch: expected ${DEFAULT_SAMPLE_RATE}Hz, got ${audio.sampleRate}Hz. ` +
        `Resample audio before calling publishAudio().`,
      ));
      return;
    }

    // Validate channel count
    if (!audio.channels || audio.channels < 1) {
      logger.error({ message: 'Invalid channel count', channels: audio.channels });
      this.emitError(new Error(
        `Invalid channels: expected a positive integer, got ${audio.channels}.`,
      ));
      return;
    }

    try {
      // Normalize Buffer | ArrayBuffer to Int16Array
      const pcmData: Int16Array = audio.data instanceof ArrayBuffer
        ? new Int16Array(audio.data)
        : new Int16Array(audio.data.buffer, audio.data.byteOffset, audio.data.byteLength / 2);

      // Validate PCM data length
      if (pcmData.length === 0) {
        logger.warn({ message: 'Empty audio frame, skipping' });
        return;
      }
      if (pcmData.length % audio.channels !== 0) {
        this.emitError(new Error(
          `PCM data length (${pcmData.length}) is not evenly divisible by channels (${audio.channels}).`,
        ));
        return;
      }

      const samplesPerChannel = Math.floor(pcmData.length / audio.channels);

      const frame = new AudioFrame(
        pcmData,
        audio.sampleRate,
        audio.channels,
        samplesPerChannel,
      );

      await this.publishSource.captureFrame(frame);
    } catch (err) {
      logger.error({ message: 'Failed to publish audio', err });
      this.emitError(err instanceof Error ? err : new Error(String(err)));
    }
  }

  /**
   * Subscribe to a specific transport event with type-safe callback.
   * Handlers receive raw payloads — not wrapped event objects.
   *
   * @example
   * transport.on('audioReceived', (audio, participantId) => { ... })
   * transport.on('connected', (roomName) => { ... })
   */
  on<K extends keyof TransportEventMap>(event: K, handler: TransportEventMap[K]): void {
    this.emitter.on(event, handler);
  }

  /** Remove event handler. */
  off<K extends keyof TransportEventMap>(event: K, handler: TransportEventMap[K]): void {
    this.emitter.off(event, handler);
  }

  /**
   * Clean up all resources.
   * Must be called when the transport is no longer needed.
   */
  async destroy(): Promise<void> {
    logger.info({ message: 'Destroying LiveKit transport' });

    await this.disconnect();
    this.emitter.removeAllListeners();
    // Note: dispose() is NOT called here — it's a global FFI cleanup.
    // Call once at process exit if needed: process.on('exit', () => dispose());
  }

  // ----- Private: Publish -----

  /**
   * Initialize the persistent publish track.
   * Creates one AudioSource + LocalAudioTrack, publishes once.
   * Called automatically at the end of connect().
   */
  private async initPublish(): Promise<void> {
    if (!this.room) {
      throw new Error('Cannot init publish: no room');
    }
    if (this.publishSource) {
      return; // Already initialized
    }

    if (!this.room.localParticipant) {
      throw new Error('Cannot init publish: no local participant');
    }

    const source = new AudioSource(DEFAULT_SAMPLE_RATE, DEFAULT_CHANNELS);
    const track = LocalAudioTrack.createAudioTrack('agent-audio', source);

    const options = new TrackPublishOptions();
    options.source = TrackSource.SOURCE_MICROPHONE;

    try {
      await this.room.localParticipant.publishTrack(track, options);
    } catch (err) {
      // Note: track and source have no explicit close() — native resources freed by GC
      throw err;
    }

    // Only assign after successful publish
    this.publishSource = source;
    this.publishTrack = track;

    logger.info({ message: 'Published persistent audio track' });
  }

  // ----- Private: Room Events -----

  /** Wire LiveKit room events to our type-safe event emitter. */
  private wireRoomEvents(): void {
    if (!this.room) return;

    this.room.on(RoomEvent.Connected, () => {
      // Initial connection — connect() already emitted, but handle reconnection too.
      if (this._connected) return;
      logger.info({ message: 'LiveKit room connected' });
      this._connected = true;
      this.emitConnected(this.room?.name ?? 'unknown');
    });

    this.room.on(RoomEvent.Disconnected, (reason?: string) => {
      // If disconnect() was initiated locally, it will emit its own event.
      if (this._disconnecting) return;
      logger.info({ message: 'LiveKit room disconnected', reason });
      this._connected = false;
      this.emitDisconnected(reason);
    });

    this.room.on(RoomEvent.Reconnected, () => {
      logger.info({ message: 'LiveKit room reconnected' });
      this._connected = true;
      this.emitConnected(this.room?.name ?? 'unknown');
    });

    this.room.on(RoomEvent.TrackSubscribed, (track, _publication, participant) => {
      if (track instanceof RemoteAudioTrack) {
        const identity = participant.identity;
        logger.debug({ message: 'Received audio track', participant: identity });
        this.handleIncomingAudio(track, identity);
      }
    });

    this.room.on(RoomEvent.ParticipantConnected, (participant) => {
      logger.debug({ message: 'Participant joined', identity: participant.identity });
      this.emitParticipantJoined(participant.identity, 'audio');
    });

    this.room.on(RoomEvent.ParticipantDisconnected, (participant) => {
      const identity = participant.identity;
      logger.debug({ message: 'Participant left', identity });
      this.closeStream(identity);
      this.emitParticipantLeft(identity);
    });

    // Clean up streams when tracks are unpublished
    this.room.on(RoomEvent.TrackUnsubscribed, (track, _publication, participant) => {
      if (track instanceof RemoteAudioTrack) {
        logger.debug({ message: 'Audio track unsubscribed', participant: participant.identity });
        this.closeStream(participant.identity);
      }
    });
  }

  // ----- Private: Receive -----

  /**
   * Handle incoming audio from a remote participant.
   * Creates an AudioStream and listens for frameReceived events.
   * Filters own audio to prevent feedback loops.
   */
  private handleIncomingAudio(track: RemoteAudioTrack, participantIdentity: string): void {
    // Self-audio filter — prevent infinite feedback loop
    // Try to re-capture identity if it wasn't available at connect time
    if (!this._localIdentity && this.room?.localParticipant?.identity) {
      this._localIdentity = this.room.localParticipant.identity;
    }

    if (this._localIdentity && participantIdentity === this._localIdentity) {
      logger.debug({ message: 'Ignoring own audio track', participant: participantIdentity });
      return;
    }

    if (!this._localIdentity) {
      logger.warn({ message: 'Cannot filter self-audio: local identity unknown', participant: participantIdentity });
    }

    // Close existing stream for this participant if any
    this.closeStream(participantIdentity);

    const stream = new AudioStream(track);

    // Store for cleanup
    this.activeStreams.set(participantIdentity, stream);

    // Listen for raw PCM frames — EventEmitter pattern, not ReadableStream.
    // Buffer.from() copy is intentional — AudioStream reuses the underlying buffer
    // for the next frame, so we must snapshot it for async consumers.
    //
    // Backpressure: @livekit/rtc-node's native AudioStream handles its own buffering
    // internally. We don't need an application-level queue. The native layer will
    // drop frames if the consumer can't keep up — we just forward what we receive.

    stream.on('frameReceived', (event: { frame: AudioFrame }) => {
      const frame = event.frame;

      // Snapshot the frame data — AudioStream reuses the underlying buffer
      const pcmBuffer = Buffer.from(
        frame.data.buffer,
        frame.data.byteOffset,
        frame.data.byteLength,
      );

      // Emit raw payload — handler receives (audio, participantId)
      this.emitAudioReceived(
        {
          data: pcmBuffer,
          sampleRate: frame.sampleRate,
          channels: frame.channels,
        },
        participantIdentity,
      );
    });
  }

  /** Close a single participant's audio stream and remove all listeners. */
  private closeStream(participantIdentity: string): void {
    const stream = this.activeStreams.get(participantIdentity);
    if (!stream) return;

    stream.removeAllListeners();
    stream.close();
    this.activeStreams.delete(participantIdentity);
  }

  /** Close all active audio streams. */
  private closeAllStreams(): void {
    for (const [identity, stream] of this.activeStreams) {
      stream.removeAllListeners();
      stream.close();
    }
    this.activeStreams.clear();
  }

  // ----- Private: Emit (raw payloads) -----

  /**
   * Emit with raw payloads — handlers receive individual arguments, not wrapped events.
   * This matches the TransportEventMap contract:
   *   transport.on('audioReceived', (audio, participantId) => { ... })
   */
  private emitConnected(roomName: string): void {
    this.emitter.emit('connected', roomName);
  }

  private emitDisconnected(reason?: string): void {
    this.emitter.emit('disconnected', reason);
  }

  private emitAudioReceived(audio: AudioChunk, participantId: string): void {
    // Safe emit: don't throw if no listener attached
    if (this.emitter.listenerCount('audioReceived') === 0) {
      return;
    }
    this.emitter.emit('audioReceived', audio, participantId);
  }

  private emitParticipantJoined(identity: string, kind: string): void {
    this.emitter.emit('participantJoined', identity, kind);
  }

  private emitParticipantLeft(identity: string, reason?: string): void {
    this.emitter.emit('participantLeft', identity, reason);
  }

  private emitError(error: Error): void {
    // Safe emit: don't throw if no error listener attached
    if (this.emitter.listenerCount('error') === 0) {
      logger.error({ message: 'Unhandled transport error', error });
      return;
    }
    this.emitter.emit('error', error);
  }
}
