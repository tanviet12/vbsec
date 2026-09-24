#!/usr/bin/env python3
"""Kiểm tra JSON summary cuối report vbsec có đúng schema không.

Usage: python3 <skill-dir>/references/validate-report.py vbsec-reports/scan-<timestamp>.md

Exit 0 = hợp lệ. Exit 1 = in danh sách lỗi, agent phải sửa report rồi chạy lại.
Chỉ dùng thư viện chuẩn Python 3.
"""
import glob
import json
import os
import re
import sys

SKILL_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SEVERITIES = {"CRITICAL", "HIGH", "MEDIUM", "LOW"}
VERDICTS = {"PASS", "WARN", "FAIL"}
REQUIRED_TOP = ["verdict", "summary", "scope", "files_reviewed", "primary_language",
                "specialized_rules_used", "mode", "date", "findings"]
REQUIRED_FINDING = ["file", "line", "rule_id", "severity", "issue_summary", "fix_summary"]
ALIASES = {"id": "rule_id", "rule": "rule_id", "note": "issue_summary", "summary": "issue_summary",
           "issue": "issue_summary", "fix": "fix_summary", "path": "file"}


def canonical_rule_ids():
    ids = set()
    for path in glob.glob(os.path.join(SKILL_DIR, "rules", "generic", "*.md")):
        m = re.search(r"^id:\s*(\S+)", open(path, encoding="utf-8").read(), re.M)
        if m:
            ids.add(m.group(1))
    return ids


def validate(report_path):
    text = open(report_path, encoding="utf-8").read()
    blocks = re.findall(r"```json\s*\n(.*?)\n```", text, re.S)
    if not blocks:
        return ["không có block ```json ở cuối report"]
    try:
        data = json.loads(blocks[-1])
    except json.JSONDecodeError as e:
        return [f"JSON không parse được: {e}"]

    errors = []
    rule_ids = canonical_rule_ids()
    for key in REQUIRED_TOP:
        if key not in data:
            errors.append(f"thiếu field top-level `{key}`")
    if data.get("verdict") not in VERDICTS:
        errors.append(f"`verdict` phải là PASS/WARN/FAIL, đang là {data.get('verdict')!r}")

    findings = data.get("findings", [])
    if not isinstance(findings, list):
        return errors + ["`findings` phải là array"]

    for i, f in enumerate(findings):
        where = f"findings[{i}] ({f.get('file', '?')}:{f.get('line', '?')})"
        for alias, proper in ALIASES.items():
            if alias in f and proper not in f:
                errors.append(f"{where}: dùng key `{alias}`, phải đổi thành `{proper}`")
        for key in REQUIRED_FINDING:
            if key not in f and not any(a in f and p == key for a, p in ALIASES.items()):
                errors.append(f"{where}: thiếu `{key}`")
        rid = f.get("rule_id")
        if rid is not None and rid not in rule_ids:
            errors.append(f"{where}: rule_id `{rid}` không thuộc danh sách rule canonical "
                          f"— map sang rule gần nhất hoặc chuyển sang `hardening_notes`")
        sev = f.get("severity")
        if sev is not None and sev not in SEVERITIES:
            errors.append(f"{where}: severity phải viết hoa CRITICAL/HIGH/MEDIUM/LOW, đang là {sev!r}")
        line = f.get("line")
        if line is not None and not (isinstance(line, int) and line >= 1):
            errors.append(f"{where}: `line` phải là số nguyên >= 1 (dòng bắt đầu), đang là {line!r}")

    summary = data.get("summary", {})
    for sev in SEVERITIES:
        want = sum(1 for f in findings if f.get("severity") == sev)
        got = summary.get(sev.lower())
        if got != want:
            errors.append(f"summary.{sev.lower()} = {got!r} nhưng findings có {want} mục {sev}")

    notes = data.get("hardening_notes", [])
    if not isinstance(notes, list) or any(not isinstance(n, dict) or "note" not in n for n in notes):
        errors.append("`hardening_notes` (nếu có) phải là array các object có field `note`")
    return errors


def main():
    if len(sys.argv) != 2:
        print(__doc__)
        sys.exit(2)
    errors = validate(sys.argv[1])
    if errors:
        print(f"JSON summary KHÔNG hợp lệ ({len(errors)} lỗi):")
        for e in errors:
            print(f"  - {e}")
        sys.exit(1)
    print("JSON summary hợp lệ")


if __name__ == "__main__":
    main()
