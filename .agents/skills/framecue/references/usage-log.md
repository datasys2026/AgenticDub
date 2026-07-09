# FrameCue Usage Log

Append one entry each time this skill is used or the user gives feedback about FrameCue.

Use this format:

```markdown
## YYYY-MM-DD - short task name

Request:
- What the user asked for.

Implementation plan:
- The concrete steps planned before changing files or serving preview.

Result:
- Preview path or URL.
- Cue/block counts.
- Validation performed.
- Files changed.

User feedback:
- Direct user feedback during or after review.

Skill update candidates:
- What should be added, removed, or clarified in this skill next time.
```

## 2026-07-08 - add usage logging

Request:
- Record the implementation plan and user feedback each time FrameCue is used, so the skill can be improved later.

Implementation plan:
- Add usage-log guidance to `SKILL.md`.
- Add this append-only usage log template.
- Validate the skill files exist and references resolve.

Result:
- Added `references/usage-log.md`.
- Updated `SKILL.md` to require implementation-plan and feedback logging.
- Validation passed with `framecue skill ok`.

User feedback:
- User wants implementation plans and operational feedback preserved for future skill updates.

Skill update candidates:
- After several real FrameCue runs, summarize repeated feedback into `SKILL.md` and trim old noisy log entries if they start bloating context.
