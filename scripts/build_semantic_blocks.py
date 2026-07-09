#!/usr/bin/env python3
import argparse
import hashlib
import json
import shutil
import subprocess
from pathlib import Path


WORKFLOW = "semantic_blocks_v1"


STYLE_RULES = [
    "Translate by meaning block, not by isolated subtitle cue.",
    "Rewrite as natural Traditional Chinese for a tech interview.",
    "Drop filler fragments when they do not carry meaning.",
    "Keep protected technical names in display text, e.g. OpenClaw, MCP, Unix, CLI.",
    "Use speech_text for pronunciation hints, e.g. OpenClaw can become Open Claw.",
]


def ms(seconds):
    return round(float(seconds) * 1000)


def seconds(value):
    return float(value) / 1000 if isinstance(value, int) and value > 10_000 else float(value)


def text(row, *keys):
    for key in keys:
        value = row.get(key)
        if value:
            return " ".join(str(value).split())
    return ""


def checksum(*parts):
    return hashlib.sha1("\n".join(parts).encode("utf-8")).hexdigest()


def ffprobe_duration_ms(path):
    if not path or not shutil.which("ffprobe"):
        return 0
    video = Path(path)
    if not video.exists():
        return 0
    out = subprocess.check_output([
        "ffprobe", "-v", "error",
        "-show_entries", "format=duration",
        "-of", "default=nw=1:nk=1",
        str(video),
    ], text=True)
    return ms(float(out.strip()))


def load_rows(path):
    data = json.loads(Path(path).read_text(encoding="utf-8"))
    if isinstance(data, dict) and "cues" in data:
        rows = data["cues"]
        source_type = "review_package"
        video = data.get("video", "")
    elif isinstance(data, list):
        rows = data
        source_type = "segments"
        video = ""
    else:
        raise SystemExit(f"unsupported input shape: {path}")

    cues = []
    for i, row in enumerate(rows, 1):
        start = seconds(row["start"])
        end = seconds(row["end"])
        cues.append({
            "id": int(row.get("id", i)),
            "start_ms": ms(start),
            "end_ms": ms(end),
            "speaker": row.get("speaker", ""),
            "scene_id": row.get("scene_id"),
            "source_text": text(row, "original_text", "source_text"),
            "raw_source_cue_ids": row.get("source_cue_ids", []),
            "target_text": text(row, "text", "target_text"),
            "speech_text": text(row, "speech_text", "text", "target_text"),
        })
    return source_type, video, cues


def enrich_source_text(cues, source_package):
    if not source_package:
        return
    _, _, source_cues = load_rows(source_package)
    for cue in cues:
        if cue["source_text"]:
            continue
        overlapping = [
            row for row in source_cues
            if row["start_ms"] < cue["end_ms"] and row["end_ms"] > cue["start_ms"]
        ]
        cue["source_text"] = " ".join(row["source_text"] for row in overlapping if row["source_text"]).strip()
        cue["raw_source_cue_ids"] = [row["id"] for row in overlapping]


def should_break(cur, cue, max_cues, max_gap_ms, max_duration_ms):
    if not cur:
        return False
    if len(cur) >= max_cues:
        return True
    if cue["start_ms"] - cur[-1]["end_ms"] > max_gap_ms:
        return True
    if cue["end_ms"] - cur[0]["start_ms"] > max_duration_ms:
        return True
    return bool(cur[-1]["speaker"] and cue["speaker"] and cur[-1]["speaker"] != cue["speaker"])


def build_blocks(cues, args, video_duration_ms):
    groups = []
    cur = []
    for cue in cues:
        if should_break(cur, cue, args.max_cues, ms(args.max_gap), ms(args.max_duration)):
            groups.append(cur)
            cur = []
        cur.append(cue)
    if cur:
        groups.append(cur)

    blocks = []
    for i, rows in enumerate(groups, 1):
        source_text = " ".join(row["source_text"] for row in rows if row["source_text"]).strip()
        target_text = " ".join(row["target_text"] for row in rows if row["target_text"]).strip()
        speech_text = " ".join(row["speech_text"] for row in rows if row["speech_text"]).strip()
        end_ms = rows[-1]["end_ms"]
        flags = []
        if video_duration_ms and end_ms > video_duration_ms:
            flags.append("over_video_end")
        if not target_text:
            flags.append("empty_target")
        blocks.append({
            "id": f"b{i:04d}",
            "speaker": next((row["speaker"] for row in rows if row["speaker"]), ""),
            "start_ms": rows[0]["start_ms"],
            "end_ms": end_ms,
            "budget_ms": max(0, end_ms - rows[0]["start_ms"]),
            "input_ids": [row["id"] for row in rows],
            "source_cue_ids": sorted({
                source_id
                for row in rows
                for source_id in (row["raw_source_cue_ids"] or [row["id"]])
            }),
            "source_text": source_text,
            "target_text": target_text,
            "speech_text": speech_text,
            "subtitle_cues": [{
                "id": row["id"],
                "start_ms": row["start_ms"],
                "end_ms": row["end_ms"],
                "source_cue_ids": row["raw_source_cue_ids"] or [row["id"]],
                "source_text": row["source_text"],
                "text": row["target_text"],
            } for row in rows],
            "tts": {
                "voice": args.voice,
                "rate": 1.0,
                "synthesized_ms": None,
                "trimmed_silence_ms": None,
                "fit": "pending",
                "attempts": 0,
            },
            "checksum": checksum(target_text, speech_text),
            "flags": flags,
        })
    return blocks


