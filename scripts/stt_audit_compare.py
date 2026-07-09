#!/usr/bin/env python3
import argparse
import difflib
import json
import string
import sys
import unicodedata
from pathlib import Path


COMPARE_PUNCT = "，。？！、；：·・～…—,.?!;:'\"()（）[]「」『』- "
TERM_PUNCT = "".join(ch for ch in string.punctuation if ch not in "_")


def normalize_compare(value):
    text = unicodedata.normalize("NFKC", value or "").lower()
    return "".join(ch for ch in text if not ch.isspace() and ch not in COMPARE_PUNCT)


def normalize_term(value):
    text = unicodedata.normalize("NFKC", value or "").lower()
    drop = set(TERM_PUNCT)
    return "".join(ch for ch in text if not ch.isspace() and ch not in drop)


def similarity(expected, transcript):
    return difflib.SequenceMatcher(None, normalize_compare(expected), normalize_compare(transcript)).ratio()


def load_pairs(args):
    if args.pairs:
        if args.expected or args.transcripts:
            raise SystemExit("--pairs cannot be used with --expected or --transcripts")
        data = json.loads(Path(args.pairs).read_text(encoding="utf-8"))
        return [{
            "id": int(row["id"]),
            "expected": row.get("expected", ""),
            "transcript": row.get("transcript", ""),
            "has_transcript": "transcript" in row,
        } for row in data]
    if not args.expected or not args.transcripts:
        raise SystemExit("use either --pairs, or both --expected and --transcripts")

    package = json.loads(Path(args.expected).read_text(encoding="utf-8"))
    transcripts = json.loads(Path(args.transcripts).read_text(encoding="utf-8"))
    by_id = {int(key): value for key, value in transcripts.items()}
    return [{
        "id": int(cue["id"]),
        "expected": cue.get("text", ""),
        "transcript": by_id.get(int(cue["id"]), ""),
        "has_transcript": int(cue["id"]) in by_id,
    } for cue in package.get("cues", [])]


def glossary_terms(path):
    if not path:
        return []
    data = json.loads(Path(path).read_text(encoding="utf-8"))
    terms = data.get("terms", [])
    if isinstance(terms, dict):
        rows = list(terms.keys()) + list(terms.values())
    else:
        rows = terms
    rows += data.get("protected_names", [])
    return [str(row) for row in rows if str(row).strip()]


def missing_terms(expected, transcript, terms):
    expected_norm = normalize_term(expected)
    transcript_norm = normalize_term(transcript)
    missing = []
    for term in terms:
        term_norm = normalize_term(term)
        if term_norm and term_norm in expected_norm and term_norm not in transcript_norm:
            missing.append(term)
    return list(dict.fromkeys(missing))


def failure_reason(score, threshold, missing, has_transcript):
    if not has_transcript:
        return "missing_transcript"
    reasons = []
    if score < threshold:
        reasons.append("low_similarity")
    if missing:
        reasons.append("missing_terms")
    return "|".join(reasons)


def build_report(pairs, threshold, terms):
    failed = []
    total = 0
    for pair in pairs:
        expected = (pair.get("expected") or "").strip()
        if not expected:
            continue
        total += 1
        transcript = pair.get("transcript") or ""
        score = similarity(expected, transcript) if pair.get("has_transcript") else 0.0
        missing = missing_terms(expected, transcript, terms) if pair.get("has_transcript") else []
        reason = failure_reason(score, threshold, missing, pair.get("has_transcript"))
        if reason:
            failed.append({
                "id": pair["id"],
                "similarity": round(score, 4),
                "missing_terms": missing,
                "reason": reason,
                "expected": expected,
                "transcript": transcript,
            })
    return {
        "threshold": threshold,
        "total": total,
        "passed": total - len(failed),
        "failed": failed,
        "failed_ids": [row["id"] for row in failed],
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--expected")
    parser.add_argument("--transcripts")
    parser.add_argument("--pairs")
    parser.add_argument("--glossary")
    parser.add_argument("--threshold", type=float, default=0.85)
    parser.add_argument("-o", "--output")
    args = parser.parse_args()

    pairs = load_pairs(args)
    report = build_report(pairs, args.threshold, glossary_terms(args.glossary))
    text = json.dumps(report, ensure_ascii=False, indent=2) + "\n"
    if args.output:
        Path(args.output).write_text(text, encoding="utf-8")
    else:
        sys.stdout.write(text)
    raise SystemExit(1 if report["failed"] else 0)


if __name__ == "__main__":
    main()
