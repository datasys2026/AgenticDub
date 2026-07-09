---
name: video-translation
description: AI video translation pipeline - video to dubbed video with TTS and burned-in subtitles
license: MIT
compatibility: opencode
metadata:
  audience: developers
  workflow: ai-video-processing
---

## What I do

Translate videos with the full pipeline:
```
影片 → 音軌分離 → STT → 字幕切割 → LLM 翻譯 → [HITL 審核] → TTS 合成 → 字幕燒錄 → 合併影片
```

**HITL 審核流程：**
1. 翻譯完成後 → 生成 `review.txt` 等待審核
2. 人員編輯 `字幕：` 行（可修改內容）
3. 核准 → 清理標點 → 繼續 TTS
4. 否決 → 可重新翻譯

## Endpoints (aiark.com.tw)

| 服務 | 端點 | 模型 |
|------|------|------|
| **STT** | `https://aiark.com.tw/v1/audio/transcriptions` | `faster-whisper-large-v3-fp16` |
| **LLM** | `https://aiark.com.tw/v1/chat/completions` | `aiark/qwen36-35b-iq3` |
| **TTS** | `https://aiark.com.tw/tts/v1/audio/speech` | `aiark/qwen3-tts-0.6b-customvoice` |

API Key: `datasys2026`

## Important Config Notes

### 1. Language Code Mapping
API receives Chinese names (`繁體中文`, `簡體中文`) but internal code uses `zh_tw`, `zh_cn`.
- `繁體中文` → `zh_tw`
- `簡體中文` → `zh_cn`

When translating, prompt becomes: `Translate to 繁體中文` (via `GetStandardLanguageName(zh_tw)`)

### 2. TTS Voice
Default voice is `Ryan` (not `paul-chen-zh-tw-v1` which is the registered clone voice).

TTS response is JSON with `file` field, not direct audio:
```json
{"file":"/audio/xxx.wav","model":"...","sample_rate":24000,"voice":"Ryan"}
```
Need to download audio from `/tts/audio/{filename}` (replace `/v1` in base URL).

### 3. ffmpeg with libass (CRITICAL)
System `ffmpeg` does NOT have libass. Must use `ffmpeg-full`:
```bash
brew install ffmpeg-full
```

Path detection order in `checker.go`:
1. Check `/opt/homebrew/opt/ffmpeg-full/bin/ffmpeg` first (has libass)
2. Fall back to `ffmpeg` in PATH (no libass)
3. Download if neither found

### 4. TTS Concurrency
Set `maxConcurrency := 1` to avoid GPU contention on aiark server.

### 5. Subtitle Embedding
- Vertical video (9:16 Shorts): use `embed_subtitle_video_type = "vertical"`
- Horizontal video: use `embed_subtitle_video_type = "horizontal"`
- If video is vertical but `horizontal` is specified, subtitle burning is SKIPPED

## Audio Timing Adjustment Strategy

TTS synthesis duration often differs from original subtitle duration. Solution in `srt2speech.go::adjustAudioDuration`:

**When TTS < Original Duration:**
- Keep TTS at original speed (natural speech)
- Add silence gap at front (30%) and back (70%)
- Gap = `original_duration - tts_duration`
- Produces natural conversational pauses

**When TTS > Original Duration:**
- Speed up using ffmpeg `atempo` filter
- Speed factor = `tts_duration / original_duration`

**Implementation Details:**
- Silence files generated at sample rate 24000 Hz (matches TTS output)
- Concat demuxer used for joining silence + TTS + silence
- Concat file uses relative filenames (not full paths)
- Final audio duration matches original video duration

**Duration Log Format:**
```
[id] 原文時間=5.844s | 翻譯=[讓我跟你談...] | TTS=4.160s | 調整=gap(0.505+1.179) | 最終=5.844s
```

## API Request Format

```json
POST /api/capability/subtitleTask
{
  "url": "local:/path/to/video.mp4",
  "origin_lang": "en",
  "target_lang": "繁體中文",
  "bilingual": 0,
  "translation_subtitle_pos": 0,
  "modal_filter": 0,
  "tts": 1,
  "tts_voice_code": "",
  "language": "zh",
  "embed_subtitle_video_type": "vertical"
}
```

## HITL Review System (INTEGRATED)

### Workflow
1. Translation completed → Generate `review.txt`
2. Task status = `pending_review` (process_percent = 90)
3. Poll `GET /api/hitl/status/<task_id>` until approved
4. Call `POST /api/hitl/approve/<task_id>` to continue TTS
5. Or call `POST /api/hitl/reject/<task_id>` with reason to abort

