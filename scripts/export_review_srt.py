#!/usr/bin/env python3
import argparse
import json
import sys
from pathlib import Path


def srt_time(seconds):
    ms = round(float(seconds) * 1000)
    h, rem = divmod(ms, 3_600_000)
    m, rem = divmod(rem, 60_000)
    s, ms = divmod(rem, 1000)
    return f"{h:02d}:{m:02d}:{s:02d},{ms:03d}"


def cue_text(cue, key):
    return (cue.get(key) or "").strip()


def write_srt(cues, path, keep_empty, bilingual):
    lines = []
    skipped_empty = 0
    exported = 0
    for cue in sorted(cues, key=lambda row: float(row.get("start", 0))):
        text = cue_text(cue, "text")
        if not text and not keep_empty:
            skipped_empty += 1
            continue
        start = float(cue["start"])
        end = float(cue["end"])
        if end <= start:
            end = start + 0.5
            print(f"warning: cue {cue.get('id')} end <= start; clamped to {end:.3f}", file=sys.stderr)

        body = [text]
        original = cue_text(cue, "original_text")
        if bilingual and original:
            body.append(original)
        exported += 1
        lines += [
            str(exported),
            f"{srt_time(start)} --> {srt_time(end)}",
            *body,
            "",
        ]
    path.write_text("\n".join(lines), encoding="utf-8")
    return exported, skipped_empty


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("package")
    parser.add_argument("-o", "--output")
    parser.add_argument("--keep-empty", action="store_true")
    parser.add_argument("--bilingual", action="store_true")
    args = parser.parse_args()

    package_path = Path(args.package)
    package = json.loads(package_path.read_text(encoding="utf-8"))
    output = Path(args.output) if args.output else package_path.parent / "edited_subtitles.srt"
    output.parent.mkdir(parents=True, exist_ok=True)
    cues = package.get("cues", [])
    exported, skipped_empty = write_srt(cues, output, args.keep_empty, args.bilingual)
    print(f"total cues: {len(cues)}")
    print(f"exported count: {exported}")
    print(f"skipped-empty count: {skipped_empty}")
    print(f"output path: {output}")


if __name__ == "__main__":
    main()
