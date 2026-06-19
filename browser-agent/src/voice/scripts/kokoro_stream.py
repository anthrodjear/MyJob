#!/usr/bin/env python3
"""
Kokoro TTS daemon — long-lived process for Node.js voice provider.

Protocol:
  - Accepts config via CLI arguments: --model-path, --voices-path
  - Reads synthesis requests from stdin as length-prefixed JSON
  - Writes audio responses to stdout as length-prefixed JSON
  - Stays resident for all subsequent calls (model loaded once)

Wire format: [4-byte LE uint32 length][JSON payload]

Request (stdin):
  { "voice": "af_nicole", "speed": 1.0, "lang": "en-us", "text": "Hello" }

Response (stdout), streamed per chunk:
  { "type": "audio", "data": "<base64 PCM16>", "sample_rate": 24000 }
  ...
  { "type": "done" }

Usage:
  python3 kokoro_stream.py --model-path /path/to/model.onnx --voices-path /path/to/voices.bin
"""

import sys
import json
import struct
import asyncio
import base64
import argparse


def read_message(stream):
    """Read a length-prefixed JSON message from a binary stream."""
    length_bytes = stream.read(4)
    if not length_bytes or len(length_bytes) < 4:
        return None
    length = struct.unpack('<I', length_bytes)[0]
    if length == 0:
        return None
    data = stream.read(length)
    if not data or len(data) < length:
        return None
    return json.loads(data.decode('utf-8'))


def write_message(stream, obj):
    """Write a length-prefixed JSON message to a binary stream."""
    payload = json.dumps(obj).encode('utf-8')
    stream.write(struct.pack('<I', len(payload)))
    stream.write(payload)
    stream.flush()


async def main():
    parser = argparse.ArgumentParser(description='Kokoro TTS daemon')
    parser.add_argument('--model-path', required=True, help='Path to Kokoro ONNX model file')
    parser.add_argument('--voices-path', required=True, help='Path to voices binary file')
    args = parser.parse_args()

    import numpy as np
    from kokoro_onnx import Kokoro

    # Load model once at startup — this is the expensive operation
    try:
        kokoro = Kokoro(args.model_path, args.voices_path)
        write_message(sys.stdout.buffer, {"type": "ready"})
    except Exception as e:
        write_message(sys.stdout.buffer, {"type": "error", "message": str(e)})
        return

    # Process synthesis requests in a loop
    while True:
        request = read_message(sys.stdin.buffer)
        if request is None:
            break

        text = str(request.get('text', ''))
        if not text.strip():
            write_message(sys.stdout.buffer, {"type": "done"})
            continue

        voice = request.get('voice', 'af_nicole')
        speed = float(request.get('speed', 1.0))
        lang = request.get('lang', 'en-us')

        try:
            stream = kokoro.create_stream(text, voice=voice, speed=speed, lang=lang)
            async for samples, sample_rate in stream:
                pcm_data = (np.clip(samples, -1.0, 1.0) * 32767).astype(np.int16).tobytes()
                write_message(sys.stdout.buffer, {
                    "type": "audio",
                    "data": base64.b64encode(pcm_data).decode('ascii'),
                    "sample_rate": sample_rate,
                })
            write_message(sys.stdout.buffer, {"type": "done"})
        except Exception as e:
            write_message(sys.stdout.buffer, {"type": "error", "message": str(e)})


if __name__ == '__main__':
    asyncio.run(main())
