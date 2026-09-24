#!/usr/bin/env bash
# run-fixtures.sh — chạy /vbs-scan-security trên từng bộ fixture bằng Claude Code headless,
# rồi chấm điểm bằng check-fixtures.py.
#
# Skill được lấy từ ~/.claude/skills/vbs-scan-security (install.sh symlink về repo này),
# nên đang test đúng bản trong working copy.
#
# Fixture phải được git track (scope `all` dùng `git ls-files`) — commit trước khi chạy.
# Mỗi ngôn ngữ tốn 1 lần scan thật (token), chạy riêng lẻ khi chỉ sửa 1 ngôn ngữ.
#
# Usage:
#   ./scripts/run-fixtures.sh                # mọi ngôn ngữ
#   ./scripts/run-fixtures.sh python php     # chỉ vài ngôn ngữ
#   ./scripts/run-fixtures.sh --check-only   # không scan, chỉ chấm report mới nhất
#
# Token mỗi lần scan được ghi vào tests/usage.log để so trước/sau khi đổi skill.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHECK_ONLY=0
LANGS=()
for arg in "$@"; do
  case "$arg" in
    --check-only) CHECK_ONLY=1 ;;
    -h|--help) grep -E '^# ' "$0" | sed 's/^# //'; exit 0 ;;
    *) LANGS+=("$arg") ;;
  esac
done
if [ ${#LANGS[@]} -eq 0 ]; then
  for f in "$ROOT"/tests/expected/*.json; do LANGS+=("$(basename "$f" .json)"); done
fi

USAGE_LOG="$ROOT/tests/usage.log"   # token mỗi lần scan (gitignored), dùng để so trước/sau
USAGE_TMP="$(mktemp)"

if [ "$CHECK_ONLY" -eq 0 ]; then
  command -v claude >/dev/null || { echo "Cần Claude Code CLI (claude) để chạy scan"; exit 1; }
  for lang in "${LANGS[@]}"; do
    dir="$ROOT/tests/fixtures/$lang"
    [ -d "$dir" ] || { echo "Không có fixture: $lang"; exit 1; }
    if [ -z "$(git -C "$dir" ls-files .)" ]; then
      echo "Fixture $lang chưa được git track — commit trước khi chạy"; exit 1
    fi
    echo "== scan $lang"
    (cd "$dir" && claude -p "/vbs-scan-security all lang=en" \
      --allowedTools "Bash Read Grep Glob Write" --output-format json > "$USAGE_TMP") || true
    python3 - "$USAGE_TMP" "$lang" "$USAGE_LOG" <<'PY'
import json, sys, datetime
path, lang, log = sys.argv[1:4]
try:
    d = json.load(open(path))
except Exception:
    print(f"   (không đọc được usage của {lang})"); sys.exit(0)
u = d.get("usage", {})
total = sum(u.get(k, 0) for k in ("input_tokens", "cache_creation_input_tokens", "cache_read_input_tokens", "output_tokens"))
line = (f"{datetime.datetime.now():%Y-%m-%d %H:%M} {lang} turns={d.get('num_turns')} "
        f"total={total} input={u.get('input_tokens', 0)} cache_write={u.get('cache_creation_input_tokens', 0)} "
        f"cache_read={u.get('cache_read_input_tokens', 0)} output={u.get('output_tokens', 0)} "
        f"cost_usd={d.get('total_cost_usd', 0):.2f}")
print("   " + line)
open(log, "a").write(line + "\n")
PY
  done
  rm -f "$USAGE_TMP"
fi

python3 "$ROOT/scripts/check-fixtures.py" "${LANGS[@]}"
