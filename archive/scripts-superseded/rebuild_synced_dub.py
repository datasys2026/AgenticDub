#!/usr/bin/env python3
import argparse
import concurrent.futures
import json
import re
import subprocess
from pathlib import Path


FFMPEG = "/opt/homebrew/opt/ffmpeg-full/bin/ffmpeg"
FFPROBE = "/opt/homebrew/bin/ffprobe"
SR = "24000"
WORKFLOW_NAME = "rebuild_synced_dub_v3"


def run(cmd):
    subprocess.run(cmd, check=True)


def duration(path):
    out = subprocess.check_output([
        FFPROBE,
        "-v",
        "error",
        "-show_entries",
        "format=duration",
        "-of",
        "default=nw=1:nk=1",
        str(path),
    ], text=True)
    return float(out.strip())


def parse_time(value):
    hh, mm, rest = value.replace(",", ".").split(":")
    return int(hh) * 3600 + int(mm) * 60 + float(rest)


def parse_srt(path):
    text = Path(path).read_text()
    pattern = re.compile(
        r"(?ms)^\s*(\d+)\s*\n"
        r"(\d{2}:\d{2}:\d{2}[,.]\d{3})\s*-->\s*"
        r"(\d{2}:\d{2}:\d{2}[,.]\d{3}).*?\n"
        r"(.*?)(?=\n\s*\d+\s*\n\d{2}:\d{2}:\d{2}[,.]\d{3}\s*-->|\Z)"
    )
    cues = []
    for m in pattern.finditer(text):
        start = parse_time(m.group(2))
        end = parse_time(m.group(3))
        cues.append((int(m.group(1)), start, end, " ".join(m.group(4).split())))
    return cues


def atempo_chain(speed):
    parts = []
    while speed > 2.0:
        parts.append("atempo=2.0")
        speed /= 2.0
    while speed < 0.5:
        parts.append("atempo=0.5")
        speed /= 0.5
    parts.append(f"atempo={speed:.6f}")
    return ",".join(parts)


def make_silence(path, seconds):
    run([
        FFMPEG,
        "-hide_banner",
        "-loglevel",
        "error",
        "-y",
        "-f",
        "lavfi",
        "-i",
        f"anullsrc=channel_layout=mono:sample_rate={SR}",
        "-t",
        f"{seconds:.3f}",
        "-c:a",
        "pcm_s16le",
        str(path),
    ])


def make_segment(item):
    task, work, idx, start, end = item
    src = task / f"subtitle_{idx}.wav"
    dst = work / f"seg_{idx:04d}.wav"
    target = max(0.05, end - start)
    if dst.exists() and abs(duration(dst) - target) < 0.02:
        src_duration = duration(src)
        return {
            "index": idx,
            "cue_duration": target,
            "tts_duration": src_duration,
            "speed": src_duration / target if src_duration > target else 1.0,
            "file": str(dst),
        }
    src_duration = duration(src)
    filters = []
    if src_duration > target:
        filters.append(atempo_chain(src_duration / target))
    filters.extend([f"apad", f"atrim=0:{target:.3f}", "asetpts=N/SR/TB"])
    run([
        FFMPEG,
        "-hide_banner",
        "-loglevel",
        "error",
        "-y",
        "-i",
        str(src),
        "-af",
        ",".join(filters),
        "-ar",
        SR,
        "-ac",
        "1",
        "-c:a",
        "pcm_s16le",
        str(dst),
    ])
    return {
        "index": idx,
        "cue_duration": target,
        "tts_duration": src_duration,
        "speed": src_duration / target if src_duration > target else 1.0,
        "file": str(dst),
    }


def concat_audio(parts, out, cwd):
    list_file = cwd / "concat.txt"
    list_file.write_text("".join(f"file '{Path(p).name}'\n" for p in parts))
    run([
        FFMPEG,
        "-hide_banner",
        "-loglevel",
        "error",
        "-y",
        "-f",
        "concat",
        "-safe",
        "0",
        "-i",
        str(list_file),
        "-ar",
        SR,
        "-ac",
        "1",
        "-c:a",
        "pcm_s16le",
        str(out),
    ])


def choose_srt(task):
    candidates = [
        task / "final.srt",
        task / "retimed_v2" / "target_original_timing.srt",
    ]
    for candidate in candidates:
        if not candidate.exists():
            continue
        cues = parse_srt(candidate)
        if cues and all((task / f"subtitle_{idx}.wav").exists() for idx, _, _, _ in cues):
            return candidate, cues
    raise SystemExit(f"{task}: no SRT matches existing subtitle_N.wav files")


def write_manifest(work, report, quality_status):
    manifest = {
        "workflow": WORKFLOW_NAME,
        "quality_status": quality_status,
        "script": "scripts/rebuild_synced_dub.py",
        "inputs": [
            "origin_video.mp4",
            report["srt"],
            "subtitle_*.wav",
        ],
        "outputs": [
            "synced_v3/tts_synced_per_cue.wav",
            "synced_v3/video_synced_audio.mp4",
            report["final"],
            "synced_v3/sync_report.json",
        ],
        "settings": {
            "sample_rate": int(SR),
            "original_audio_volume": 0.18,
            "dubbed_audio_volume": 1.0,
            "subtitle_style": "FontSize=16,PrimaryColour=&HFFFFFF&,Outline=2,Shadow=2",
        },
        "quality_flags": {
            "speed_over_1_5x": report["speed_over_1_5x"],
            "max_speed": report["max_speed"],
            "extra_video": report["extra_video"],
        },
    }
    (work / "workflow.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n")


