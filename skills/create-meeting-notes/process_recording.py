# /// script
# requires-python = ">=3.12"
# dependencies = ["openai>=1.0"]
# ///
"""Process a video recording: extract audio, transcribe via Groq Whisper
with timestamps, and extract candidate screenshot frames.
"""
from __future__ import annotations

import asyncio
import json
import os
import shutil
import subprocess
import sys
import tempfile
from datetime import datetime
from pathlib import Path

from openai import AsyncOpenAI, OpenAIError

GROQ_BASE_URL = "https://api.groq.com/openai/v1"
MODEL = "whisper-large-v3"
MAX_FILE_BYTES = 24 * 1024 * 1024  # 25MB limit with safety margin
DEFAULT_CHUNK_MINUTES = 10
MAX_CONCURRENT = 4
def frame_interval(duration_seconds: float) -> int:
    """Adaptive frame interval to keep candidate count around ~60."""
    minutes = duration_seconds / 60
    if minutes <= 15:
        return 15
    elif minutes <= 30:
        return 30
    else:
        return 45


# ── ffmpeg helpers ──────────────────────────────────────────────────

def get_duration(path: str) -> float:
    result = subprocess.run(
        ["ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", path],
        capture_output=True, text=True,
    )
    if result.returncode != 0:
        print(f"ffprobe failed: {result.stderr}", file=sys.stderr)
        sys.exit(1)
    return float(json.loads(result.stdout)["format"]["duration"])


def get_resolution(path: str) -> str:
    result = subprocess.run(
        [
            "ffprobe", "-v", "quiet", "-select_streams", "v:0",
            "-show_entries", "stream=width,height",
            "-of", "csv=s=x:p=0", path,
        ],
        capture_output=True, text=True, check=True,
    )
    return result.stdout.strip()


def format_duration(seconds: float) -> str:
    m = int(seconds) // 60
    s = int(seconds) % 60
    return f"{m}:{s:02d}"


def format_timestamp(seconds: float) -> str:
    m = int(seconds) // 60
    s = int(seconds) % 60
    return f"{m}:{s:02d}"


def extract_audio(input_path: str, output_path: str) -> None:
    subprocess.run(
        [
            "ffmpeg", "-y", "-i", input_path,
            "-vn", "-acodec", "libmp3lame",
            "-ab", "64k", "-ar", "16000", "-ac", "1",
            output_path,
        ],
        capture_output=True, check=True,
    )


def split_audio(input_path: str, output_dir: str, chunk_seconds: int) -> list[str]:
    duration = get_duration(input_path)
    chunks = []
    start = 0.0
    idx = 0
    while start < duration:
        chunk_path = os.path.join(output_dir, f"chunk_{idx:04d}.mp3")
        subprocess.run(
            [
                "ffmpeg", "-y", "-ss", str(start), "-i", input_path,
                "-t", str(chunk_seconds),
                "-acodec", "libmp3lame", "-ab", "64k", "-ar", "16000", "-ac", "1",
                chunk_path,
            ],
            capture_output=True, check=True,
        )
        if os.path.getsize(chunk_path) > 0:
            chunks.append(chunk_path)
        start += chunk_seconds
        idx += 1
    return chunks


# ── transcription ───────────────────────────────────────────────────

async def transcribe_chunk(
    client: AsyncOpenAI, chunk_path: str, idx: int, total: int, offset_seconds: float,
) -> dict:
    print(f"  Transcribing chunk {idx + 1}/{total}...")
    with open(chunk_path, "rb") as f:
        response = await client.audio.transcriptions.create(
            model=MODEL,
            file=f,
            response_format="verbose_json",
            timestamp_granularities=["segment"],
        )

    segments = []
    if hasattr(response, "segments") and response.segments:
        for seg in response.segments:
            segments.append({
                "start": seg.start + offset_seconds,
                "end": seg.end + offset_seconds,
                "text": seg.text,
            })

    return {
        "chunk_index": idx,
        "text": response.text,
        "segments": segments,
    }


async def transcribe_audio(api_key: str, audio_path: str, tmpdir: str) -> dict:
    audio_size = os.path.getsize(audio_path)
    duration = get_duration(audio_path)
    print(f"Audio: {audio_size / 1024 / 1024:.1f}MB, {duration / 60:.1f} min")

    async with AsyncOpenAI(base_url=GROQ_BASE_URL, api_key=api_key, timeout=120) as client:
        if audio_size <= MAX_FILE_BYTES:
            print("Transcribing 1 chunk(s) via Groq Whisper...")
            result = await transcribe_chunk(client, audio_path, 0, 1, 0.0)
            return {"text": result["text"], "segments": result["segments"]}

        # Split into chunks
        chunk_seconds = DEFAULT_CHUNK_MINUTES * 60
        chunks = split_audio(audio_path, tmpdir, chunk_seconds)
        print(f"Transcribing {len(chunks)} chunk(s) via Groq Whisper...")

        # Calculate offsets
        offsets = []
        cumulative = 0.0
        for chunk_path in chunks:
            offsets.append(cumulative)
            cumulative += get_duration(chunk_path)

        # Transcribe concurrently
        semaphore = asyncio.Semaphore(MAX_CONCURRENT)
        results: list[dict | None] = [None] * len(chunks)

        async def _do(idx: int, path: str) -> None:
            async with semaphore:
                try:
                    results[idx] = await transcribe_chunk(
                        client, path, idx, len(chunks), offsets[idx],
                    )
                except OpenAIError as e:
                    print(f"  Error on chunk {idx + 1}: {e}")
                    results[idx] = {"chunk_index": idx, "text": f"[error: {e}]", "segments": []}

        await asyncio.gather(*[_do(i, p) for i, p in enumerate(chunks)])

    # Merge
    valid = [r for r in results if r is not None]
    valid.sort(key=lambda r: r["chunk_index"])
    full_text = " ".join(r["text"] for r in valid)
    all_segments = []
    for r in valid:
        all_segments.extend(r["segments"])
    all_segments.sort(key=lambda s: s["start"])

    return {"text": full_text, "segments": all_segments}


