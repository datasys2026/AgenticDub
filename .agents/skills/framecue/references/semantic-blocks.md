# Semantic Blocks

`semantic_blocks/semantic_blocks.json` is the block-level review input for FrameCue.

Expected shape:

```json
{
  "workflow": "openclaw_block_review_v1",
  "blocks": [
    {
      "id": "b0001",
      "input_ids": [1, 2],
      "source_cue_ids": [1, 2],
      "start": 0,
      "end": 3.9,
      "budget_ms": 3900,
      "source_text": "English source text",
      "target_text": "中文字幕稿",
      "speech_text": "中文口語稿"
    }
  ],
  "validation": {
    "block_count": 1,
    "cue_count": 2,
    "empty_target_blocks": 0
  }
}
```

Important fields:

- `source_text`: English/source content for comparison.
- `target_text`: block-level Chinese text for review.
- `speech_text`: spoken interpretation text intended for later TTS.
- `input_ids` / `source_cue_ids`: cue ids covered by this block.
- `budget_ms`: approximate available duration.

FrameCue exports `block_decisions.json`:

```json
{
  "workflow": "framecue_block_decisions_v1",
  "decisions": [
    {
      "id": "b0001",
      "source_cue_ids": [1, 2],
      "target_text": "edited target",
      "speech_text": "edited speech",
      "note": "LLM rewrite prompt for this block",
      "decision": "approved"
    }
  ]
}
```

Use block notes as later LLM rewrite prompts, not as visible subtitles.
