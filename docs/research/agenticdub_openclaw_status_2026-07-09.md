# AgenticDub OpenClaw 專案現況

日期：2026-07-09

## 目前結論

OpenClaw 這支影片目前不應再使用最早的長版 TTS / 舊 draft。可繼續工作的基準是 `subtitle_review_retimed_clipped_v7`：它使用原始影片 `origin_video.mp4`、`retimed_v2/target_original_timing.srt`，並裁到原片實際長度，避免尾端約 18-20 秒黑畫面。

目前最重要的下一步不是直接重跑 TTS，而是先把人工修正版字幕匯出成正式 SRT，確認專有名詞與口語風格後，再用這份字幕重新進語音流程。

## 主要工作目錄

- 專案：`/Users/baochen10luo/PaultoDo/agents/AgenticDub`
- OpenClaw task：`/Users/baochen10luo/PaultoDo/agents/AgenticDub/tasks/youtube_4uzGDAoNOZc_zh_tw_2026-06-26`
- 人工修正版 package：`/Users/baochen10luo/.codex/attachments/8bf96dae-f41f-484e-b418-d246123b5934/edited_review_package (1).json`
- 目前可用預覽：`http://mac-mini.tail4227fa.ts.net:3067/index.html`

## 版本狀態

| 版本 | 狀態 | 說明 |
| --- | --- | --- |
| `video_with_tts.mp4` | 不建議使用 | 舊 TTS 版本，長度 `1516.400s`，不是目前要的同步版本。 |
| `retimed_v2/video_retimed_audio.mp4` | 可參考 | 長度 `1355.854s`，接近原片，但字幕內容仍需人工修正。 |
| `synced_v3/video_synced_audio.mp4` | 不建議作為最終版 | 長度 `1373.805s`，比原片多約 `17.9s`，有尾端空白問題。 |
| `subtitle_review_retimed_clipped_v7` | 目前工作基準 | 487 cues，裁到 `1355.886917s`，雙語字幕與逐段音檔可在瀏覽器檢查。 |

原始影片 `origin_video.mp4` 長度是 `1355.906032s`。

## 字幕檢視工具現況

已新增兩個工具檔：

- `/Users/baochen10luo/PaultoDo/agents/AgenticDub/scripts/build_subtitle_review.py`
- `/Users/baochen10luo/PaultoDo/agents/AgenticDub/scripts/subtitle_review_viewer.html`

目前功能：

- 偵測換幕並抽場景圖。
- 用瀏覽器逐句檢視字幕。
- 左右鍵切換 cue。
- 同時顯示原文與中文字幕。
- 可直接修改字幕文字。
- 每句可播放對應音檔。
- 可下載修正後的 JSON / SRT。
- 會標記可能有破音或專有名詞風險的 cue。

目前 v7 package：

- package：`tasks/youtube_4uzGDAoNOZc_zh_tw_2026-06-26/subtitle_review_retimed_clipped_v7/review_package.json`
- cues：487
- 最後一 cue：`1354.395 -> 1355.886917`
- 最後一 cue 文字：`而且我簡直超級期待`

## 人工修正版比對

你提供的人工修正版與 v7 baseline 比對結果：

- baseline cues：487
- edited cues：487
- 修改 cues：51
- 被清空的 cues：`50, 99, 116, 135, 138, 157, 159, 163, 233, 259, 470, 471`

人工修改的主要風格：

- 刪掉沒有資訊量的填充詞，例如「嗯」「你知道」「我是說」。
- 減少硬接的「而且」「然後」，讓句子更像真正口語。
- 保留技術詞英文，例如 `OpenClaw`、`MCP`、`Agent`。
- 專有名詞比直譯重要，例如 `OpenCloud` 改成 `OpenClaw`。
- 允許小幅補語境，例如 `我沒做那個（語音功能）`。
- 不是逐字翻譯，而是偏現場口譯後的整理版。

暫時不要直接學入 glossary 的疑似 typo / 待確認詞：

- `Whipser`：可能應為 `Whisper`
- `Antropic`：可能應為 `Anthropic`
- `Claudbot`：需確認是否為 `ClaudeBot` 或其他正式名
- `Molty` / `Multy`：目前不一致，需確認正式拼法

已確認詞：

- `OpenClaw` 不是 `OpenCLAW`
- TTS 發音應導成 `Open Claw`
- `Moltbook` 不是 `Multibot`
- `MCP` 不要誤成 `MCV`
- 避免使用容易唸錯的「驚訝」

## 目前 pipeline 判斷

這次流程已證明單純「照翻譯字幕產 TTS」品質不夠穩。比較可行的方向是「同步口譯稿」：

1. 先以原文語意為基準切成語意段。
2. 中文不是逐字翻，而是壓縮、整理、口語化。
3. 英文 cue 開始後約半秒到一秒，中文口語可開始進入，不必完全等英文結束。
4. 每段 TTS 產完後要做 STT audit。
5. audit 檢查語音是否真的唸出字幕文字，尤其是專有名詞與破音字。
6. 不合格 cue 只重生該段，不重跑整部。

現有 artifacts 代表之前嘗試：

- `interpreter_script_xai_sal_v*`：同步口譯腳本與 xAI Sal voice 測試。
- `interpreter_script_content_review_v8`：內容 review 版。
- `block_rewrite_from_paul_v1`：依人工口味做區塊重寫的方向。
- `framecue_openclaw_blocks_v1` / `framecue_rewritten_v1`：用 FrameCue 檢查區塊與畫面對應。

## 不要再踩的坑

- 不要再用 `video_with_tts.mp4` 或根目錄 `subtitle_*.wav` 當最新成果；那是舊長版。
- 不要用 `synced_v3/video_synced_audio.mp4` 當最終版；尾端多約 18 秒。
- 不要只看 SRT 尾端時間，因為部分 SRT 名義上會到 `00:22:53,806`，但實際影片約 `1355.9s`。
- 不要在未確認 glossary 前直接把 typo 學進規則。
- 不要在字幕尚未定稿前重跑整部 TTS，會浪費時間且難 debug。

## 建議下一步

1. 從人工修正版 package 匯出正式字幕：
   - 建議輸出：`tasks/youtube_4uzGDAoNOZc_zh_tw_2026-06-26/subtitle_review_retimed_clipped_v7/edited_subtitles_from_user.srt`
   - 空白 cue 建議在最終 SRT 中省略；若 TTS pipeline 需要 cue id 對齊，則 JSON 保留空白 cue。
2. 把人工修正風格整理成小型 style note / glossary。
3. 用修正版字幕重跑 TTS。
4. 對 TTS 音檔做 STT audit。
5. 只針對 audit fail 的 cue 重生音檔。
6. 合成影片並確認：
   - 長度接近 `1355.9s`
   - 尾端沒有黑畫面空白
   - 字幕與語音對齊
   - `OpenClaw`、`Moltbook`、`MCP` 等詞沒有唸錯

## 目前未完成

- 尚未將人工修正版 package 匯出成正式 SRT。
- 尚未把人工修正風格寫入可重複使用的 prompt / glossary。
- 尚未用人工修正版重跑 TTS。
- 尚未做新版 TTS 的 STT audit。
- 尚未產出可上傳 YouTube private 的最終影片。
