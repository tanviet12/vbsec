#!/usr/bin/env bash
# load-rules.sh — in phần PHÁT HIỆN của bộ rule hiệu lực cho các ngôn ngữ đã detect.
#
# Usage: bash <skill-dir>/references/load-rules.sh --part N [lang ...]
#   vd:  bash references/load-rules.sh --part 1 typescript
#        bash references/load-rules.sh --part 2 go typescript   # repo đa ngôn ngữ
#        bash references/load-rules.sh --part 1                 # không có overlay → chỉ generic
#
# Output được chia thành nhiều phần, mỗi phần < ~20.000 ký tự, vì Bash tool của agent
# cắt output dài (~30.000 ký tự). Dòng cuối mỗi phần cho biết tổng số phần; chạy đủ
# mọi phần (có thể chạy song song). Không truyền --part = in phần 1.
#
# Với mỗi rule generic (01-21):
#   - Mọi lang đã detect đều có overlay cùng `id` → chỉ in overlay (generic bị thay thế hoàn toàn).
#   - Ngược lại → in generic, kèm overlay của lang nào có.
# Mỗi file chỉ in tới trước heading chi tiết đầu tiên (## Examples / ## Fix recommendation /
# ## Cross-references). Phần chi tiết đọc sau, chỉ cho rule có finding CRITICAL/HIGH,
# bằng Read tool theo đường dẫn in ở header `=== RULE ... ===`.
# File không có heading chi tiết → in toàn bộ (fallback an toàn).

set -euo pipefail

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PART_BUDGET=20000   # ký tự mỗi phần
PART=1
LANGS=()
while [ $# -gt 0 ]; do
  case "$1" in
    --part) PART="$2"; shift 2 ;;
    --part=*) PART="${1#--part=}"; shift ;;
    *) LANGS+=("$1"); shift ;;
  esac
done

rule_id() { sed -n 's/^id:[[:space:]]*//p' "$1" | head -1; }

overlay_for() {  # overlay_for <lang> <id> → path hoặc rỗng
  local dir="$SKILL_DIR/rules/languages/$1" f
  [ -d "$dir" ] || return 0
  for f in "$dir"/[0-9]*.md; do
    [ -f "$f" ] && [ "$(rule_id "$f")" = "$2" ] && { echo "$f"; return 0; }
  done
  return 0
}

detection() {  # detection <path> → phần phát hiện của 1 rule, kèm header
  local rel="${1#"$SKILL_DIR"/}"
  echo "=== RULE $(rule_id "$1") (source: $rel) ==="
  awk '/^## (Examples|Fix recommendation|Cross-references)/ { exit } { print }' "$1"
  echo
}

FILES_SELECTED=()

for generic in "$SKILL_DIR"/rules/generic/[0-9]*.md; do
  id="$(rule_id "$generic")"
  overlays=()
  all_covered=1
  for lang in ${LANGS[@]+"${LANGS[@]}"}; do
    o="$(overlay_for "$lang" "$id")"
    if [ -n "$o" ]; then overlays+=("$o"); else all_covered=0; fi
  done
  [ ${#LANGS[@]} -eq 0 ] && all_covered=0
  [ "$all_covered" -eq 1 ] || FILES_SELECTED+=("$generic")
  for o in ${overlays[@]+"${overlays[@]}"}; do FILES_SELECTED+=("$o"); done
done

# Gom rule vào từng phần theo ngân sách ký tự (không cắt ngang 1 rule).
part_of=()
current=1
used=0
for f in "${FILES_SELECTED[@]}"; do
  size=$(detection "$f" | wc -m | tr -d ' ')
  if [ "$used" -gt 0 ] && [ $((used + size)) -gt "$PART_BUDGET" ]; then
    current=$((current + 1))
    used=0
  fi
  part_of+=("$current")
  used=$((used + size))
done
total_parts=$current

if ! [ "$PART" -ge 1 ] 2>/dev/null || [ "$PART" -gt "$total_parts" ]; then
  echo "--part phải từ 1 đến $total_parts" >&2
  exit 1
fi

for i in "${!FILES_SELECTED[@]}"; do
  [ "${part_of[$i]}" -eq "$PART" ] && detection "${FILES_SELECTED[$i]}"
done
if [ "$PART" -lt "$total_parts" ]; then
  echo "=== PART $PART/$total_parts — CHƯA ĐỦ: chạy tiếp --part $((PART + 1)) đến --part $total_parts ==="
else
  echo "=== PART $PART/$total_parts — đã in đủ ${#FILES_SELECTED[@]} rule ==="
fi
