<div align="center">

# KrillinAI-CLI

AI 影片翻譯配音工具（Go CLI）

</div>

## 概覽

```
影片 → 音軌分離 → STT → 字幕切割 → LLM 翻譯 → [HITL 審核] → TTS 合成 → 字幕燒錄 → 合併影片
```

**開發目標：** Agentic 化 — planner、tool use、memory、state machine 讓流程可規劃、可 interruption 恢復。

## 快速開始

```bash
# Web server mode
go run ./cmd/server/main.go

# MCP server mode（供 AI client 呼叫）
go build -o krillin-mcp ./cmd/mcp/ && ./krillin-mcp
```

## 架構

```
cmd/
  server/              # Web server entry point (Gin)
  mcp/                 # MCP server entry point
  polydub/             # 未來 CLI entry point (cobra)

internal/
  agent/               # Agent 核心
    hitl/              # HITL 審核系統
  api/                 # Gin API handlers
  service/             # 核心商業邏輯
    audio2subtitle.go  # STT + 翻譯 pipeline
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

## MCP Server

MCP server 讓 AI client（如 Claude Desktop）可以呼叫 KrillinAI 的翻譯功能。

### 編譯

```bash
go build -o krillin-mcp ./cmd/mcp/
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
    "krillin-ai": {
      "command": "/absolute/path/to/krillin-mcp"
    }
  }
}
```

### MCP Tools

| Tool | 說明 |
|------|------|
| `translate_video` | 翻譯影片（URL → STT → 翻譯 → TTS → 燒錄） |
| `get_task_status` | 查詢任務狀態 |
| `list_tasks` | 列出所有任務 |
| `approve_hitl` | 核准 HITL 審核，繼續 TTS |
| `reject_hitl` | 否決 HITL 審核，放棄任務 |
| `get_review` | 取得 review.txt 內容 |
| `get_review_status` | 取得審核狀態 |

### 使用範例

```
使用者：幫我翻譯這個影片 https://youtube.com/watch?v=xxx
Claude：使用 translate_video tool
        - url: "https://youtube.com/watch?v=xxx"
        - target_lang: "繁體中文"
        - tts: true
        - voice: "Ryan"

結果：task_id = "xxx_abc1"
```

## Phase 狀態

| Phase | 描述 | 狀態 |
|-------|------|------|
| 0 | Go skeleton + Gin web server + config | ✅ |
| 1 | STT providers (openai, fasterwhisper, whispercpp, whisperkit, aliyun) | ✅ |
| 2 | LLM translation + subtitle segmentation | ✅ |
| 3 | TTS providers (openai, aliyun, edge-tts) | ✅ |
| 4 | Video compose (ffmpeg) + subtitle burn | ✅ |
| 5 | **Agentic 重構** — planner + tools + memory + state machine | 🔄 |
| 6 | SQLite task DB — 可恢復 pipeline | 🔜 |
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
go build -o krillin-ai ./cmd/server/   # 編譯
go test ./...                           # 測試
go test -cover ./...                    # 含覆蓋率
```
