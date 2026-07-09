---
name: framecue
description: Use when preparing, serving, validating, or collecting FrameCue subtitle review packages, including cue review, risk cue review, semantic block review, and browser preview before TTS.
license: MIT
metadata:
  audience: content-ops
  workflow: subtitle-review
---

# FrameCue

Use FrameCue for human-in-the-loop subtitle and semantic block review before TTS.

## Scope

FrameCue stops at review. Do not run TTS, render final video, publish, upload, or delete remote posts unless the user explicitly asks after review approval.

## Inputs

Find or create a preview directory containing:

- `review_package.json`
- `semantic_blocks/semantic_blocks.json` when block review is requested
- `frames/`
- `audio/` when cue audio preview is available
- `index.html` copied from the FrameCue viewer

If schema details matter, read:

- `references/review-package.md`
- `references/semantic-blocks.md`
- `references/usage-log.md` when updating this skill from prior runs

## Standard Workflow

1. Append a short implementation plan to `references/usage-log.md`.
2. Locate the task directory and current subtitle/block artifacts.
3. Build or refresh `review_package.json`.
4. Build or refresh `semantic_blocks/semantic_blocks.json` for block review.
5. Copy the FrameCue viewer to the preview directory as `index.html`.
6. Serve the preview directory locally and expose it over Tailscale when useful.
7. Validate with `curl` and a browser screenshot.
8. Append the result, user feedback if any, and skill update candidates to `references/usage-log.md`.
9. Report the preview URL, cue count, block count, and explicit next stop.

## Validation

Check the smallest useful set:

```bash
curl -fsS "$URL/index.html" | rg "review_package|semantic_blocks|tabCue|tabBlock"
curl -fsS "$URL/review_package.json" | jq '{cue_count: (.cues|length), scene_count: (.scenes|length)}'
curl -fsS "$URL/semantic_blocks/semantic_blocks.json" | jq '{block_count: (.blocks|length), validation}'  # when present
```

For visual validation, use headless Chrome or the current browser tool to verify:

- frame image is visible
- subtitle overlay is visible
- cue/block tabs work
- risk/all cue list tabs work
- help tips do not cover the work area incorrectly

## Review Outputs

FrameCue may export:

- `edited_review_package.json` for cue text changes
- `subtitle_change_list.json` for changed cues and cue prompt notes
- `block_decisions.json` for block target text, speech text, block prompt notes, and approval status

After the user approves the review, hand off to the video translation pipeline for TTS and final rendering.

## Pitfalls

- Do not treat local browser draft state as committed output unless the user exports it or you inspect localStorage intentionally.
- Do not overwrite a user-edited review package without saving a backup.
- Do not reuse stale subtitles when a newer manually edited package exists.
- Do not continue to TTS while the user is still editing FrameCue.
- Do not skip the usage log; it is the source material for improving this skill later.
