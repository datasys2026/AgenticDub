#!/usr/bin/env python3
import argparse
import json
import os
import re
import shutil
import subprocess
import urllib.request
from difflib import SequenceMatcher
from pathlib import Path


FFMPEG = "/opt/homebrew/opt/ffmpeg-full/bin/ffmpeg"
FFPROBE = "/opt/homebrew/bin/ffprobe"
SR = "24000"
WORKFLOW = "interpreter_preview_v1"


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


def fmt_time(seconds):
    ms = round(seconds * 1000)
    h, rem = divmod(ms, 3_600_000)
    m, rem = divmod(rem, 60_000)
    s, ms = divmod(rem, 1000)
    return f"{h:02d}:{m:02d}:{s:02d},{ms:03d}"


def load_segments(path):
    segments = json.loads(Path(path).read_text())
    for i, seg in enumerate(segments, 1):
        for key in ("id", "start", "end", "text"):
            if key not in seg:
                raise SystemExit(f"{path}: segment {i} missing {key}")
    return segments


def load_glossary(path):
    if not path:
        return {}
    data = json.loads(Path(path).read_text())
    terms = data.get("terms", {})
    if isinstance(terms, dict):
        return terms
    return {row["source"]: row["target"] for row in terms if "source" in row and "target" in row}


def apply_glossary(segments, terms):
    if not terms:
        return segments
    updated = []
    for seg in segments:
        item = dict(seg)
        text = item["text"]
        for source, target in terms.items():
            text = text.replace(source, target)
        item["text"] = text
        if "speech_text" in item:
            speech_text = item["speech_text"]
            for source, target in terms.items():
                speech_text = speech_text.replace(source, target)
            item["speech_text"] = speech_text
        updated.append(item)
    return updated


def tts_request(text, voice_id, model, api_key, out_mp3, out_meta):
    payload = json.dumps({
        "model": model,
        "input": text,
        "voice_id": voice_id,
        "language": "Chinese",
        "response_format": "mp3",
    }).encode()
    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
        "User-Agent": "curl/8.7.1",
    }
    req = urllib.request.Request(
        "https://aiark.com.tw/tts/v1/audio/speech",
        data=payload,
        method="POST",
        headers=headers,
    )
    with urllib.request.urlopen(req, timeout=180) as resp:
        meta = json.loads(resp.read().decode())
    out_meta.write_text(json.dumps(meta, ensure_ascii=False, indent=2) + "\n")
    req = urllib.request.Request(
        "https://aiark.com.tw/tts" + meta["file"],
        headers={"Authorization": f"Bearer {api_key}", "User-Agent": "curl/8.7.1"},
    )
    with urllib.request.urlopen(req, timeout=120) as resp:
        out_mp3.write_bytes(resp.read())


def prepare_tts(segments, out_dir, voice_id, model, api_key, reuse_dir):
    reuse_text = {}
    if reuse_dir and (reuse_dir / "segments.json").exists():
        old_segments = load_segments(reuse_dir / "segments.json")
        reuse_text = {
            int(seg["id"]): seg.get("speech_text", seg["text"])
            for seg in old_segments
        }
    for seg in segments:
        idx = int(seg["id"])
        mp3 = out_dir / f"tts_{idx:03d}.mp3"
        meta = out_dir / f"tts_{idx:03d}.json"
        if mp3.exists() and mp3.stat().st_size > 1000:
            continue
        speech_text = seg.get("speech_text", seg["text"])
        if reuse_dir:
            src_mp3 = reuse_dir / mp3.name
            src_meta = reuse_dir / meta.name
            if src_mp3.exists() and reuse_text.get(idx) == speech_text:
                shutil.copy2(src_mp3, mp3)
                if src_meta.exists():
                    shutil.copy2(src_meta, meta)
                continue
        if not api_key:
            raise SystemExit("TTS API key missing; pass --api-key or set TTS_API_KEY")
        tts_request(speech_text, voice_id, model, api_key, mp3, meta)
        print(f"tts {idx}", flush=True)


def make_silence(path, seconds):
    run([
        FFMPEG, "-hide_banner", "-loglevel", "error", "-y",
        "-f", "lavfi", "-i", f"anullsrc=channel_layout=mono:sample_rate={SR}",
        "-t", f"{seconds:.3f}", "-c:a", "pcm_s16le", str(path),
    ])


