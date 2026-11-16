#!/usr/bin/env bash
# load-rules.sh — in phần PHÁT HIỆN của bộ rule hiệu lực cho các ngôn ngữ đã detect.
#
# Usage: bash <skill-dir>/references/load-rules.sh [--root DIR] --part N [lang ...]
#   vd:  bash references/load-rules.sh --root "$SCAN_ROOT" --part 1 typescript
#        bash references/load-rules.sh --part 2 go typescript   # repo đa ngôn ngữ
#        bash references/load-rules.sh --part 1                 # không có overlay → chỉ generic
#
# Output được chia thành nhiều phần, mỗi phần < ~20.000 ký tự, vì Bash tool của agent
# cắt output dài (~30.000 ký tự). Dòng cuối mỗi phần cho biết tổng số phần; chạy đủ
# mọi phần (có thể chạy song song). Không truyền --part = in phần 1.
#
# Rule generic có `opt_in:` trong frontmatter (vd rule 22, chỉ chạy khi có --sca) KHÔNG được nạp ở
# pha quét — workflow tương ứng tự Read rule đó khi user bật cờ.
#
# Rule có `load_when: '<regex>'` trong frontmatter (vd rule AI/MCP, tenant) chỉ được nạp khi truyền
# --root và trong DIR có ít nhất 1 file khớp regex (grep -E -i, bỏ node_modules/vendor/.git...). Không
# truyền --root → nạp mọi rule (an toàn: thà tốn token còn hơn bỏ sót). Rule bị bỏ được liệt kê ở
# dòng cuối để báo cáo ghi vào bảng độ phủ.
#
# Với mỗi rule generic còn lại:
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
ROOT=""
LANGS=()
while [ $# -gt 0 ]; do
  case "$1" in
    --part) PART="$2"; shift 2 ;;
    --part=*) PART="${1#--part=}"; shift ;;
    --root) ROOT="$2"; shift 2 ;;
    --root=*) ROOT="${1#--root=}"; shift ;;
    *) LANGS+=("$1"); shift ;;
  esac
done

rule_id() { sed -n 's/^id:[[:space:]]*//p' "$1" | head -1; }
is_opt_in() { sed -n '/^---$/,/^---$/p' "$1" | grep -q '^opt_in:'; }
load_when() { sed -n '/^---$/,/^---$/p' "$1" | sed -n "s/^load_when:[[:space:]]*//p" | head -1 | sed -e "s/^'//" -e "s/'\$//" -e 's/^"//' -e 's/"$//'; }

triggered() {  # triggered <regex> → 0 nếu ROOT có file khớp
  [ -z "$ROOT" ] && return 0
  [ -d "$ROOT" ] || return 0
  grep -rIqiE --exclude-dir=node_modules --exclude-dir=vendor --exclude-dir=.git \
    --exclude-dir=dist --exclude-dir=build --exclude-dir=vbsec-reports --exclude-dir=.vbsec-tmp \
    -e "$1" "$ROOT" 2>/dev/null
}

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
SKIPPED=()

for generic in "$SKILL_DIR"/rules/generic/[0-9]*.md; do
  is_opt_in "$generic" && continue
  id="$(rule_id "$generic")"
  lw="$(load_when "$generic")"
  if [ -n "$lw" ] && ! triggered "$lw"; then SKIPPED+=("$id"); continue; fi
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

echo "=== vbsec rules — PART $PART/$total_parts. Đọc toàn bộ output này; không cắt bằng head/tail ==="
for i in "${!FILES_SELECTED[@]}"; do
  [ "${part_of[$i]}" -eq "$PART" ] && detection "${FILES_SELECTED[$i]}"
done
if [ "$PART" -lt "$total_parts" ]; then
  echo "=== PART $PART/$total_parts — CHƯA ĐỦ: chạy tiếp --part $((PART + 1)) đến --part $total_parts ==="
else
  echo "=== PART $PART/$total_parts — đã in đủ ${#FILES_SELECTED[@]} rule ==="
  if [ ${#SKIPPED[@]} -gt 0 ]; then
    echo "=== Rule không nạp vì repo không có dấu hiệu liên quan (ghi vào coverage.rules_skipped): ${SKIPPED[*]} ==="
  fi
fi
