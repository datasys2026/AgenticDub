# YouTube Private Upload Status - 2026-06-30

Channel: 有AI的小編元元  
Handle: @yaunyuan2026  
YouTube integrationId: `cmp26rj7b000lnt85qhpth6vr`

## Current Goal

Upload completed videos to YouTube as actual `private` videos, not only Postiz drafts.

## Completed

- Uploaded media files to Postiz media storage on 2026-06-29.
- Created 5 YouTube `private` Postiz posts on 2026-06-30.
- Called `postiz_publish_post_now` for all 5 posts.

## Current Blocker

All 5 posts are still in Postiz `QUEUE`.

Reason found:

- Their `publishDate` is `2026-06-30T08:00:00Z`.
- This is `2026-06-30 16:00` Asia/Taipei.
- Current MCP tools do not expose a way to delete/cancel queued Postiz posts or edit their publish date.

The user said a new tool `postiz_delete_post` was added, but after MCP tool discovery it is still not visible in this Codex session.

## Queued Private Posts

| Video | Postiz postId | State | Intended visibility |
|---|---|---|---|
| OpenClaw Creator: Why 80% Of Apps Will Disappear | `cmr07fi36003so1804v8a0h8y` | `QUEUE` | private |
| The AI Agent Economy Is Here | `cmr07fi3n003to180xjeftrrp` | `QUEUE` | private |
| Common Mistakes With Vibe Coded Websites | `cmr07fi3s003uo1800s5uae1y` | `QUEUE` | private |
| The Meta Harness: Why Every AI Developer Needs This | `cmr07fi3x003vo18095qg5hth` | `QUEUE` | private |
| 從 AI 到 Agent：2026 年的典範轉移 | `cmr07fi41003wo180s2vryqej` | `QUEUE` | private |

## Older Drafts To Avoid Publishing

These were created earlier as `unlisted` drafts and should not be used for the current private-upload goal:

| Video | Old unlisted draft postId |
|---|---|
| OpenClaw Creator | `cmqyyc7hb003no180q3a9gwgd` |
| AI Agent Economy | `cmqyyezkm003oo1809v9nx05y` |
| Vibe Coding Websites | `cmqyyhnyu003po180e8g8lddt` |
| Meta Harness | `cmqyyj7ri003qo180h5zjvgxl` |
| 從 AI 到 Agent | `cmqyylnc3003ro18089t37kvl` |

## Next Step

When `postiz_delete_post` becomes visible:

1. Delete the 5 queued private posts listed above.
2. Recreate 5 private posts with an immediate date.
3. Call `postiz_publish_post_now`.
4. Poll `postiz_list_posts` until each post has `state=PUBLISHED` and a YouTube `releaseURL`.
5. Confirm each YouTube video privacy is `private` with `postiz_get_youtube_video`.

If `postiz_delete_post` does not become available, wait until `2026-06-30 16:00` Asia/Taipei and then verify release URLs.
