# Archive Cleanup - 2026-07-09

## Upstream krillin-ai translated docs

Planned move: `docs/ar`, `docs/de`, `docs/es`, `docs/fr`, `docs/jp`, `docs/kr`, `docs/pt`, `docs/rus`, `docs/vi`, and `docs/zh` to `archive/docs-upstream/`.
Reason: upstream translated documentation kept for reference, not active AgenticDub development.

## Desktop-era faq.md

Planned move: `faq.md` to `archive/docs-upstream/faq.md`.
Reason: desktop-era FAQ retained for reference.

## Stale duplicate agent-skill

Planned move: `agent-skill` to `archive/skills-legacy/agent-skill`.
Reason: stale duplicate of the canonical `.agents/skills/video-translation-with-voice`.

## Superseded one-off Python scripts

Moved `scripts/rebuild_synced_dub.py` and `scripts/build_interpreter_preview.py` to `archive/scripts-superseded/`.
Reason: superseded one-off helpers retained outside the active scripts folder.

## Pre-June-2026 experiment task dirs

Moved top-level `tasks/` entries with mtime strictly before 2026-06-01 to `archive/runtime/tasks/`, excluding `tasks/task_state.db` and `tasks/xai_voice_clone_preview_2026-07-01`.
Reason: old experiment task artifacts retained outside active runtime state.

## Superseded pre-2026-06-20 output renders

Moved top-level `output/` entries with mtime strictly before 2026-06-20 to `archive/runtime/output/`.
Reason: superseded generated renders retained outside active output.

## Runtime log

Moved `app.log` to `archive/runtime/logs/app.log`.
Reason: runtime log retained outside the repo root.

## Large binary history

Rewriting git history to purge old large-binary blobs was deliberately not done as part of this cleanup.
