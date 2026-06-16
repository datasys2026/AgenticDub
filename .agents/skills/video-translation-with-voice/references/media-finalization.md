# Media Finalization

Use this reference when producing the final dubbed/subtitled video.

## ffmpeg Selection

ASS burn-in requires `libass`. Check the active binary first:

```bash
ffmpeg -hide_banner -filters | rg ' ass |subtitles|drawtext'
```

If `ass` or `subtitles` is missing, look for `ffmpeg-full`:

```bash
find /opt/homebrew /usr/local /usr/bin /opt -name ffmpeg -type f 2>/dev/null
```

Prefer:

```text
/opt/homebrew/Cellar/ffmpeg-full/<version>/bin/ffmpeg
```

Do not silently fall back to an old burned-subtitle video unless the user explicitly accepts stale subtitles.

## Aspect Ratio

- Preserve the source aspect ratio by default.
- Do not convert 16:9 source to 9:16 unless the user explicitly asks for vertical output.
- Validate final dimensions with `ffprobe`.

```bash
ffprobe -v error \
  -show_entries stream=index,codec_type,codec_name,width,height,duration:format=duration,size \
  -of json "<final.mp4>"
```

## Audio Longer Than Video

If dubbed audio is longer than source video:

- Do not cut the dubbed audio.
- Extend the last video frame with `tpad=stop_mode=clone`.
- Add a short visual fade-out near the end.
- Keep final duration at least as long as the mixed audio.

Example filter:

```text
tpad=stop_mode=clone:stop_duration=0.4,ass='<subtitles.ass>',fade=t=out:st=<duration-1.3>:d=1.3
```

## Audio Mix

Default mix:

- Original audio under dub: about `0.18`.
- TTS voice: `1.0`.
- `amix=inputs=2:duration=longest:normalize=0`.

## Naming

Use meaningful filenames:

```text
output/agenticdub_<source_id>_<target_lang>_<aspect>_<domain>_<version>_dubbed_subtitled_<date>.mp4
```

Example:

```text
output/agenticdub_GYNDicr91Mw_zh_tw_16x9_pickleball_glossary_v2_dubbed_subtitled_2026-06-16.mp4
```

Avoid random hashes in local output filenames. Remote media services may rename uploaded files internally; keep the local artifact meaningful.

## Final Checks

Before upload or delivery:

- `ffprobe` confirms expected aspect ratio and codec.
- A representative screenshot confirms subtitles are burned in and positioned correctly.
- `subtitle_audit.mjs` passes for SRT/ASS.
- Final file size is compatible with the upload path.
