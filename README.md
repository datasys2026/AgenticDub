<div align="center">

# AgenticDub

AI 影片翻譯配音工具（Go HTTP backend + MCP proxy）

</div>

## 概覽

```
影片 → 下載 → STT → LLM 翻譯 → 翻譯稽核 → HITL 人工審核 → TTS → 字幕燒錄
```

**開發目標：** Agentic 化 — planner、tool use、memory、state machine 讓流程可規劃、可 interruption 恢復。

## 快速開始

```bash
# Web server mode
go run ./cmd/server/main.go

# MCP server mode（供 AI client 呼叫）
go build -o agenticdub-mcp ./cmd/mcp/ && ./agenticdub-mcp
```

## 架構

```
cmd/
  server/              # Web server entry point (Gin)
  mcp/                 # thin stdio MCP proxy to HTTP backend
  cli/                 # legacy standalone CLI entry point

internal/
  agent/               # Agent 核心
    hitl/              # HITL 審核系統
  service/             # 核心商業邏輯
    audio2subtitle.go  # Gin pipeline: 下載/STT/翻譯/稽核/HITL
    legacy_cli_pipeline.go # CLI legacy pipeline
    task_state.go      # SQLite-backed task state
    srt2speech.go      # TTS 合成
    srt_embed.go       # 字幕燒錄
  deps/                # 環境依賴檢查
  handler/             # API handler
  router/              # Gin router
  storage/             # 檔案處理
  types/               # 類型定義 + prompts

pkg/
  fasterwhisper/       # 本地 STT
  openai/              # OpenAI-compatible 客戶端
  whispercpp/          # Whisper.cpp
  whisperkit/          # WhisperKit (macOS M-series)
  aliyun/              # 阿里雲 STT/TTS/OSS
  localtts/            # Edge-TTS
```

主管線是 `cmd/server` 的 Gin HTTP backend：`StartSubtitleTask` 負責下載 → STT → LLM 翻譯 → 翻譯稽核 → HITL 人工審核 → TTS → 字幕燒錄。`cmd/mcp` 只是一層 thin stdio MCP proxy，透過 HTTP 呼叫 backend。`cmd/cli run` 保留為獨立 legacy pipeline（`internal/service/legacy_cli_pipeline.go`），直連 aiark 端點並使用 `internal/translator`。

Python 審核工具鏈維持在 `scripts/build_subtitle_review.py`、`subtitle_review_viewer.html`、`build_semantic_blocks.py`，搭配 `.agents/skills/framecue` 使用。

## Archive Layout

- `archive/docs-upstream`：上游 krillin-ai 多語文件
- `archive/_code-legacy`：desktop UI、whisperx
- `archive/runtime`：舊 task/output 產物，gitignored
- `archive/scripts-superseded`：已被取代的腳本

## MCP Server

MCP server 讓 AI client（如 Claude Desktop）可以呼叫 AgenticDub 的翻譯功能。

### 編譯

```bash
go build -o agenticdub-mcp ./cmd/mcp/
```

### 設定

在 `config/config.toml` 設定 server URL：

```toml
[mcp]
server_url = "http://127.0.0.1:8888"  # 預設使用 [server] 的 host:port
```

### Claude Desktop 配置

```json
{
  "mcpServers": {
    "agenticdub": {
      "command": "/absolute/path/to/agenticdub-mcp"
    }
  }
}
```

### MCP Tools

| Tool | 說明 |
|------|------|
| `list_model_profiles` | 列出可用 LLM/STT/TTS profile 與 0.6B TTS voices |
| `translate_video` | 翻譯影片（URL → STT → 翻譯 → TTS → 燒錄） |
| `get_task_status` | 查詢任務狀態 |
| `list_tasks` | 列出所有任務 |
| `approve_hitl` | 核准 HITL 審核，繼續 TTS |
| `reject_hitl` | 否決 HITL 審核，放棄任務 |
| `get_review` | 取得 review.txt 內容 |
| `get_review_status` | 取得審核狀態 |

### xAI / Grok OAuth（experimental）

