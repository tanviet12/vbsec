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
REQUIRED_TOP = ["vbsec_version", "verdict", "summary", "scope", "files_reviewed", "primary_language",
                "specialized_rules_used", "mode", "date", "findings"]
# Lỗi có đường khai thác mà không rule nào khớp → OTHER-CWE-<số>, kèm field `cwe`, tính vào summary/verdict.
OTHER_CWE = re.compile(r"^OTHER-CWE-(\d+)$")
OTHER_SEVERITY_MAX = "HIGH"
RANK = {"LOW": 1, "MEDIUM": 2, "HIGH": 3, "CRITICAL": 4}
REQUIRED_FINDING = ["file", "line", "rule_id", "severity", "issue_summary", "fix_summary"]
ALIASES = {"id": "rule_id", "rule": "rule_id", "note": "issue_summary", "summary": "issue_summary",
           "issue": "issue_summary", "fix": "fix_summary", "path": "file"}
# Finding mà chính issue_summary thừa nhận là không khai thác được, hoặc thú nhận đã gán rule gần nhất
# → không phải finding (chuyển sang hardening_notes) hoặc phải dùng OTHER-CWE-<n>. Ngoại lệ: rule
# dependency (20, 22) — "CVE không reachable với code hiện tại" ở đó là lý do hạ severity, vẫn là finding.
DEPENDENCY_RULES = {"OUTDATED-DEPENDENCY", "VULNERABLE-DEPENDENCY"}
SELF_NEGATING = re.compile(
    r"\b(not|isn't|is not|aren't|are not|cannot be|can't be|no longer|never)\s+"
    r"(currently\s+|directly\s+|actually\s+)?(reachable|exploitable|exploited|triggerable|affected|vulnerable|attacker[- ]controlled)\b"
    r"|\b(un)?reachable by current code\b|\bnone (of (them|these) )?(is|are)? ?reachable\b"
    r"|\bno (concrete |direct |known )?(exploit|attack) (path|vector)\b"
    r"|\b(is|are|remains?|looks?|seems?|appears?) (already |currently )?safe\b"
    r"|\bsafe (as[- ]is|by default|in practice)\b"
    r"|\b(defense[- ]in[- ]depth|hardening|best[- ]practice) (only|suggestion|note)\b"
    r"|\bmapped to (the )?(closest|nearest)\b|\b(closest|nearest) (canonical )?rule\b"
    r"|\bno security impact\b|\btheoretical only\b"
    r"|không khai thác được|không tới được|không có đường khai thác|chỉ là (gợi ý|phòng thủ|hardening)"
    r"|(?<!không )(?<!chưa )\ban toàn( với| trong| ở)? (hiện tại|thực tế|mặc định)\b|\bđã an toàn\b"
    r"|gán (vào |cho )?rule gần nhất|rule gần nhất",
    re.I)


def canonical_rules():
    """{rule_id: severity_max} đọc từ frontmatter rules/generic/*.md."""
    rules = {}
    for path in glob.glob(os.path.join(SKILL_DIR, "rules", "generic", "*.md")):
        text = open(path, encoding="utf-8").read()
        m = re.search(r"^id:\s*(\S+)", text, re.M)
        if m:
            sm = re.search(r"^severity_max:\s*(\S+)", text, re.M)
            rules[m.group(1)] = sm.group(1).upper() if sm else "CRITICAL"
    return rules


def expected_verdict(findings):
    sevs = {f.get("severity") for f in findings}
    return "FAIL" if "CRITICAL" in sevs else "WARN" if "HIGH" in sevs else "PASS"


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
    rules = canonical_rules()
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
        sev = f.get("severity")
        cap = None
        m_other = OTHER_CWE.match(rid) if isinstance(rid, str) else None
        if m_other:
            cap = OTHER_SEVERITY_MAX
            cwe = str(f.get("cwe", ""))
            if cwe.upper().replace("CWE-", "") != m_other.group(1):
                errors.append(f"{where}: rule_id `{rid}` cần field `cwe` = \"CWE-{m_other.group(1)}\"")
        elif rid is not None and rid not in rules:
            errors.append(f"{where}: rule_id `{rid}` không thuộc danh sách rule. KHÔNG gán sang rule gần nhất: "
                          f"nếu có đường khai thác cụ thể mà không rule nào khớp → dùng `OTHER-CWE-<số>` kèm `cwe`, "
                          f"còn không có đường khai thác thì chuyển sang `hardening_notes`")
        else:
            cap = rules.get(rid)
        if cap and sev in RANK and RANK[sev] > RANK[cap]:
            errors.append(f"{where}: severity {sev} vượt trần {cap} của `{rid}`")
        issue_text = " ".join(str(f.get(k, "")) for k in ("issue_summary", "issue", "note", "summary"))
        m = SELF_NEGATING.search(issue_text)
        if m and rid not in DEPENDENCY_RULES:
            errors.append(f"{where}: issue_summary tự phủ định (\"{m.group(0)}\"). Chọn 1 trong 2: "
                          f"(a) lỗi nằm trong code (check/sanitizer viết sai, input tới được sink) → GIỮ finding, hạ severity nếu cần, "
                          f"viết lại issue_summary nêu attacker làm gì và điều kiện bypass, bỏ cụm phủ định; "
                          f"(b) input attacker không tới được sink, chỉ là lớp phòng thủ thêm → xoá khỏi `findings[]`, "
                          f"ghi vào `hardening_notes[]` (không rule_id), cập nhật `summary` và verdict")
        if sev is not None and sev not in SEVERITIES:
            errors.append(f"{where}: severity phải viết hoa CRITICAL/HIGH/MEDIUM/LOW, đang là {sev!r}")
        line = f.get("line")
        if line is not None and not (isinstance(line, int) and line >= 1):
            errors.append(f"{where}: `line` phải là số nguyên >= 1 (dòng bắt đầu), đang là {line!r}")

    seen = {}
    for i, f in enumerate(findings):
        key = (str(f.get("file", "")).lstrip("./"), f.get("line"), f.get("rule_id"))
        if key in seen:
            errors.append(f"findings[{i}] trùng findings[{seen[key]}] (cùng file:line:rule_id) — gộp lại, giữ severity cao nhất")
        else:
            seen[key] = i

    summary = data.get("summary", {})
    for sev in SEVERITIES:
        want = sum(1 for f in findings if f.get("severity") == sev)
        got = summary.get(sev.lower())
        if got != want:
            errors.append(f"summary.{sev.lower()} = {got!r} nhưng findings có {want} mục {sev}")

    if data.get("verdict") in VERDICTS and isinstance(findings, list):
        want_v = expected_verdict(findings)
        if data["verdict"] != want_v and not data.get("verdict_downgraded_reason"):
            errors.append(f"verdict = {data['verdict']} nhưng findings cho ra {want_v} (CRITICAL → FAIL, HIGH → WARN). "
                          f"Finding OTHER-CWE-* cũng tính. Chỉ được nặng hơn khi có `verdict_downgraded_reason` (vd chunk lỗi)")

    cov = data.get("coverage")
    if cov is not None and not isinstance(cov, dict):
        errors.append("`coverage` (nếu có) phải là object: {files_read, files_grep_only, files_skipped, rules_skipped, context_file}")

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
