# Speaker Voice Assignment

Use this reference before TTS when content has, or may have, multiple speakers.

## Hard Rules

- Never choose a random voice per subtitle cue.
- Use one stable voice for each inferred speaker across the whole video.
- Persist the mapping to `tasks/<task_id>/tts_voice_assignments.json`.
- If the user specifies a role or voice, that instruction overrides automatic assignment for the matching speaker.
- If speaker confidence is low, prefer fewer speakers over noisy voice switching.

## Default xAI Voices

Use configured xAI voices from the project profile, currently:

```text
eve, ara, rex, sal, leo
```

Do not assume gender from a name unless the source text or user states it. Pick voices by stable assignment, not by stereotype.

## Detection Inputs

Prefer structured diarization if available. If no diarization exists, infer speakers from:

- STT segment speaker labels if present.
- Original transcript speaker tags.
- Dialogue markers, interview turn-taking, commentator/player context, and repeated names.
- Visual or title context only as weak supporting evidence.

## Speaker Plan

Create a speaker plan before TTS:

```json
{
  "strategy": "auto",
  "confidence": "medium",
  "voices": {
    "speaker_1": {
      "label": "primary_commentator",
      "voice": "sal",
      "evidence": ["dominant narration voice"]
    },
    "speaker_2": {
      "label": "secondary_commentator",
      "voice": "rex",
      "evidence": ["alternating short reactions"]
    }
  },
  "cue_speakers": {
    "1": "speaker_1",
    "2": "speaker_1",
    "3": "speaker_2"
  }
}
```

## Assignment Algorithm

1. Build `speaker_id` values from available diarization or inferred turns.
2. Merge short ambiguous cues into the nearest confident speaker when context supports it.
3. Assign voices by stable hash of `speaker_id` over the configured voice list.
4. If adjacent active speakers receive the same voice and another voice is available, rotate the later speaker to the next available voice.
5. If speaker count exceeds voice count, reuse voices by stable hash, but keep the most frequent speakers distinct.
6. Write the final mapping before TTS generation.

## TTS Generation

For each cue:

- Resolve `speaker_id`.
- Use that speaker's mapped voice.
- Keep text exactly aligned with the final subtitle text.
- If a cue is edited, regenerate only that cue with the same speaker voice.

## Quality Gate

Before finalizing:

```bash
jq . "tasks/<task_id>/tts_voice_assignments.json"
```

Check:

- No cue has an empty voice.
- The same speaker never changes voice.
- Adjacent speaker changes are intentional and not caused by cue-level randomization.
- Low-confidence voice changes are minimized.
