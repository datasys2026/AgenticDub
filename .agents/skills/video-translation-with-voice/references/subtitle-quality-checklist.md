# Subtitle Quality Checklist

Use this reference when validating AgenticDub subtitle smoothness, punctuation cleanup, ASS rendering, and TTS timing.

## Review Text

Check `tasks/<task_id>/review.txt` at HITL pause:

- `字幕：` lines should be natural Traditional Chinese.
- No target subtitle punctuation such as `，。？！、；：·・･`.
- Technical tokens should remain intact: `OpenCL`, `Linux`, `30`.
- Segment indices should be sequential and match SRT order.

## Target SRT

After HITL approval, check:

```bash
sed -n '1,160p' "tasks/<task_id>/target_language_srt.srt"
```

Rules:

- Target text lines should not contain `，。？！、；：·・･`.
- Target-only tasks should not include source English in `target_language_srt.srt`.
- Keep each subtitle as one visual line unless the user explicitly asks for multiline subtitles.
- Keep protected English names, acronyms, and brands intact.
- Subtitle durations should be readable:
  - minimum display around `1.5s`;
  - target pace around `6` CJK chars/sec;
  - never extend past the next subtitle start.

Run the bundled audit script:

```bash
node .agents/skills/video-translation-with-voice/scripts/subtitle_audit.mjs \
  --srt "tasks/<task_id>/target_language_srt.srt" \
  --ass "tasks/<task_id>/formatted_subtitles.ass" \
  --glossary "config/glossaries/<domain>_zh_tw.json"
```

## ASS Burn-In

Check:

```bash
sed -n '1,220p' "tasks/<task_id>/formatted_subtitles.ass"
```

Rules:

- Chinese target lines should use `Major`, not `Minor`, even if they contain `OpenCL` or `Linux`.
- One SRT block should normally produce one ASS `Dialogue` event.
- Use `\N` line breaks inside the same `Dialogue`.
- Do not split protected ASCII tokens:
  - good: `OpenCL`
  - good: `Linux`
  - good: `30`
  - bad: `3\N0`

## TTS Timing

Check:

```bash
sed -n '1,180p' "tasks/<task_id>/audio_duration_details.txt"
```

Rules:

- Each `[n] 原文時間=` should correspond to the current subtitle's own display interval.
- Long gaps between subtitles should appear as separate `句後靜音=...`.
- The first subtitle should not speak before its start; if needed, the log should include initial `Silence`.
- Avoid putting the entire gap to next subtitle inside a sentence's front/back padding.

## Media Output

Check final video and TTS audio:

```bash
ffprobe -v error -show_entries format=duration:stream=index,codec_type,codec_name,width,height,sample_rate,channels -of json "<path>"
```

Expected for the standard 47s vertical test video:

- Final video duration about `47.067s`.
- Final video resolution `1080x1920`.
- Final video audio is AAC.
- `tts_final_audio.wav` is PCM, 24kHz mono, about the same duration as the original video.

For normal source videos, preserve original aspect ratio unless the user explicitly requested vertical output.

## Common Failure Patterns

- Punctuation remains in final SRT: HITL approved content was not applied to target SRT.
- Punctuation remains in ASS: display cleaner is not applied in `srt_embed.go`.
- Subtitle appears too late vs speech: sentence audio is absorbing the gap to next subtitle.
- Subtitle disappears too fast: readable timing thresholds are too aggressive.
- `OpenCL`/`Linux` lines use `Minor`: mixed Chinese/Latin detection is wrong.
- Speaker voice changes randomly across cues: voice assignment happened per cue instead of per inferred speaker.
- Final video has stale subtitles: ASS burn-in reused an older video after text/TTS edits.
