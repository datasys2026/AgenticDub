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
影片 → 音軌分離 → STT → 字幕切割 → LLM 翻譯 → TTS 合成 → 字幕燒錄 → 合併影片
```

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

### 3. ffmpeg with libass
System ffmpeg needs libass for subtitle burning. Install:
```bash
brew install ffmpeg-full
```

Or use: `/opt/homebrew/opt/ffmpeg-full/bin/ffmpeg`

### 4. TTS Concurrency
Set `maxConcurrency := 1` to avoid GPU contention on aiark server.

### 5. Subtitle Embedding
Default `embed_subtitle_video_type = "horizontal"` when not specified.

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
  "embed_subtitle_video_type": "horizontal"
}
```

## Key Files

- `internal/service/audio2subtitle.go` - STT + translation pipeline
- `internal/service/srt2speech.go` - TTS synthesis
- `internal/service/srt_embed.go` - Subtitle burning
- `internal/service/subtitle_service.go` - Task orchestration
- `pkg/openai/openai.go` - OpenAI-compatible client (TTS)
- `internal/deps/checker.go` - ffmpeg path detection

## When to use

Use this skill when:
- Translating English videos to Chinese with dubbed audio
- Burning bilingual subtitles into video
- Processing videos through the full STT→Translate→TTS→Merge pipeline

## Common Issues

1. **Subs are Simplified instead of Traditional**: Check `target_lang` is `繁體中文` not `简体中文`
2. **TTS returns 404 on audio download**: URL should be `/tts/audio/xxx.wav` not `/tts/v1/audio/xxx.wav`
3. **ffmpeg ass filter not found**: Need ffmpeg-full with libass support
4. **GPU busy on TTS**: Reduce concurrency to 1

## Testing

Test video: `/Users/baochen10luo/PaultoDo/downloads/original_en_1min.mp4`
Output: `tasks/<task_id>/output/horizontal_embed.mp4`
