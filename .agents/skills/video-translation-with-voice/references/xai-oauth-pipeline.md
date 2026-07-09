# xAI OAuth Pipeline

Use this reference when running or debugging the AgenticDub default xAI OAuth pipeline.

## Preconditions

- Local config has xAI OAuth enabled and token path configured.
- `ffmpeg-full` with libass is available.
- The local API server is not already occupying `127.0.0.1:8899`.
- Test video exists at:

```text
/Users/baochen10luo/PaultoDo/agents/AgenticDub/testdata/videos/shorts_I3W46NuGg18.mp4
```

## Start Server

```bash
go run ./cmd/server/main.go
```

Expected server:

```text
127.0.0.1:8899
```

If the port is busy:

```bash
lsof -nP -iTCP:8899 -sTCP:LISTEN
```

Stop only the local AgenticDub server process started for this test. Do not stop aiark-agent, Hermes, xAI services, or unrelated processes.

## Submit Test Task

```bash
curl -s http://127.0.0.1:8899/api/capability/subtitleTask \
  -X POST \
  -H 'Content-Type: application/json' \
  -d '{
    "url":"local:/Users/baochen10luo/PaultoDo/agents/AgenticDub/testdata/videos/shorts_I3W46NuGg18.mp4",
    "origin_lang":"en",
    "target_lang":"繁體中文",
    "bilingual":0,
    "translation_subtitle_pos":0,
    "modal_filter":0,
    "tts":1,
    "tts_voice_code":"",
    "llm_profile":"grok",
    "stt_profile":"xai",
    "tts_profile":"xai",
    "language":"zh",
    "embed_subtitle_video_type":"vertical"
  }'
```

Record `data.task_id`.

## Poll

```bash
curl -s "http://127.0.0.1:8899/api/capability/subtitleTask?taskId=<task_id>" | jq .
```

At `process_percent = 90`, inspect:

```bash
sed -n '1,160p' "tasks/<task_id>/review.txt"
sed -n '1,160p' "tasks/<task_id>/target_language_srt.srt"
```

Approve only after review text is acceptable:

```bash
curl -s -X POST "http://127.0.0.1:8899/api/hitl/approve/<task_id>" | jq .
```

After approval, poll until `process_percent = 100`.

## Expected Completion

Successful API response should include:

- `speech_download_url`
- `video_download_url`

Example final files:

```text
tasks/<task_id>/target_language_srt.srt
tasks/<task_id>/formatted_subtitles.ass
tasks/<task_id>/audio_duration_details.txt
tasks/<task_id>/tts_final_audio.wav
tasks/<task_id>/video_with_tts.mp4
output/YYYY-MM-DD_shorts_I3W46NuGg18_vertical_embed.mp4
```

## Media Checks

```bash
ffprobe -v error \
  -show_entries format=duration:stream=index,codec_type,codec_name,width,height,sample_rate,channels \
  -of json "output/YYYY-MM-DD_shorts_I3W46NuGg18_vertical_embed.mp4"
```

Expected:

- Duration about `47.067s`.
- Video `1080x1920`.
- Video codec `h264`.
- Audio codec `aac`.

Check download URL:

```bash
curl -sI "http://127.0.0.1:8899/api/file/output/YYYY-MM-DD_shorts_I3W46NuGg18_vertical_embed.mp4"
```

Expected: `HTTP/1.1 200 OK`.

## Server Cleanup

After live validation:

```bash
lsof -nP -iTCP:8899 -sTCP:LISTEN
```

Stop only the AgenticDub server process if still running.
