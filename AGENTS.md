# AgenticDub — PROJECT AGENTS.md

**Project**: `AgenticDub`
**Module**: `krillin-ai`
**Language**: Go 1.22+

---

## OVERVIEW

AI 影片翻譯配音工具。核心流程：
```
影片 → 下載 → STT → LLM 翻譯 → 翻譯稽核 → HITL 人工審核 → TTS → 字幕燒錄
```

**開發目標：Agentic 化** — 加入 planner、tool use、memory、state machine 讓流程可規劃、可 interruption 恢復。

---

## LOCAL AI ENDPOINTS (aiark-agent)

同一台機器上的 aiark-agent 提供這些端點，CLI 將直接呼叫：

| 服務 | 端點 | 模型 |
|------|------|------|
| **LLM** | `http://localhost:4000/v1/chat/completions` | `aiark/gemma4-e2b`, `aiark/qwen36-35b-iq3`, `aiark/deepseek-r1-14b` |
| **STT** | `http://localhost:8006/v1/audio/transcriptions` | `faster-whisper-large-v3-fp16` |
| **TTS** | `http://localhost:8002/v1/audio/speech` | `Qwen3-TTS-0.6B-CustomVoice` |

所有端點皆為 OpenAI-compatible，共享 `sashabaranov/go-openai` 客戶端。

## XAI / GROK OAUTH (experimental)

目前 xAI / Grok 作為 experimental OAuth LLM/STT/TTS provider：
- 不使用 `XAI_API_KEY`
- token 預設路徑：`~/.agenticdub/auth/xai.json`
- 本機 Hermes Agent 整合路徑：`~/.hermes/auth.json`
- CLI status：`go run ./cmd/cli auth xai status`
- CLI probe：`go run ./cmd/cli auth xai probe --token-path ~/.hermes/auth.json --model grok-4.20-0309-non-reasoning`
- model profile：`models.llm.grok` → `provider = "xai-oauth"`, `model = "grok-4.20-0309-non-reasoning"`
- STT profile：`models.stt.xai` → `provider = "xai-oauth"`, `model = "xai-stt"`
- TTS profile：`models.tts.xai` → `provider = "xai-oauth"`, `model = "xai-tts"`, voices `eve`, `ara`, `rex`, `sal`, `leo`

OAuth audio surface 仍可能受 Grok subscription / entitlement 限制；本機已用 Hermes token smoke test 通過 xAI STT/TTS endpoint。
架構細節見 `docs/architecture/xai-oauth-provider.md`。

---

## STRUCTURE

```
cmd/
  server/              # 目前 entry point (Gin web server)
  cli/                 # 目前 CLI entry point (cobra)
  mcp/                 # thin stdio MCP proxy to HTTP backend

internal/
  agent/               # Agent 核心
    db.go              # SQLite task DB
    planner.go         # LLM 規劃翻譯策略
    tool.go            # Tool definitions (STT/TTS/LLM)
    memory.go          # 術語庫、對話歷史
    state.go           # 狀態機管理
  deps/                # 環境依賴檢查 (ffmpeg, yt-dlp)
  dto/                 # Data transfer objects
  handler/             # API handler + config UI
  translator/          # 字幕翻譯、切分、對齊
  response/            # API response wrappers
  router/              # Gin router setup
  server/              # Gin server 啟動
  service/             # 核心商業邏輯 (主要 pipeline)
    audio2subtitle.go  # 主要 pipeline (1389 行)
    legacy_cli_pipeline.go # 獨立 legacy CLI pipeline
    task_state.go      # 可恢復 task state
    srt2speech.go      # TTS 合成
    srt_embed.go       # 字幕燒錄進影片
    timestamps.go      # 時間軸對齊
  providers/           # STT / LLM / TTS providers
  storage/             # 檔案處理工具
  types/               # 類型定義 + 翻譯 prompts

pkg/
  aliyun/              # 阿里雲 STT/TTS/OSS
  fasterwhisper/       # 本地 faster-whisper
  openai/              # OpenAI-compatible 客戶端工廠
  whisper/             # Whisper API
  whispercpp/          # Whisper.cpp
  whisperkit/          # WhisperKit (macOS M-series)
  localtts/            # Edge-TTS

config/
  config-example.toml  # 設定檔範例
```

---

## EXISTING ENTRY POINTS

```bash
# Web server mode (目前)
go run ./cmd/server/main.go

# CLI mode (目前)
go run ./cmd/cli run "https://youtube.com/watch?v=xxx"

# MCP server
go run ./cmd/mcp
```