def rebuild(task_dir, out_dir, workers):
    task = Path(task_dir)
    out_dir = Path(out_dir)
    name = task.name.replace("youtube_", "").replace("_zh_tw_2026-06-26", "")
    work = task / "synced_v3"
    work.mkdir(exist_ok=True)
    srt_path, cues = choose_srt(task)

    missing = [idx for idx, _, _, _ in cues if not (task / f"subtitle_{idx}.wav").exists()]
    if missing:
        raise SystemExit(f"{task}: missing subtitle wavs: {missing[:10]}")

    jobs = [(task, work, idx, start, end) for idx, start, end, _ in cues]
    with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as pool:
        stats = list(pool.map(make_segment, jobs))

    parts = []
    cursor = 0.0
    for idx, start, end, _ in cues:
        if start > cursor + 0.01:
            silence = work / f"gap_{idx:04d}.wav"
            make_silence(silence, start - cursor)
            parts.append(silence)
        parts.append(work / f"seg_{idx:04d}.wav")
        cursor = max(cursor, end)

    video_duration = duration(task / "origin_video.mp4")
    if video_duration > cursor + 0.01:
        tail = work / "tail.wav"
        make_silence(tail, video_duration - cursor)
        parts.append(tail)

    synced_wav = work / "tts_synced_per_cue.wav"
    concat_audio(parts, synced_wav, work)

    mixed = work / "video_synced_audio.mp4"
    run([
        FFMPEG,
        "-hide_banner",
        "-loglevel",
        "error",
        "-y",
        "-i",
        str(task / "origin_video.mp4"),
        "-i",
        str(synced_wav),
        "-filter_complex",
        "[0:a]volume=0.18[a0];[1:a]volume=1.0[a1];[a0][a1]amix=inputs=2:duration=longest:normalize=0[a]",
        "-map",
        "0:v",
        "-map",
        "[a]",
        "-c:v",
        "copy",
        "-c:a",
        "aac",
        "-b:a",
        "192k",
        str(mixed),
    ])

    final = out_dir / f"{name}_synced_per_cue_subbed_720p_2026-06-30.mp4"
    extra_video = max(0.0, duration(synced_wav) - video_duration)
    subtitle_style = "FontSize=16,PrimaryColour=&HFFFFFF&,Outline=2,Shadow=2"
    video_filter = f"scale=1280:-2,subtitles='{srt_path}':force_style='{subtitle_style}'"
    if extra_video > 0.01:
        video_filter = f"scale=1280:-2,tpad=stop_mode=clone:stop_duration={extra_video:.3f},subtitles='{srt_path}':force_style='{subtitle_style}'"
    run([
        FFMPEG,
        "-hide_banner",
        "-loglevel",
        "error",
        "-y",
        "-i",
        str(mixed),
        "-vf",
        video_filter,
        "-c:v",
        "libx264",
        "-preset",
        "medium",
        "-crf",
        "22",
        "-c:a",
        "copy",
        "-movflags",
        "+faststart",
        str(final),
    ])

    report = {
        "task": str(task),
        "cue_count": len(cues),
        "srt": str(srt_path),
        "video_duration": video_duration,
        "synced_audio_duration": duration(synced_wav),
        "extra_video": extra_video,
        "final": str(final),
        "speed_over_1_5x": sum(1 for row in stats if row["speed"] > 1.5),
        "max_speed": max(row["speed"] for row in stats),
    }
    (work / "sync_report.json").write_text(json.dumps(report, ensure_ascii=False, indent=2))
    write_manifest(work, report, "needs_revision")
    print(json.dumps(report, ensure_ascii=False))


def self_check():
    report = {
        "srt": "tasks/example/final.srt",
        "final": "output/example.mp4",
        "speed_over_1_5x": 3,
        "max_speed": 2.25,
        "extra_video": 0.5,
    }
    tmp = Path("/tmp/rebuild_synced_dub_manifest_check")
    tmp.mkdir(exist_ok=True)
    write_manifest(tmp, report, "needs_revision")
    manifest = json.loads((tmp / "workflow.json").read_text())
    assert manifest["workflow"] == WORKFLOW_NAME
    assert manifest["quality_status"] == "needs_revision"
    assert manifest["quality_flags"]["max_speed"] == 2.25


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("task_dir", nargs="*")
    parser.add_argument("--out-dir", default="output")
    parser.add_argument("--workers", type=int, default=8)
    parser.add_argument("--self-check", action="store_true")
    args = parser.parse_args()
    if args.self_check:
        self_check()
        return
    if not args.task_dir:
        parser.error("task_dir is required unless --self-check is used")
    for task_dir in args.task_dir:
        rebuild(task_dir, args.out_dir, args.workers)


if __name__ == "__main__":
    main()
