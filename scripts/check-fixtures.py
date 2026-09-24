#!/usr/bin/env python3
"""check-fixtures.py — chấm điểm báo cáo vbsec so với đáp án trong tests/expected/.

Đọc JSON summary (block ```json cuối cùng) trong report .md, so với
tests/expected/<lang>.json:
  - must_find      : finding bắt buộc phải có (file + rule_id, severity >= min_severity,
                     tuỳ chọn lines = [start, end] khi 1 file có cả đoạn lỗi lẫn đoạn an toàn)
  - must_not_find  : finding KHÔNG được có (bẫy false positive)
  - clean_files    : file không được có finding nào

Usage:
  python3 scripts/check-fixtures.py                 # chấm mọi ngôn ngữ, lấy report mới nhất
  python3 scripts/check-fixtures.py python go       # chỉ chấm 1 số ngôn ngữ
  python3 scripts/check-fixtures.py python --report path/to/scan.md

Exit code 1 nếu có must_find bị sót hoặc dính must_not_find / clean_files.
"""
import argparse
import glob
import json
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
EXPECTED_DIR = os.path.join(ROOT, "tests", "expected")
SEVERITY = {"LOW": 1, "MEDIUM": 2, "HIGH": 3, "CRITICAL": 4}


def latest_report(fixture_dir):
    reports = sorted(glob.glob(os.path.join(fixture_dir, "vbsec-reports", "scan-*.md")))
    return reports[-1] if reports else None


def load_findings(report_path):
    text = open(report_path, encoding="utf-8").read()
    blocks = re.findall(r"```json\s*\n(.*?)\n```", text, re.S)
    if not blocks:
        raise ValueError(f"không tìm thấy block ```json trong {report_path}")
    data = json.loads(blocks[-1])
    return data.get("findings", [])


def norm_path(path, fixture):
    path = path.strip()
    if path.startswith("./"):
        path = path[2:]
    prefix = fixture.rstrip("/") + "/"
    return path[len(prefix):] if path.startswith(prefix) else path


def in_lines(entry, finding):
    """entry["lines"] = [start, end] (tuỳ chọn): finding phải nằm trong khoảng dòng này."""
    if "lines" not in entry:
        return True
    try:
        line = int(finding.get("line"))
    except (TypeError, ValueError):
        return False
    return entry["lines"][0] <= line <= entry["lines"][1]


def rule_ids(entry):
    rid = entry["rule_id"]
    return rid if isinstance(rid, list) else [rid]


def check(lang, report_path):
    expected = json.load(open(os.path.join(EXPECTED_DIR, f"{lang}.json"), encoding="utf-8"))
    fixture = expected["fixture"]
    fixture_dir = os.path.join(ROOT, fixture)
    report_path = report_path or latest_report(fixture_dir)
    print(f"\n== {lang}")
    if not report_path:
        print(f"   chưa có report trong {fixture}/vbsec-reports/ — chạy scan trước")
        return None

    findings = [
        {**f, "file": norm_path(f.get("file", ""), fixture), "severity": str(f.get("severity", "")).upper()}
        for f in load_findings(report_path)
    ]
    print(f"   report: {os.path.relpath(report_path, ROOT)} ({len(findings)} findings)")

    hit, missed, weak = 0, [], []
    matched = set()
    for exp in expected["must_find"]:
        ids = rule_ids(exp)
        cands = [i for i, f in enumerate(findings) if f["file"] == exp["file"] and f["rule_id"] in ids and in_lines(exp, f)]
        if not cands:
            missed.append(exp)
            continue
        hit += 1
        matched.update(cands)
        best = max(SEVERITY.get(findings[i]["severity"], 0) for i in cands)
        if best < SEVERITY[exp.get("min_severity", "LOW")]:
            weak.append((exp, best))

    false_pos = [
        (bad, f) for bad in expected["must_not_find"] for f in findings
        if f["file"] == bad["file"] and f["rule_id"] in rule_ids(bad) and in_lines(bad, f)
    ]
    dirty = [f for f in findings if f["file"] in expected["clean_files"]]
    extra = [
        f for i, f in enumerate(findings)
        if i not in matched and f["file"] not in expected["clean_files"] and not any(f is fp for _, fp in false_pos)
    ]

    total = len(expected["must_find"])
    print(f"   recall: {hit}/{total} ({hit * 100 // total}%)")
    for exp in missed:
        print(f"   SÓT      {exp['file']}  {' | '.join(rule_ids(exp))}")
    for exp, best in weak:
        got = next((k for k, v in SEVERITY.items() if v == best), "?")
        print(f"   YẾU      {exp['file']}  {' | '.join(rule_ids(exp))}: {got} < {exp['min_severity']}")
    for bad, f in false_pos:
        print(f"   BÁO NHẦM {f['file']}:{f.get('line', '?')}  {f['rule_id']}  ({bad['reason']})")
    for f in dirty:
        if not any(f is fp for _, fp in false_pos):
            print(f"   BÁO NHẦM {f['file']}:{f.get('line', '?')}  {f['rule_id']}  (file sạch)")
    if extra:
        print(f"   khác: {len(extra)} finding ngoài đáp án (không tính điểm, nên xem tay):")
        for f in extra:
            print(f"            {f['file']}:{f.get('line', '?')}  {f['rule_id']}  {f['severity']}")

    return not missed and not false_pos and not dirty


def main():
    langs_available = sorted(os.path.splitext(os.path.basename(p))[0] for p in glob.glob(os.path.join(EXPECTED_DIR, "*.json")))
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("langs", nargs="*", metavar="lang", help=f"một trong: {', '.join(langs_available)}")
    ap.add_argument("--report", help="đường dẫn report cụ thể (chỉ dùng khi chấm 1 ngôn ngữ)")
    args = ap.parse_args()
    langs = args.langs or langs_available
    unknown = [l for l in langs if l not in langs_available]
    if unknown:
        ap.error(f"không có đáp án cho: {', '.join(unknown)}")
    if args.report and len(langs) != 1:
        ap.error("--report chỉ dùng với đúng 1 ngôn ngữ")

    results = {lang: check(lang, args.report) for lang in langs}
    print()
    ok = all(r is True for r in results.values())
    print("PASS" if ok else "FAIL", "—", ", ".join(f"{k}: {'ok' if v else ('chưa chạy' if v is None else 'fail')}" for k, v in results.items()))
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