主管線是 `cmd/server` 的 Gin HTTP backend：`StartSubtitleTask` 負責下載 → STT → LLM 翻譯 → 翻譯稽核 → HITL 人工審核 → TTS → 字幕燒錄。`cmd/mcp` 只是一層 thin stdio MCP proxy，透過 HTTP 呼叫 backend。`cmd/cli run` 保留為獨立 legacy pipeline（`internal/service/legacy_cli_pipeline.go`），直連 aiark 端點並使用 `internal/translator`。

Python 審核工具鏈維持在 `scripts/build_subtitle_review.py`、`subtitle_review_viewer.html`、`build_semantic_blocks.py`，搭配 `.agents/skills/framecue` 使用。

## ARCHIVE LAYOUT

- `archive/docs-upstream`：上游 krillin-ai 多語文件
- `archive/_code-legacy`：desktop UI、whisperx
- `archive/runtime`：舊 task/output 產物，gitignored
- `archive/scripts-superseded`：已被取代的腳本

---

## AGENTIC ARCHITECTURE (規劃)

```
┌──────────────────────────────────────────────┐
│              Planner Agent                   │
│  - 分析影片內容、語言、領域                    │
│  - 決定翻譯策略 (fast / reflective)          │
│  - 選擇 TTS voice                            │
│  - 提取術語庫                                │
└──────────────────────────────────────────────┘
                           │
          ┌────────────────┼────────────────┐
          ▼                ▼                ▼
    [STT Tool]       [LLM Tool]        [TTS Tool]
    語音轉文字        批次翻譯           語音合成
          │                │                │
          └────────────────┴────────────────┘
                           │
                    [State Machine]
                    task.db (SQLite)
```

**Reflective Translation Strategy (3 步驟)**:
1. `direct` — 直接翻譯
2. `reflect` — 檢視問題
3. `paraphrase` — 修正後產出最終譯文

---

## CONVENTIONS

### 命名
- interface：`Transcriber`, `ChatCompleter`, `Ttser`
- 具體實作：`OpenAIClient`, `FasterWhisperProcessor`
- constructor：`NewXxxProcessor()` 回傳 interface

### 錯誤處理
- 使用 `go.uber.org/zap` logging
- 業務錯誤不回傳 panic
- 所有 error 向上傳遞

### IO
- Web mode：Gin JSON API
- 未來 CLI mode：stdout JSON，stderr logs

### 設定優先順序
flag > 環境變數 > `config/config.toml` > 預設值

### 編碼行為準則
- **編碼前深思熟慮**：不假設、不隱藏困惑；明確指出假設，不確定就提問；有更簡單的方法主動提出
- **簡單至上**：寫最少量程式碼解決問題，不做推測性設計；不添加要求以外的功能
- **精確的修改**：只更動絕對必要的部分；不「改善」相鄰程式碼；配合現有程式碼風格
- **目標導向執行**：定義成功標準，持續驗證直到確認無誤；多步驟任務列出計畫

---

## COMMANDS

```bash
# 開發
go build -o AgenticDub ./cmd/server/     # 編譯 web server
go build -o AgenticDub-cli ./cmd/cli/    # 編譯 CLI
go test ./...                            # 測試
go test -cover ./...                     # 含覆蓋率

# 環境檢查 (doctor)
go run ./cmd/cli doctor

# 影片翻譯 (CLI mode)
go run ./cmd/cli run "url" --target-lang "繁體中文"
```

---

## ANTI-PATTERNS

- 禁止在業務流程函式內分散建立 provider；統一集中在 `service.NewService()` / `NewServiceWithConfig()`
- 禁止在 `types/` 放業務邏輯（純資料結構 + prompts）
- 禁止在 cmd/ 層寫商業邏輯

---

## PHASE STATUS

- [x] Phase 0：Go skeleton + Gin web server + config
- [x] Phase 1：STT providers (openai, fasterwhisper, whispercpp, whisperkit, aliyun)
- [x] Phase 2：LLM translation + subtitle segmentation
- [x] Phase 3：TTS providers (openai, aliyun, edge-tts)
- [x] Phase 4：Video compose (ffmpeg) + subtitle burn
- [ ] Phase 5：**Agentic 重構** — planner + tools + memory + state machine
- [x] Phase 6：SQLite task DB — 可恢復 pipeline（`internal/service/task_state.go` + `internal/agent/db.go` 已接入主管線）
- [ ] Phase 7：Reflective translation (3-step)

**Current：Phase 5 進行中**
