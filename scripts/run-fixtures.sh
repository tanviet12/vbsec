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
      --allowedTools "Bash Read Grep Glob Write" >/dev/null)
  done
fi

python3 "$ROOT/scripts/check-fixtures.py" "${LANGS[@]}"
