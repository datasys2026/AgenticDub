# AgenticDub 大整理與 Skill 化成果

日期：2026-07-10（工作日 2026-07-09）

## 這次做了什麼

一次完整的專案收斂：全專案 review → 過時內容封存 → mcplize merge 回 master → git 歷史瘦身 → 把「同步口譯配音」流程固化成可重複使用的 skill。

### 1. 封存整理（archive/）

| 位置 | 內容 |
| --- | --- |
| `archive/docs-upstream/` | 上游 krillin-ai 的 10 語系文件、桌面時代 faq.md |
| `archive/_code-legacy/` | cmd/desktop、internal/desktop、internal/api、pkg/whisperx（底線開頭目錄，go tool 自動略過） |
| `archive/scripts-superseded/` | rebuild_synced_dub.py、build_interpreter_preview.py（已被語意區塊法取代） |
| `archive/skills-legacy/` | 舊版 agent-skill/（canonical 版在 .agents/skills/） |
| `archive/runtime/` | 2026-06 前的舊 task/output 產物、app.log（gitignored） |

保留判斷的重點：`internal/translator` 原列死碼候選，但 `legacy_cli_pipeline.go`（現行 `cli run`）仍引用，故保留。

### 2. Merge 與歷史瘦身

- mcplize（17 commits + 5 個收尾 commits）以 `--no-ff` merge 回 master，master 為唯一主線。
- `git filter-repo` 清除歷史中的 data/ 影片、output/ 渲染、各代 binaries：**.git 從 426MB → 15MB**。
- 誤 commit 的 255MB `data/` 與 18MB `main` binary 已徹底移出版控。
- 改寫前完整備份：`../AgenticDub-pre-filter-repo-2026-07-09.bundle`（316MB），穩定後可刪。
- 已 force push；其他機器的舊 clone 需重新 clone。

### 3. 現行架構

```
主管線    cmd/server (Gin HTTP :8899) ← cmd/mcp (stdio MCP proxy)
          下載 → STT → LLM 翻譯 → 翻譯稽核 → HITL 90% 暫停 → TTS → 混音 → 字幕燒錄
          + SQLite task state (internal/service/task_state.go)
Legacy    cmd/cli run → internal/service/legacy_cli_pipeline.go（直連 aiark）
審核層    scripts/build_subtitle_review.py + subtitle_review_viewer.html
          + build_semantic_blocks.py + rewrite_review_blocks.py + FrameCue skill
```

### 4. 新增腳本（補齊 OpenClaw 筆記的「未完成」缺口）

- `scripts/export_review_srt.py` — FrameCue 人工編輯後的 package → 正式 SRT。
  ```bash
  python3 scripts/export_review_srt.py edited_review_package.json [-o out.srt] [--bilingual] [--keep-empty]
  ```
  跳過被清空的 cue、時間異常自動 clamp、輸出統計摘要。
- `scripts/stt_audit_compare.py` — 逐句 TTS 稽核比對。
  ```bash
  python3 scripts/stt_audit_compare.py --expected review_package.json \
    --transcripts cue_transcripts.json --glossary glossary.json -o audit_report.json
  ```
  相似度門檻（預設 0.85）+ glossary 保護詞強制檢查（跨大小寫/空格，`OpenClaw` 唸成 `open claw` 算過、`open cloud` 算錯）。exit 1 時讀 `failed_ids` 只重生失敗的 cue。

## Skills 使用方式

### `agenticdub`（全域，~/.claude/skills/agenticdub/）

**用途**：任何目錄下，給一支影片檔做同步口譯配音。
**觸發**：對 Claude 說「幫這支影片配音」「做同步口譯」「dub this video」或 `/agenticdub`。

五階段（⛔ = 人工硬閘門）：

1. **轉錄+初翻** — server 提交 `local:/abs/path.mp4` 任務（xai profiles），跑到 HITL 90% 暫停，不 approve。
2. **口譯稿** — build_semantic_blocks.py 分段後 LLM 重寫：壓縮口語化、刪填充詞、術語保留英文、專有名詞正確性 > 直譯、中文可比英文晚 0.5–1s 進。
3. **人工審核 ⛔** — FrameCue 瀏覽器逐句檢視改字 → export_review_srt.py 出定稿 SRT。
4. **逐句 TTS + STT 稽核** — approve HITL、產音檔、每句轉錄回來用 stt_audit_compare.py 比對，只重生 fail 的 cue；量大可平行 fan-out。
5. **合成 QC ⛔** — 原音墊底混音、裁到原片長度、ffprobe 比對長度（不信 SRT 尾端時間）、檢查尾端無黑畫面、專有名詞發音，過了才發布。

鐵律：字幕未定稿不跑整部 TTS；稽核失敗只重生單 cue；glossary 只學人工確認過的詞。

### `fable-dispatch`（全域，~/.claude/skills/fable-dispatch/）

**用途**：大型多步驟任務的多代理派工模式。
**觸發**：「fable 主導」「指揮 codex」「派工」或 `/fable-dispatch`。

分工：Fable 拆解決策彙整、Codex 優先執行、Sonnet 並行偵察與後備。實測要點：Codex sandbox 無法寫 `.git/index`（git 操作留給主線）；codex-rescue 只回 task id，用 codex-companion.mjs 從主線輪詢。

### 專案內子 skills（$REPO/.agents/skills/）

- `video-translation-with-voice` — 引擎手冊：payload、HITL、多說話者配音、媒體收尾、Postiz 發布。
- `framecue` — 審核工具機制：review package schema、viewer 驗證、匯出格式。

`agenticdub` 是編排層，細節都指回這兩個子 skill，不重複維護。

## 未完成 / 後續

- OpenClaw 影片本身：人工修正版重跑 TTS → STT 稽核 → 最終影片（工具已備齊，流程照 agenticdub skill Phase 4–5）。
- `mcplize` branch 確認穩定後可刪（本地 + 遠端）。
- 備份 bundle 穩定後可刪。
- 口譯稿風格 prompt 在 skill 層跑熟後，再評估是否沉到 Go 管線。
