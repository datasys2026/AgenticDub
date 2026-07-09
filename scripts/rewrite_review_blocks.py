#!/usr/bin/env python3
import argparse
import json
from pathlib import Path


STYLE_RULES = [
    "以台灣科技訪談字幕風格重寫中文。",
    "不要逐字翻譯 filler；you know、like、I mean、yeah 只有必要時才保留。",
    "沒有獨立資訊量的短 cue 可以留空，意思併到前後 cue。",
    "先理解同一個 block 的英文，再把自然中文回填到各 cue。",
    "中文要短、自然、口語，不要像第三人播報。",
    "技術詞優先保留英文：Agent、MCP、Unix、CLI、APP、WhatsApp、podcast。",
    "專有名詞依 glossary，不要自行猜。",
]


def srt_time(seconds):
    ms = round(float(seconds) * 1000)
    h, rem = divmod(ms, 3_600_000)
    m, rem = divmod(rem, 60_000)
    s, ms = divmod(rem, 1000)
    return f"{h:02d}:{m:02d}:{s:02d},{ms:03d}"


def cue_text(cue, key):
    return (cue.get(key) or "").strip()


def build_blocks(cues, max_cues, max_gap, max_duration):
    blocks = []
    cur = []
    for cue in cues:
        if cur:
            gap = float(cue["start"]) - float(cur[-1]["end"])
            duration = float(cue["end"]) - float(cur[0]["start"])
            if len(cur) >= max_cues or gap > max_gap or duration > max_duration:
                blocks.append(cur)
                cur = []
        cur.append(cue)
    if cur:
        blocks.append(cur)

    out = []
    for i, rows in enumerate(blocks, 1):
        out.append({
            "id": i,
            "cue_ids": [row["id"] for row in rows],
            "start": rows[0]["start"],
            "end": rows[-1]["end"],
            "original_text": " ".join(cue_text(row, "original_text") for row in rows).strip(),
            "target_text": " ".join(cue_text(row, "text") for row in rows).strip(),
            "cues": [{
                "id": row["id"],
                "start": row["start"],
                "end": row["end"],
                "original_text": cue_text(row, "original_text"),
                "text": cue_text(row, "text"),
            } for row in rows],
        })
    return out


def load_rewrite_cues(path):
    if not path:
        return {}
    data = json.loads(Path(path).read_text(encoding="utf-8"))
    rows = data.get("cues", data) if isinstance(data, dict) else data
    return {int(row["id"]): row.get("text", "") for row in rows if "id" in row and "text" in row}


def write_srt(cues, path):
    lines = []
    for i, cue in enumerate(cues, 1):
        lines += [
            str(i),
            f"{srt_time(cue['start'])} --> {srt_time(cue['end'])}",
            cue.get("text", ""),
            "",
        ]
    path.write_text("\n".join(lines), encoding="utf-8")


def write_prompt(path, glossary):
    glossary_note = f"\nGlossary:\n{Path(glossary).read_text(encoding='utf-8')}\n" if glossary else ""
    path.write_text(
        "# Subtitle Block Rewrite\n\n"
        "Rewrite each block as natural Traditional Chinese subtitles, then return cue-level JSON.\n\n"
        "Rules:\n"
        + "\n".join(f"- {rule}" for rule in STYLE_RULES)
        + glossary_note
        + "\nReturn format:\n"
        '```json\n{"cues":[{"id":1,"text":"改寫後字幕"}]}\n```\n',
        encoding="utf-8",
    )


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--review-package")
    parser.add_argument("--out-dir")
    parser.add_argument("--apply-rewrites", default="")
    parser.add_argument("--glossary", default="")
    parser.add_argument("--max-cues", type=int, default=5)
    parser.add_argument("--max-gap", type=float, default=0.9)
    parser.add_argument("--max-duration", type=float, default=12.0)
    parser.add_argument("--self-check", action="store_true")
    args = parser.parse_args()

    if args.self_check:
        cues = [
            {"id": 1, "start": 0, "end": 1, "text": "你知道", "original_text": "you know,"},
            {"id": 2, "start": 1.1, "end": 2, "text": "我們有 AI", "original_text": "we have AI"},
            {"id": 3, "start": 4, "end": 5, "text": "下一段", "original_text": "next"},
        ]
        blocks = build_blocks(cues, 5, 0.9, 12)
        assert len(blocks) == 2
        assert blocks[0]["cue_ids"] == [1, 2]
        print("self-check ok")
        return
    if not args.review_package or not args.out_dir:
        parser.error("--review-package and --out-dir are required unless --self-check is used")

    package_path = Path(args.review_package)
    package = json.loads(package_path.read_text(encoding="utf-8"))
    rewrites = load_rewrite_cues(args.apply_rewrites)
    if rewrites:
        for cue in package["cues"]:
            if int(cue["id"]) in rewrites:
                cue["text"] = rewrites[int(cue["id"])]

    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)
    blocks = build_blocks(package["cues"], args.max_cues, args.max_gap, args.max_duration)
    block_package = {
        "workflow": "subtitle_block_rewrite_v1",
        "source_review_package": str(package_path.resolve()),
        "style_rules": STYLE_RULES,
        "blocks": blocks,
    }
    (out_dir / "block_rewrite_input.json").write_text(
        json.dumps(block_package, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    write_prompt(out_dir / "block_rewrite_prompt.md", args.glossary)
    if rewrites:
        (out_dir / "review_package.rewritten.json").write_text(
            json.dumps(package, ensure_ascii=False, indent=2) + "\n",
            encoding="utf-8",
        )
        write_srt(package["cues"], out_dir / "rewritten.srt")
    print(json.dumps({"blocks": len(blocks), "cues": len(package["cues"]), "out_dir": str(out_dir)}, ensure_ascii=False))


if __name__ == "__main__":
    main()
