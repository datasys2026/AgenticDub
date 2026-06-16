# Postiz YouTube Publish

Use this reference when uploading AgenticDub outputs through the AIA-SMA MCP/Postiz server.

## Safety

- Never print bearer tokens or upload tokens.
- Read MCP tokens from a secret file or configured MCP client secret store.
- Do not write tokens to repo files, logs, summaries, or frontend code.

## Tool Discovery

Always call `tools/list` or the available MCP tool listing first. Do not assume publish tools exist.

Required draft-to-publish tools:

```text
postiz_create_upload_session
postiz_complete_upload_session
postiz_create_youtube_unlisted_post
postiz_publish_post_now
postiz_list_posts
```

Optional but useful:

```text
postiz_auth_status
postiz_account_overview
postiz_list_integrations
postiz_get_post
postiz_get_upload_instructions
```

## Default Flow

1. Call `postiz_auth_status`.
2. Call `postiz_list_integrations` and select the YouTube integration.
3. Call `postiz_get_upload_instructions`.
4. Create upload session with meaningful filename, MIME type, and size.
5. PATCH chunks to the returned tus-compatible upload URL.
6. Complete upload session and capture returned media `id` and `path`.
7. Call `postiz_create_youtube_unlisted_post` with `mode: "draft"`.
8. Call `postiz_publish_post_now` only after the user has clearly asked to publish, or when the current task explicitly requires draft-then-publish.
9. Poll `postiz_list_posts` until `releaseURL` appears.

## Known API Quirks

- `postiz_create_youtube_unlisted_post` schema may show `date` as optional, but the server may still require a valid ISO 8601 `date`. If a 400 says date is required, retry with `new Date(Date.now()+60000).toISOString()`.
- `postiz_get_post` can return 404 for a fresh draft/post even when creation succeeded. Fall back to `postiz_list_posts` over the relevant date range.
- Tool results may return an array such as `[{ "postId": "...", "integration": "..." }]`; handle array results, not only object results.

## YouTube Settings

Use unlisted visibility through `postiz_create_youtube_unlisted_post`.

Recommended payload fields:

```json
{
  "integrationId": "<youtube integration id>",
  "title": "AgenticDub ...",
  "content": "AgenticDub xAI OAuth pipeline ...",
  "media": [{ "id": "<media id>", "path": "<media path>" }],
  "mode": "draft",
  "date": "<ISO 8601 date>",
  "selfDeclaredMadeForKids": "no",
  "shortLink": false
}
```

## Verification

After publishing:

- Record Postiz `postId`.
- Record YouTube `releaseURL`.
- Check `postiz_list_posts` state is `PUBLISHED`.
- Do not report success until a release URL exists.
