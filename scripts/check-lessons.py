#!/usr/bin/env python3
"""check-lessons.py — mỗi bài học bảo mật phải có ca kiểm trong bộ test vbsec.

Kho bài học là file markdown riêng của người dùng (mặc định ~/.claude/rules/bai-hoc-bao-mat.md,
đổi bằng --lessons hoặc biến VBSEC_LESSONS), KHÔNG nằm trong repo. Mỗi bài có dạng
`<số>. **<tiêu đề>.** ...`. Repo chỉ giữ bảng nối `tests/lessons-map.json`:

  {"lessons": {
     "34": {"title": "...", "status": "covered", "rule": "TENANT-ISOLATION",
            "cases": [{"set": "saas-hard", "file": "internal/repo/report.go", "rule_id": "TENANT-ISOLATION"}]},
     "4":  {"title": "...", "status": "out_of_scope", "reason": "cấu hình vỏ desktop, không thấy trong mã quét"}
  }}

  status  covered       → mọi ca phải tồn tại trong tests/expected/<set>.json (must_find hoặc must_not_find,
                          khớp file và rule_id)
          out_of_scope  → bắt buộc có `reason` (vd quy trình, hạ tầng, không phát hiện được bằng đọc mã)
          todo          → đã ghi nhận, chưa có ca (tính là thiếu)

Đỏ (exit 1) khi: bài có trong kho mà chưa có trong bảng nối, bài `todo`, ca trỏ tới đáp án không tồn tại,
hoặc tiêu đề bài đã đổi so với bảng nối (soát lại ca rồi cập nhật `title`).
Không tìm thấy kho bài học → in cảnh báo, exit 0 (người dùng repo public không có kho này).

Usage:
  python3 scripts/check-lessons.py
  python3 scripts/check-lessons.py --lessons path/to/bai-hoc.md
  python3 scripts/check-lessons.py --init      # in khung JSON cho bài chưa có trong bảng nối (status todo)
"""
import argparse
import json
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MAP_PATH = os.path.join(ROOT, "tests", "lessons-map.json")
EXPECTED_DIR = os.path.join(ROOT, "tests", "expected")
DEFAULT_LESSONS = os.environ.get("VBSEC_LESSONS", os.path.expanduser("~/.claude/rules/bai-hoc-bao-mat.md"))
LESSON_RE = re.compile(r"^(\d+)\.\s+\*\*(.+?)\*\*", re.M)


def norm_title(t):
    return re.sub(r"\s+", " ", t).strip().rstrip(".").strip()


def parse_lessons(path):
    text = open(path, encoding="utf-8").read()
    lessons = {}
    for m in LESSON_RE.finditer(text):
        lessons.setdefault(m.group(1), norm_title(m.group(2)))
    return lessons


def rule_ids(entry):
    rid = entry.get("rule_id")
    return set(rid if isinstance(rid, list) else [rid])


def case_exists(case, cache):
    s = case.get("set")
    if s not in cache:
        p = os.path.join(EXPECTED_DIR, f"{s}.json")
        cache[s] = json.load(open(p, encoding="utf-8")) if os.path.exists(p) else None
    exp = cache[s]
    if exp is None:
        return f"không có tests/expected/{s}.json"
    want = case.get("rule_id")
    wants = set(want if isinstance(want, list) else [want])
    for entry in exp.get("must_find", []) + exp.get("must_not_find", []):
        if entry.get("file") == case.get("file") and rule_ids(entry) & wants:
            return None
    return f"{s}: không có ca {case.get('file')} / {want}"


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--lessons", default=DEFAULT_LESSONS)
    ap.add_argument("--init", action="store_true", help="in khung JSON cho bài chưa có trong bảng nối")
    args = ap.parse_args()

    if not os.path.exists(args.lessons):
        print(f"Không thấy kho bài học {args.lessons} — bỏ qua (dùng --lessons hoặc VBSEC_LESSONS).")
        sys.exit(0)
    lessons = parse_lessons(args.lessons)
    mapping = json.load(open(MAP_PATH, encoding="utf-8")).get("lessons", {}) if os.path.exists(MAP_PATH) else {}

    if args.init:
        skel = {n: {"title": t, "status": "todo"} for n, t in sorted(lessons.items(), key=lambda x: int(x[0])) if n not in mapping}
        print(json.dumps(skel, ensure_ascii=False, indent=2))
        return

    errors, cache = [], {}
    counts = {"covered": 0, "out_of_scope": 0, "todo": 0, "missing": 0}
    for n, title in sorted(lessons.items(), key=lambda x: int(x[0])):
        m = mapping.get(n)
        if m is None:
            counts["missing"] += 1
            errors.append(f"bài {n} chưa có trong bảng nối: {title[:70]}")
            continue
        if norm_title(m.get("title", "")) != title:
            errors.append(f"bài {n} đã đổi tiêu đề — soát lại ca rồi cập nhật `title`: {title[:70]}")
        st = m.get("status")
        if st == "covered":
            cases = m.get("cases") or []
            if not cases:
                errors.append(f"bài {n} covered nhưng không có `cases`")
            for c in cases:
                e = case_exists(c, cache)
                if e:
                    errors.append(f"bài {n}: {e}")
            counts["covered"] += 1
        elif st == "out_of_scope":
            if not m.get("reason"):
                errors.append(f"bài {n} out_of_scope thiếu `reason`")
            counts["out_of_scope"] += 1
        else:
            counts["todo"] += 1
            errors.append(f"bài {n} chưa có ca kiểm (status {st!r}): {title[:70]}")

    stale = sorted(set(mapping) - set(lessons), key=int)
    total = len(lessons)
    in_scope = total - counts["out_of_scope"]
    pct = counts["covered"] * 100 // in_scope if in_scope else 100
    print(f"Kho: {args.lessons} — {total} bài")
    print(f"  có ca kiểm {counts['covered']}/{in_scope} bài trong phạm vi ({pct}%), ngoài phạm vi {counts['out_of_scope']}, "
          f"todo {counts['todo']}, chưa nối {counts['missing']}")
    if stale:
        print(f"  bảng nối có bài không còn trong kho: {', '.join(stale)} (đã chuyển thành quy tắc cứng? xoá khỏi bảng nối)")
    for e in errors:
        print(f"  ĐỎ  {e}")
    sys.exit(1 if errors else 0)


if __name__ == "__main__":
    main()