### Review.txt Format
```
【第 1 句】 00:00:12,000 --> 00:00:15,500
原文：Hello world, how are you?
字幕：你好世界 你好嗎

【第 2 句】 00:00:15,500 --> 00:00:18,200
原文：I'm fine, thank you.
字幕：我很好 謝謝你
```

### 標點清理
核准時所有標點符號會被清理成空格

### HITL API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/hitl/review/<task_id>` | Get review.txt content |
| POST | `/api/hitl/approve/<task_id>` | Approve and continue TTS |
| POST | `/api/hitl/reject/<task_id>` | Reject with reason |
| GET | `/api/hitl/status/<task_id>` | Get review status |

### Example Workflow
```bash
# Start task (will pause at 90% for review)
curl -s http://127.0.0.1:8899/api/capability/subtitleTask -X POST ... | jq '.data.task_id'

# Poll until status shows pending_review
curl -s "http://127.0.0.1:8899/api/hitl/status/<task_id>"

# Get review content
curl -s "http://127.0.0.1:8899/api/hitl/review/<task_id>"

# Approve and continue
curl -s -X POST "http://127.0.0.1:8899/api/hitl/approve/<task_id>"

# Or reject
curl -s -X POST "http://127.0.0.1:8899/api/hitl/reject/<task_id>" -d '{"reason":"翻譯錯誤"}'
```

### Files
- `internal/agent/hitl/entity.go` - ReviewDocument, Segment 實體
- `internal/agent/hitl/parser.go` - TxtParser (parse/generate review.txt)
- `internal/agent/hitl/merger.go` - SRTMerger (合併編輯回 SRT)
- `internal/agent/hitl/service.go` - ReviewService (workflow)
- `internal/agent/hitl/cleaner.go` - CleanPunctuation (清理標點)
- `internal/service/subtitle_service.go` - Integration with waitForReview loop
- `internal/handler/hitl_review.go` - HITL API handlers
- `internal/router/router.go` - HITL API routes

## Key Files

- `internal/service/audio2subtitle.go` - STT + translation pipeline
- `internal/service/srt2speech.go` - TTS synthesis + timing adjustment
- `internal/service/srt_embed.go` - Subtitle burning
- `internal/service/subtitle_service.go` - Task orchestration
- `internal/deps/checker.go` - ffmpeg path detection (ffmpeg-full first)
- `pkg/openai/openai.go` - OpenAI-compatible client (TTS)

## When to use

Use this skill when:
- Translating English videos to Chinese with dubbed audio
- Burning bilingual subtitles into video
- Processing videos through the full STT→Translate→Review→TTS→Merge pipeline

**Note:** HITL review system is **INTEGRATED** - tasks will pause at 90% for review before TTS.

## Common Issues

1. **Subs are Simplified instead of Traditional**: Check `target_lang` is `繁體中文` not `簡體中文`
2. **TTS returns 404 on audio download**: URL should be `/tts/audio/xxx.wav` not `/tts/v1/audio/xxx.wav`
3. **ffmpeg subtitles filter not found**: Need ffmpeg-full with libass, check `/opt/homebrew/opt/ffmpeg-full/bin/ffmpeg`
4. **GPU busy on TTS**: Reduce concurrency to 1
5. **Vertical video with horizontal embed**: Subtitle burning is skipped - always match `embed_subtitle_video_type` to actual video orientation

## Polling Task Status

After creating a task, poll every 30 seconds to check completion:

```bash
# Create task
curl -s http://127.0.0.1:8899/api/capability/subtitleTask -X POST -H "Content-Type: application/json" -d '{...}'

# Poll status (every 30s)
curl -s "http://127.0.0.1:8899/api/capability/subtitleTask?taskId=<task_id>"

# Done when process_percent = 100 and video_url is present
```

## Output Location & Filename

Final video is saved to `./output/<date>_<topic>_<type>_embed.mp4`:

**Filename format:** `YYYY-MM-DD_<video_id>_<type>_embed.mp4`

**Example:** `2026-05-07_KyVWnPdS8Yg_vertical_embed.mp4`

- **Date**: Creation date (YYYY-MM-DD)
- **Video ID**: YouTube video ID or task identifier
- **Type**: `vertical` (9:16) or `horizontal` (16:9)

**Note:** For YouTube videos, the video ID is extracted from the URL. For local files, use the filename without extension.

## Testing

Test video: `/Users/baochen10luo/PaultoDo/downloads/shorts_I3W46NuGg18.mp4` (47s vertical Shorts)
Output: `output/<task_id>_vertical_embed.mp4`

Expected final audio duration: ~47s (matches original video)