def write_prompt(path):
    path.write_text(
        "# Semantic Block Rewrite\n\n"
        "Rewrite each block as a simultaneous interpreter would: preserve meaning, reorder naturally for Chinese, and keep timing concise.\n\n"
        "Rules:\n"
        + "\n".join(f"- {rule}" for rule in STYLE_RULES)
        + "\n\nReturn JSON with edited blocks:\n"
        '```json\n{"blocks":[{"id":"b0001","target_text":"...","speech_text":"..."}]}\n```\n',
        encoding="utf-8",
    )


def package(input_path, args):
    source_type, video, cues = load_rows(input_path)
    enrich_source_text(cues, args.source_package)
    video_duration_ms = args.video_duration_ms or ffprobe_duration_ms(args.video or video)
    if not video_duration_ms:
        video_duration_ms = max(cue["end_ms"] for cue in cues)
    blocks = build_blocks(cues, args, video_duration_ms)
    last_end_ms = max(block["end_ms"] for block in blocks)
    return {
        "version": 1,
        "workflow": WORKFLOW,
        "source": str(Path(input_path).resolve()),
        "source_type": source_type,
        "source_lang": args.source_lang,
        "target_lang": args.target_lang,
        "video": str(Path(args.video or video).resolve()) if args.video or video else "",
        "video_duration_ms": video_duration_ms,
        "settings": {
            "max_cues": args.max_cues,
            "max_gap_ms": ms(args.max_gap),
            "max_duration_ms": ms(args.max_duration),
            "voice": args.voice,
            "timing_policy": "absolute_block_anchors",
        },
        "style_rules": STYLE_RULES,
        "blocks": blocks,
        "tail": {
            "last_block_end_ms": last_end_ms,
            "video_duration_ms": video_duration_ms,
            "overrun_ms": max(0, last_end_ms - video_duration_ms),
            "pad_to_video_end": True,
        },
        "validation": {
            "cue_count": len(cues),
            "block_count": len(blocks),
            "empty_target_blocks": sum("empty_target" in block["flags"] for block in blocks),
            "over_video_end_blocks": sum("over_video_end" in block["flags"] for block in blocks),
        },
    }


def self_check():
    tmp = Path("/tmp/semantic_blocks_check.json")
    tmp.write_text(json.dumps([
        {"id": 1, "start": 0, "end": 1, "text": "你知道", "original_text": "you know"},
        {"id": 2, "start": 1.1, "end": 2, "text": "我們有 AI", "original_text": "we have AI"},
        {"id": 3, "start": 4, "end": 5, "text": "下一段", "original_text": "next"},
    ]), encoding="utf-8")
    args = argparse.Namespace(
        max_cues=5,
        max_gap=0.9,
        max_duration=12.0,
        video_duration_ms=5000,
        video="",
        source_package="",
        voice="xai-sal-clone-v1",
        source_lang="en",
        target_lang="zh-TW",
    )
    data = package(tmp, args)
    assert data["validation"]["block_count"] == 2
    assert data["blocks"][0]["source_cue_ids"] == [1, 2]
    assert data["tail"]["overrun_ms"] == 0
    print("self-check ok")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input")
    parser.add_argument("--source-package", default="")
    parser.add_argument("--out-dir")
    parser.add_argument("--video", default="")
    parser.add_argument("--video-duration-ms", type=int, default=0)
    parser.add_argument("--source-lang", default="en")
    parser.add_argument("--target-lang", default="zh-TW")
    parser.add_argument("--voice", default="xai-sal-clone-v1")
    parser.add_argument("--max-cues", type=int, default=5)
    parser.add_argument("--max-gap", type=float, default=0.9)
    parser.add_argument("--max-duration", type=float, default=12.0)
    parser.add_argument("--self-check", action="store_true")
    args = parser.parse_args()
    if args.self_check:
        self_check()
        return
    if not args.input or not args.out_dir:
        parser.error("--input and --out-dir are required unless --self-check is used")
    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)
    data = package(args.input, args)
    (out_dir / "semantic_blocks.json").write_text(
        json.dumps(data, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    write_prompt(out_dir / "semantic_block_rewrite_prompt.md")
    print(json.dumps({
        "blocks": data["validation"]["block_count"],
        "cues": data["validation"]["cue_count"],
        "tail_overrun_ms": data["tail"]["overrun_ms"],
        "out_dir": str(out_dir),
    }, ensure_ascii=False))


if __name__ == "__main__":
    main()