AgenticDub 已加入第一階段 xAI OAuth LLM provider。這條路徑不使用 `XAI_API_KEY`，而是讀取本機 OAuth token 檔：

```bash
agenticdub auth xai status --token-path ~/.agenticdub/auth/xai.json
```

若本機已登入 Hermes Agent 的 xAI OAuth，可直接重用 Hermes auth 檔：

```bash
agenticdub auth xai status --token-path ~/.hermes/auth.json
agenticdub auth xai probe --token-path ~/.hermes/auth.json --model grok-4.20-0309-non-reasoning
```

目前已完成 token store、OAuth bearer LLM/STT/TTS providers、`grok`/`xai` model profiles、狀態檢查與 live entitlement probe。Browser login 尚未接入；OAuth surface 仍可能受 Grok subscription / entitlement 限制。

架構細節見 `docs/architecture/xai-oauth-provider.md`。

### 使用範例

```
使用者：幫我翻譯這個影片 https://youtube.com/watch?v=xxx
Claude：使用 translate_video tool
        - url: "https://youtube.com/watch?v=xxx"
        - target_lang: "繁體中文"
        - tts: true
        - llm_profile: "external"
        - stt_profile: "default"
        - tts_profile: "default"
        - voice: "Vivian"

結果：task_id = "xxx_abc1"
```

`translate_video` 預設使用 aiark STT/TTS profile。若要測試全 xAI OAuth pipeline，可使用 `llm_profile = "grok"`、`stt_profile = "xai"`、`tts_profile = "xai"`、`voice = "eve"`。xAI TTS 內建 voices 可用 `eve`、`ara`、`rex`、`sal`、`leo`；aiark 0.6B TTS preset voices 可用 `Vivian`、`Serena`、`Uncle_Fu`、`Dylan`、`Eric`、`Ryan`、`Aiden`、`Ono_Anna`、`Sohee`。

## Phase 狀態

| Phase | 描述 | 狀態 |
|-------|------|------|
| 0 | Go skeleton + Gin web server + config | ✅ |
| 1 | STT providers (openai, fasterwhisper, whispercpp, whisperkit, aliyun) | ✅ |
| 2 | LLM translation + subtitle segmentation | ✅ |
| 3 | TTS providers (openai, aliyun, edge-tts) | ✅ |
| 4 | Video compose (ffmpeg) + subtitle burn | ✅ |
| 5 | **Agentic 重構** — planner + tools + memory + state machine | 🔄 |
| 6 | SQLite task DB — 可恢復 pipeline | ✅ / 🔄 |
| 7 | Reflective translation (3-step) | 🔜 |

## HITL 審核流程

翻譯完成後暫停 90%，進入人工審核：

```
翻譯完成 → 生成 review.txt → 人員編輯字幕 → 核准 → TTS → 燒錄 → 完成
```

```bash
# 查看審核內容
curl http://127.0.0.1:8899/api/hitl/review/<task_id>

# 核准繼續
curl -X POST http://127.0.0.1:8899/api/hitl/approve/<task_id>

# 否決
curl -X POST http://127.0.0.1:8899/api/hitl/reject/<task_id> -d '{"reason":"翻譯錯誤"}'
```

## 環境需求

- Go 1.22+
- ffmpeg-full（需 libass）：`brew install ffmpeg-full`
- yt-dlp（可選）

## 設定

```toml
# config/config.toml
[llm]
provider = "openai"
model = "aiark/gemma4-e2b"
base_url = "http://localhost:4000/v1"

[transcribe]
provider = "fasterwhisper"
model = "large-v3"

[tts]
provider = "openai"
voice = "Ryan"
max_concurrency = 1
```

## 輸出

`./output/<date>_<video_id>_<type>_embed.mp4`

範例：`2026-05-12_KyVWnPdS8Yg_vertical_embed.mp4`

## 開發

```bash
go build -o AgenticDub ./cmd/server/    # 編譯 web server
go build -o agenticdub ./cmd/cli/       # 編譯 CLI
go test ./...                           # 測試
go test -cover ./...                    # 含覆蓋率
```