def build_audio_and_srt(task, out_dir, segments, lag):
    parts = []
    cue_times = []
    cursor = 0.0
    for seg in segments:
        idx = int(seg["id"])
        src = out_dir / f"tts_{idx:03d}.mp3"
        wav = out_dir / f"tts_{idx:03d}.wav"
        run([
            FFMPEG, "-hide_banner", "-loglevel", "error", "-y",
            "-i", str(src), "-ar", SR, "-ac", "1", "-c:a", "pcm_s16le", str(wav),
        ])
        d = duration(wav)
        # ponytail: one lag knob; upgrade to per-speaker/per-density timing when this stops holding.
        start = max(float(seg["start"]) + lag, cursor + 0.05)
        end = start + d
        if start > cursor + 0.01:
            gap = out_dir / f"gap_{idx:03d}.wav"
            make_silence(gap, start - cursor)
            parts.append(gap)
        parts.append(wav)
        cursor = end
        cue_times.append({
            "id": idx,
            "source_start": float(seg["start"]),
            "source_end": float(seg["end"]),
            "start": start,
            "end": end,
            "duration": d,
            "lag_from_source_start": start - float(seg["start"]),
            "text": seg["text"],
            "speech_text": seg.get("speech_text", seg["text"]),
        })

    list_file = out_dir / "concat.txt"
    list_file.write_text("".join(f"file '{p.name}'\n" for p in parts))
    audio = out_dir / "interpreter_audio.wav"
    run([
        FFMPEG, "-hide_banner", "-loglevel", "error", "-y",
        "-f", "concat", "-safe", "0", "-i", str(list_file),
        "-ar", SR, "-ac", "1", "-c:a", "pcm_s16le", str(audio),
    ])

    srt = []
    for row in cue_times:
        srt += [
            str(row["id"]),
            f"{fmt_time(row['start'])} --> {fmt_time(row['end'])}",
            row["text"],
            "",
        ]
    (out_dir / "interpreter.srt").write_text("\n".join(srt), encoding="utf-8")
    return audio, cue_times


def build_video(task, out_dir, output_name, audio, subtitle_alpha, original_volume, clip_duration):
    video_duration = duration(task / "origin_video.mp4")
    audio_duration = duration(audio)
    clip_duration = min(video_duration, clip_duration)
    extra = max(0.0, audio_duration - clip_duration)
    raw = out_dir / "preview_audio_mix.mp4"
    vf = "scale=1280:-2"
    if extra > 0.01:
        vf = f"scale=1280:-2,tpad=stop_mode=clone:stop_duration={extra:.3f}"
    run([
        FFMPEG, "-hide_banner", "-loglevel", "error", "-y",
        "-t", f"{clip_duration:.3f}", "-i", str(task / "origin_video.mp4"),
        "-i", str(audio),
        "-filter_complex",
        f"[0:a]volume={original_volume}[a0];[1:a]volume=1.0[a1];[a0][a1]amix=inputs=2:duration=longest:normalize=0[a]",
        "-map", "0:v", "-map", "[a]", "-vf", vf,
        "-c:v", "libx264", "-preset", "veryfast", "-crf", "22",
        "-c:a", "aac", "-b:a", "192k", str(raw),
    ])
    final = out_dir / output_name
    # ASS alpha is inverse opacity; 0x99 gives a simple semi-transparent black box.
    style = (
        "FontName=Arial,FontSize=15,PrimaryColour=&H00FFFFFF,"
        f"BackColour=&H{subtitle_alpha}000000,BorderStyle=3,Outline=4,Shadow=0,MarginV=34"
    )
    run([
        FFMPEG, "-hide_banner", "-loglevel", "error", "-y",
        "-i", str(raw),
        "-vf", f"subtitles='{out_dir / 'interpreter.srt'}':force_style='{style}'",
        "-c:v", "libx264", "-preset", "veryfast", "-crf", "22",
        "-c:a", "copy", "-movflags", "+faststart", str(final),
    ])
    return final


SIMPLIFIED = {
    "躲起來": "躲起来", "雲端": "云端", "萬": "万", "顆": "颗",
    "電腦": "电脑", "這": "这", "個": "个", "為": "为",
    "機": "机", "網": "网", "專": "专", "實": "实",
    "現": "现", "對": "对", "從": "从", "過": "过",
    "週": "周", "檔": "档", "裡": "里", "開": "开",
    "創": "创", "讓": "让", "燈": "灯", "溫": "温",
    "搜尋": "搜寻", "它": "他", "兩": "两",
}


def normalize_text(text):
    text = text.lower()
    for source, target in SIMPLIFIED.items():
        text = text.replace(source, target)
    return re.sub(r"[^0-9a-z\u4e00-\u9fff]+", "", text)


def audit_stt(out_dir, segments, threshold):
    try:
        from faster_whisper import WhisperModel
    except Exception as err:
        return {"status": "blocked", "error": f"faster_whisper unavailable: {err}"}

    model = WhisperModel("base", device="cpu", compute_type="int8")
    rows = []
    for seg in segments:
        idx = int(seg["id"])
        mp3 = out_dir / f"tts_{idx:03d}.mp3"
        stt_segments, _ = model.transcribe(str(mp3), language="zh", vad_filter=False, beam_size=5)
        got = "".join(s.text.strip() for s in stt_segments)
        expected = seg.get("speech_text", seg["text"])
        ratio = SequenceMatcher(None, normalize_text(expected), normalize_text(got)).ratio() if got else 0.0
        rows.append({
            "id": idx,
            "expected": expected,
            "display_text": seg["text"],
            "stt": got,
            "ratio": round(ratio, 3),
            "needs_review": ratio < threshold,
            "audio": str(mp3),
        })
    audit = {
        "status": "completed_per_cue",
        "model": "faster-whisper base cpu int8",
        "threshold": threshold,
        "summary": {
            "cue_count": len(rows),
            "needs_review_count": sum(row["needs_review"] for row in rows),
            "lowest_ratios": sorted(rows, key=lambda row: row["ratio"])[:10],
        },
        "cue_checks": rows,
    }
    (out_dir / "stt_audit_per_cue.json").write_text(json.dumps(audit, ensure_ascii=False, indent=2) + "\n")
    return audit