# ── frame extraction ────────────────────────────────────────────────

def extract_frames(mp4_path: str, output_dir: str, interval: int, duration: float) -> list[str]:
    candidates_dir = os.path.join(output_dir, "_candidates")
    os.makedirs(candidates_dir, exist_ok=True)

    num_frames = int(duration // interval) + 1
    print(f"Extracting ~{num_frames} candidate frames (every {interval}s)...")

    frames = []
    for i in range(num_frames):
        timestamp = i * interval
        if timestamp > duration:
            break
        m = timestamp // 60
        s = timestamp % 60
        filename = f"candidate_{m:02d}m{s:02d}s.png"
        output_path = os.path.join(candidates_dir, filename)

        subprocess.run(
            [
                "ffmpeg", "-y", "-ss", str(timestamp), "-i", mp4_path,
                "-frames:v", "1", "-q:v", "2", output_path,
            ],
            capture_output=True, check=True,
        )
        frames.append(filename)

    print(f"  Extracted {len(frames)} candidate frames")
    return frames


# ── output ──────────────────────────────────────────────────────────

def write_transcript(segments: list[dict], output_path: str) -> int:
    lines = []
    for seg in segments:
        ts = format_timestamp(seg["start"])
        text = seg["text"].strip()
        if text:
            lines.append(f"[{ts}] {text}")
    content = "\n".join(lines)
    Path(output_path).write_text(content)
    return len(content)


# ── main ────────────────────────────────────────────────────────────

def main() -> None:
    if len(sys.argv) < 2:
        print("Usage: process_recording.py <path-to-mp4>")
        sys.exit(1)

    mp4_path = os.path.abspath(sys.argv[1])
    if not os.path.exists(mp4_path):
        print(f"Error: File not found: {mp4_path}")
        sys.exit(1)

    if not shutil.which("ffmpeg") or not shutil.which("ffprobe"):
        print("Error: ffmpeg and ffprobe are required")
        sys.exit(1)

    api_key = os.environ.get("GROQ_API_KEY")
    if not api_key:
        print("Error: GROQ_API_KEY not set")
        sys.exit(1)

    # Output directory next to the MP4
    basename = Path(mp4_path).stem
    output_dir = os.path.join(os.path.dirname(mp4_path), f"meeting-notes-{basename}")
    os.makedirs(output_dir, exist_ok=True)
    print(f"Output directory: {output_dir}")

    # Video info
    duration = get_duration(mp4_path)
    resolution = get_resolution(mp4_path)
    print(f"Duration: {format_duration(duration)} | Resolution: {resolution}")

    # Extract audio to temp dir, transcribe, clean up
    tmpdir = tempfile.mkdtemp(prefix="meeting_notes_")
    try:
        audio_path = os.path.join(tmpdir, "audio.mp3")
        print(f"Extracting audio → {os.path.join(output_dir, 'audio.mp3')}")
        extract_audio(mp4_path, audio_path)

        transcript = asyncio.run(transcribe_audio(api_key, audio_path, tmpdir))
    finally:
        shutil.rmtree(tmpdir, ignore_errors=True)

    # Write transcript
    transcript_path = os.path.join(output_dir, "transcript.txt")
    transcript_chars = write_transcript(transcript["segments"], transcript_path)
    print(f"Transcript written → {transcript_path}")

    # Extract frames
    interval = frame_interval(duration)
    frames = extract_frames(mp4_path, output_dir, interval, duration)

    # Write metadata
    metadata = {
        "source_file": os.path.basename(mp4_path),
        "source_path": mp4_path,
        "duration_seconds": duration,
        "duration_formatted": format_duration(duration),
        "resolution": resolution,
        "date": datetime.now().isoformat(),
        "frame_interval_seconds": interval,
        "candidate_frames": frames,
        "transcript_segments": len(transcript["segments"]),
    }
    metadata_path = os.path.join(output_dir, "metadata.json")
    with open(metadata_path, "w") as f:
        json.dump(metadata, f, indent=2)
    print(f"Metadata written → {metadata_path}")

    print(f"\nDone! Output: {output_dir}")
    print(f"  - transcript.txt ({transcript_chars} chars)")
    print(f"  - _candidates/ ({len(frames)} frames)")
    print(f"  - metadata.json")


if __name__ == "__main__":
    main()
