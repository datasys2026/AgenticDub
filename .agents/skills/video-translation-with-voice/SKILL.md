---
name: video-translation
description: Use for AgenticDub video translation and dubbing pipeline tasks, especially xAI OAuth STT/LLM/TTS runs, HITL review, glossary/name preservation, multi-speaker voice assignment, subtitle timing, punctuation cleanup, original-audio-under-dub mixing, ASS subtitle burn-in, Postiz/YouTube unlisted publishing, and end-to-end output validation.
license: MIT
metadata:
  audience: developers
  workflow: ai-video-processing
---

## Default Pipeline

Use the AgenticDub xAI OAuth pipeline by default:

```
影片 -> 音軌分離 -> xAI OAuth STT -> 字幕切割 -> xAI OAuth LLM 翻譯 -> 詞彙/人名 audit -> HITL review -> 多說話者 voice assignment -> xAI OAuth TTS -> 原音小聲混音 -> ASS 字幕燒錄 -> 最終影片
```

Default request profile fields:

```json
{
  "llm_profile": "grok",
  "stt_profile": "xai",
  "tts_profile": "xai",
  "tts_voice_code": ""
}
```

Leave `tts_voice_code` empty unless the user explicitly requests a role/voice. For single-speaker content, choose one voice for the whole run. For multi-speaker content, assign a stable voice per inferred speaker and persist the mapping.

## When To Use

Use this skill when working on:

- AgenticDub full video translation/dubbing runs.
- xAI OAuth STT, LLM, or TTS provider behavior.
- HITL review approval/rejection behavior.
- Domain glossary extraction, protected English names, and terminology replacement.
- Multi-speaker detection and deterministic per-speaker xAI voice assignment.
- Subtitle punctuation cleanup, readable timing, or ASS burn-in quality.
- TTS timing, silence gaps, original-audio-under-dub mixing, or final media validation.
- Postiz MCP upload, YouTube unlisted draft-first publishing, and release URL verification.

## Core Workflow

1. Start the local AgenticDub API server only when live validation is needed:

```bash
go run ./cmd/server/main.go
```

2. Submit a task with xAI profiles. For exact payloads and curl commands, read `references/xai-oauth-pipeline.md`.

3. Before HITL approval, apply any domain glossary and protected-name rules. For detailed workflow, read `references/glossary-hitl-workflow.md`.

4. Wait for HITL at `process_percent = 90`.

5. Inspect `tasks/<task_id>/review.txt` before approval.

6. Approve only after subtitle text is acceptable:

```bash
curl -s -X POST "http://127.0.0.1:8899/api/hitl/approve/<task_id>"
```

7. For TTS runs, assign voices before generating speech. If multiple speakers are present or likely, read `references/speaker-voice-assignment.md`.

8. Wait for `process_percent = 100`.

9. Validate SRT, ASS, audio duration, final video metadata, and download URLs. For checklist, read `references/subtitle-quality-checklist.md`. For aspect ratio, ffmpeg, fade-out, and final output naming rules, read `references/media-finalization.md`.

10. If publishing through Postiz/YouTube, read `references/postiz-youtube-publish.md`.

11. Stop the local test server after validation so port `8899` is not left occupied.

## Required HITL Behavior

- The pipeline pauses after translation and creates `review.txt`.
- The `字幕：` lines are the reviewer-facing target subtitles.
- Approval should clean punctuation and apply the final SRT to the TTS/burn-in source.
- Rejection should not continue to TTS.
- If glossary or protected-name fixes change final text after TTS, regenerate only affected cues, then rebuild timing, final audio, ASS, mixed video, and final output.

## Important Quality Rules

- Target subtitles should not contain Chinese punctuation or inline punctuation such as `，。？！、；：·・･`.
- ASS text should not split protected tokens like `OpenCL`, `Linux`, or `30`.
- Readable subtitle timing defaults should target roughly `1.5s` minimum display and about `6` CJK chars/sec.
- TTS should align each sentence to its own subtitle interval; gaps between subtitles should be separate silence, not part of the prior sentence.
- Final dubbed video should preserve original audio quietly under the dubbed track.
- Preserve original aspect ratio unless the user explicitly requests vertical output.
- Never randomize voice per cue. A speaker must keep the same voice across the whole output.

## Key Files

- `internal/service/audio2subtitle.go` - STT, segmentation, translation, readable SRT timings.
- `internal/service/subtitle_service.go` - task orchestration and HITL wait/apply flow.
- `internal/agent/hitl/cleaner.go` - HITL punctuation cleanup.
- `internal/service/srt2speech.go` - TTS generation, duration adjustment, silence gaps, audio mixing.
- `internal/service/speech_text.go` - TTS voice assignment persistence.
- `internal/service/srt_embed.go` - SRT-to-ASS conversion and subtitle burn-in.
- `internal/providers/*/xai_oauth.go` - xAI OAuth provider implementations.

## Validation Commands

Run static validation after code changes:

```bash
go test ./...
go vet ./...
go build -o /tmp/agenticdub-krillin-ai ./cmd/server
```

Use live validation only when OAuth credentials and network access are available.

## References

- For the xAI OAuth smoke-test workflow, read `references/xai-oauth-pipeline.md`.
- For subtitle, timing, and media QA checks, read `references/subtitle-quality-checklist.md`.
- For glossary extraction, protected English names, and HITL revision, read `references/glossary-hitl-workflow.md`.
- For multi-speaker detection and deterministic voice mapping, read `references/speaker-voice-assignment.md`.
- For ffmpeg/libass fallback, aspect-ratio preservation, final frame extension, fade-out, and filenames, read `references/media-finalization.md`.
- For Postiz MCP upload and YouTube unlisted draft-to-publish flow, read `references/postiz-youtube-publish.md`.
