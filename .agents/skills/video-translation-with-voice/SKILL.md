---
name: video-translation
description: Use for AgenticDub video translation and dubbing pipeline tasks, especially default xAI OAuth STT/LLM/TTS runs, HITL review, subtitle timing, punctuation cleanup, TTS voice selection, original-audio-under-dub mixing, ASS subtitle burn-in, and end-to-end output validation.
license: MIT
compatibility: opencode
metadata:
  audience: developers
  workflow: ai-video-processing
---

## Default Pipeline

Use the AgenticDub xAI OAuth pipeline by default:

```
影片 -> 音軌分離 -> xAI OAuth STT -> 字幕切割 -> xAI OAuth LLM 翻譯 -> HITL review -> xAI OAuth TTS -> 原音小聲混音 -> ASS 字幕燒錄 -> 最終影片
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

Leave `tts_voice_code` empty unless the user explicitly requests a role/voice. The server should randomly choose one xAI voice from configured candidates.

## When To Use

Use this skill when working on:

- AgenticDub full video translation/dubbing runs.
- xAI OAuth STT, LLM, or TTS provider behavior.
- HITL review approval/rejection behavior.
- Subtitle punctuation cleanup, readable timing, or ASS burn-in quality.
- TTS timing, silence gaps, original-audio-under-dub mixing, or final media validation.

## Core Workflow

1. Start the local AgenticDub API server only when live validation is needed:

```bash
go run ./cmd/server/main.go
```

2. Submit a task with xAI profiles. For exact payloads and curl commands, read `references/xai-oauth-pipeline.md`.

3. Wait for HITL at `process_percent = 90`.

4. Inspect `tasks/<task_id>/review.txt` before approval.

5. Approve only after subtitle text is acceptable:

```bash
curl -s -X POST "http://127.0.0.1:8899/api/hitl/approve/<task_id>"
```

6. Wait for `process_percent = 100`.

7. Validate SRT, ASS, audio duration, final video metadata, and download URLs. For checklist, read `references/subtitle-quality-checklist.md`.

8. Stop the local test server after validation so port `8899` is not left occupied.

## Required HITL Behavior

- The pipeline pauses after translation and creates `review.txt`.
- The `字幕：` lines are the reviewer-facing target subtitles.
- Approval should clean punctuation and apply the final SRT to the TTS/burn-in source.
- Rejection should not continue to TTS.

## Important Quality Rules

- Target subtitles should not contain Chinese punctuation or inline punctuation such as `，。？！、；：·・･`.
- ASS text should not split protected tokens like `OpenCL`, `Linux`, or `30`.
- Readable subtitle timing defaults should target roughly `1.5s` minimum display and about `6` CJK chars/sec.
- TTS should align each sentence to its own subtitle interval; gaps between subtitles should be separate silence, not part of the prior sentence.
- Final dubbed video should preserve original audio quietly under the dubbed track.

## Key Files

- `internal/service/audio2subtitle.go` - STT, segmentation, translation, readable SRT timings.
- `internal/service/subtitle_service.go` - task orchestration and HITL wait/apply flow.
- `internal/agent/hitl/cleaner.go` - HITL punctuation cleanup.
- `internal/service/srt2speech.go` - TTS generation, duration adjustment, silence gaps, audio mixing.
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