def write_report(out_dir, args, segments, cue_times, audio, final, audit):
    report = {
        "preview": str(final),
        "interpreter_audio": str(audio),
        "preview_duration": duration(final),
        "audio_duration": duration(audio),
        "segments": cue_times,
    }
    (out_dir / "media_report.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    outputs = [
        "segments.json",
        "interpreter.srt",
        "interpreter_audio.wav",
        final.name,
        "media_report.json",
    ]
    if audit.get("status") == "completed_per_cue":
        outputs.append("stt_audit_per_cue.json")
    manifest = {
        "workflow": WORKFLOW,
        "quality_status": "prototype",
        "script": "scripts/build_interpreter_preview.py",
        "voice_requested": args.voice_id,
        "tts_model": args.tts_model,
        "timing_policy": f"start=max(source_start+{args.lag}s, previous_chinese_end+0.05s)",
        "glossary": args.glossary,
        "outputs": outputs,
        "quality_flags": {
            "preview_duration": report["preview_duration"],
            "audio_duration": report["audio_duration"],
            "segment_count": len(segments),
            "average_lag_from_source_start": sum(row["lag_from_source_start"] for row in cue_times) / len(cue_times),
            "max_lag_from_source_start": max(row["lag_from_source_start"] for row in cue_times),
        },
        "stt_audit_per_cue": audit if audit.get("status") != "completed_per_cue" else {
            "status": audit["status"],
            "model": audit["model"],
            "needs_review_count": audit["summary"]["needs_review_count"],
            "threshold": audit["threshold"],
        },
    }
    (out_dir / "workflow.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps(manifest, ensure_ascii=False, indent=2))


def build(args):
    task = Path(args.task)
    out_dir = task / args.out_dir
    out_dir.mkdir(parents=True, exist_ok=True)
    segments = apply_glossary(load_segments(args.segments), load_glossary(args.glossary))
    (out_dir / "segments.json").write_text(json.dumps(segments, ensure_ascii=False, indent=2) + "\n")
    api_key = args.api_key or os.environ.get(args.api_key_env, "")
    prepare_tts(segments, out_dir, args.voice_id, args.tts_model, api_key, Path(args.reuse_tts_dir) if args.reuse_tts_dir else None)
    audio, cue_times = build_audio_and_srt(task, out_dir, segments, args.lag)
    clip_duration = args.preview_duration or max(float(seg["end"]) for seg in segments)
    final = build_video(task, out_dir, args.output_name, audio, args.subtitle_alpha, args.original_volume, clip_duration)
    audit = {"status": "skipped"}
    if not args.no_stt_audit:
        audit = audit_stt(out_dir, segments, args.stt_threshold)
    write_report(out_dir, args, segments, cue_times, audio, final, audit)


def self_check():
    assert fmt_time(1.234) == "00:00:01,234"
    assert normalize_text("這一兩週") == normalize_text("这一两周")
    segs = [{"id": 1, "start": 0, "end": 1, "text": "AI agent"}]
    assert apply_glossary(segs, {"AI agent": "AI 代理"})[0]["text"] == "AI 代理"
    assert max(float(seg["end"]) for seg in segs) == 1.0


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--task")
    parser.add_argument("--segments")
    parser.add_argument("--out-dir", default="interpreter_preview")
    parser.add_argument("--output-name", default="interpreter_preview.mp4")
    parser.add_argument("--voice-id", default="xai-sal-clone-v1")
    parser.add_argument("--tts-model", default="aiark/qwen3-tts-1.7b-base")
    parser.add_argument("--api-key", default="")
    parser.add_argument("--api-key-env", default="TTS_API_KEY")
    parser.add_argument("--reuse-tts-dir", default="")
    parser.add_argument("--lag", type=float, default=0.8)
    parser.add_argument("--subtitle-alpha", default="99")
    parser.add_argument("--original-volume", type=float, default=0.16)
    parser.add_argument("--preview-duration", type=float, default=0.0)
    parser.add_argument("--glossary", default="")
    parser.add_argument("--no-stt-audit", action="store_true")
    parser.add_argument("--stt-threshold", type=float, default=0.72)
    parser.add_argument("--self-check", action="store_true")
    args = parser.parse_args()
    if args.self_check:
        self_check()
        return
    if not args.task or not args.segments:
        parser.error("--task and --segments are required unless --self-check is used")
    build(args)


if __name__ == "__main__":
    main()
