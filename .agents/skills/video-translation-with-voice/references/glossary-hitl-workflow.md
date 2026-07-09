# Glossary and HITL Workflow

Use this reference when a video has domain-specific terminology, named people, brands, teams, tools, or competition jargon.

## Goals

- Translate domain terms consistently into Traditional Chinese.
- Keep English person names, nicknames, brands, and acronyms in English unless the user explicitly asks for Chinese names.
- Keep the glossary reusable in `config/glossaries/<domain>_zh_tw.json`.
- Apply glossary fixes before TTS so subtitles and dubbed audio match.

## Dictionary File

Create or update:

```text
config/glossaries/<domain>_zh_tw.json
```

Use this shape:

```json
{
  "domain": "pickleball",
  "target_language": "zh-TW",
  "terms": {
    "pickleball": "匹克球",
    "paddle": "匹克球拍",
    "kitchen": "廚房區",
    "kitchen violation": "廚房區違規"
  },
  "protected_names": [
    "Andre Agassi",
    "Andy Roddick",
    "Courtney Johnson"
  ],
  "notes": [
    "Keep protected_names in English in subtitles and TTS text."
  ]
}
```

## Extraction Pass

Inspect these files:

```bash
sed -n '1,220p' "tasks/<task_id>/origin_language_srt.srt"
sed -n '1,220p' "tasks/<task_id>/translated.srt"
sed -n '1,220p' "tasks/<task_id>/review.txt"
```

Extract:

- People and nicknames.
- Competition names, sponsors, teams, brands, and acronyms.
- Domain vocabulary that should have a stable Chinese rendering.
- False friends or mistranslations from the LLM pass.

## Replacement Rules

- Replace Chinese-transliterated person names with English originals.
- Keep brand/acronym casing stable, such as `YOLO`, `ATP`, `ESPN`.
- Replace domain mistranslations, such as `泡泡球` -> `匹克球`.
- Replace ambiguous sport terms with domain-specific terms, such as `廚房` -> `廚房區` when referring to pickleball.
- Avoid double replacements, such as `廚房區區`.

## Audit Commands

Run the bundled audit script after generating revised SRT/ASS:

```bash
node .agents/skills/video-translation-with-voice/scripts/subtitle_audit.mjs \
  --srt "tasks/<task_id>/target_language_srt.srt" \
  --ass "tasks/<task_id>/formatted_subtitles.ass" \
  --glossary "config/glossaries/<domain>_zh_tw.json" \
  --ban "泡泡球" \
  --ban-regex "廚房(?!區)"
```

For one-off manual scans, use PCRE2 when lookahead is needed:

```bash
rg --pcre2 -n '泡泡球|廚房(?!區)|安迪|阿加西|羅迪克' "tasks/<task_id>"
```

## Regeneration Rule

If glossary fixes change text after TTS already exists:

- Regenerate only affected cue WAV files.
- Rebuild `audio_concat_list.txt`, `tts_final_audio.wav`, `audio_duration_details.txt`, `target_language_srt.srt`, and `formatted_subtitles.ass`.
- Re-mix video audio and re-burn subtitles.
- Do not publish a video where subtitle text and spoken TTS text differ.
