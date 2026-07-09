# Review Package

`review_package.json` is the cue-level review input for FrameCue.

Expected shape:

```json
{
  "workflow": "framecue_review_v1",
  "video": {},
  "scenes": [
    {
      "id": 1,
      "start": 0,
      "end": 12.3,
      "image": "frames/scene_0001.jpg"
    }
  ],
  "cues": [
    {
      "id": 1,
      "start": 0,
      "end": 3.2,
      "scene_id": 1,
      "text": "中文字幕",
      "original_text": "English source",
      "audio": "audio/cue_0001.mp3",
      "prompt_note": "",
      "pronunciation_risks": ["OpenClaw"]
    }
  ]
}
```

Important fields:

- `text`: target subtitle shown in the overlay and exported to SRT.
- `original_text`: source subtitle shown above the target subtitle.
- `audio`: cue audio preview path, if available.
- `prompt_note`: cue-level instruction for later LLM rewrite; not shown as subtitle text.
- `pronunciation_risks`: terms surfaced in the Risk cue tab.

FrameCue exports:

- `edited_review_package.json`: all cues after edits.
- `subtitle_change_list.json`: only cues whose subtitle or prompt note changed.
