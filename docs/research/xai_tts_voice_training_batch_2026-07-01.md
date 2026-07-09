# xAI TTS Voice Training Batch - 2026-07-01

## Goal

Prepare a first reusable batch of xAI-generated TTS audio and matching Chinese text for voice adaptation / voice-clone experiments.

This is a candidate manifest only. Do not start model training until licensing and sample quality are checked.

## Source Pattern

Each source task already contains sentence-level pairs:

- audio: `tasks/<task_id>/subtitle_<cue_id>.wav`
- text: matching cue text from `tasks/<task_id>/final.srt`

The first pass uses contiguous early cues until the accumulated TTS duration is about 10 minutes.

## Candidate Batch

| Item | Task | Text Source | Audio Pattern | Cue Range | Matched WAVs | Approx TTS Duration |
|---|---|---|---|---:|---:|---:|
| OpenClaw | `tasks/youtube_4uzGDAoNOZc_zh_tw_2026-06-26` | `final.srt` | `subtitle_<id>.wav` | `1-204` | `494 / 496` total | `10.03 min` selected, `24.23 min` total |
| Tokenmaxxing | `tasks/youtube_57lDpTwiW6g_zh_tw_2026-06-26` | `final.srt` | `subtitle_<id>.wav` | `1-200` | `873 / 873` total | `10.02 min` selected, `42.90 min` total |
| AI Agent Economy | `tasks/youtube_Q8wVMdwhlh4_zh_tw_2026-06-26` | `final.srt` | `subtitle_<id>.wav` | `1-177` | `505 / 505` total | `10.01 min` selected, `27.30 min` total |
| Vibe | `tasks/youtube_DNSXlBmukck_zh_tw_2026-06-26` | `final.srt` | `subtitle_<id>.wav` | `1-208` | `835 / 835` total | `10.02 min` selected, `37.99 min` total |

Total selected duration: about `40.08 min`.

## Dataset Row Shape

Use one row per cue:

```json
{
  "audio": "tasks/youtube_4uzGDAoNOZc_zh_tw_2026-06-26/subtitle_1.wav",
  "text": "今天 我要來跟 Peter Steinberger 聊聊",
  "source_task": "youtube_4uzGDAoNOZc_zh_tw_2026-06-26",
  "cue_id": 1,
  "voice_source": "xai-tts",
  "language": "zh-TW"
}
```

## Quality Rules

- Keep only cues whose audio and text match exactly.
- Drop cues with silence, clipping, repeated words, wrong language, or obvious pronunciation failure.
- Prefer short natural lines, roughly `2-12s`.
- Preserve English technical terms such as `OpenClaw`, `OpenCL`, `GitHub`, `AI`.
- Do not mix this with aiark fallback voices such as `Ryan` or `podcast小姐姐`.

## Next Step

Create a real machine-readable manifest from the selected cue ranges:

```text
datasets/xai_tts_voice_batch_2026-07-01/manifest.jsonl
```

Skipped for now: copying or transcoding audio files. The existing `subtitle_<id>.wav` files are already local and can be referenced directly until the training tool requires a flat dataset folder.
