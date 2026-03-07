# Local Transcript Design

## Context

Create a reusable `local-transcript` skill for turning a specified local video file into a final `.txt` transcript. The workflow must stay local-first, avoid cloud APIs, and infer the transcript language from the video's spoken language. The final output should be a cleaned transcript only: natural paragraphs, corrected common ASR mistakes, and simplified Chinese for Chinese speech.

## Approach

Use `ffmpeg` to extract mono 16 kHz WAV audio from the input video, then use `whisper.cpp` via `whisper.cpp-cli` for offline transcription. Prefer Whisper language auto-detection by default, and post-process the raw transcript into a final cleaned `.txt`.

## Architecture

The skill consists of:

- `SKILL.md` with usage rules, dependency gates, and output contract
- `scripts/local_transcript.py` as the deterministic workflow entrypoint
- `agents/openai.yaml` for skill metadata

The script handles probing, model download, audio extraction, transcription, cleanup, and final file writing.

## Components

- Input handling: validate local video path and optional output path
- Dependency handling: verify `ffmpeg`, `uvx`, and model availability
- Model bootstrap: download `ggml-base.bin` if missing
- Audio extraction: produce temporary WAV from video
- Transcription: call `whisper-cpp` with auto language detection
- Cleanup: strip timestamps, detect Chinese vs non-Chinese output, simplify Chinese, fix common errors, merge natural paragraphs
- Output: write one final `.txt` file, do not keep raw transcript as deliverable

## Error Handling

- Missing input file: stop with a clear path-specific error
- Missing `ffmpeg` or `uvx`: stop with install/verify guidance
- Model download failure: stop with exact failed step
- Transcription failure: surface subprocess stderr summary
- Empty transcript: fail explicitly rather than writing a misleading blank file

## Testing

- Validate the skill folder with `quick_validate.py`
- Run a real smoke test against the existing sample video in `~/Downloads/yt-dlp`
- Confirm output file exists, is non-empty, and is paragraphized
