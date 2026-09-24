#!/usr/bin/env bash
# load-rules.sh — in phần PHÁT HIỆN của bộ rule hiệu lực cho các ngôn ngữ đã detect.
#
# Usage: bash <skill-dir>/references/load-rules.sh [lang ...]
#   vd:  bash references/load-rules.sh typescript
#        bash references/load-rules.sh go typescript     # repo đa ngôn ngữ
#        bash references/load-rules.sh                   # không có overlay → chỉ generic
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
LANGS=("$@")

rule_id() { sed -n 's/^id:[[:space:]]*//p' "$1" | head -1; }

overlay_for() {  # overlay_for <lang> <id> → path hoặc rỗng
  local dir="$SKILL_DIR/rules/languages/$1" f
  [ -d "$dir" ] || return 0
  for f in "$dir"/[0-9]*.md; do
    [ -f "$f" ] && [ "$(rule_id "$f")" = "$2" ] && { echo "$f"; return 0; }
  done
  return 0
}

print_detection() {  # print_detection <path>
  local rel="${1#"$SKILL_DIR"/}"
  echo "=== RULE $(rule_id "$1") (source: $rel) ==="
  awk '/^## (Examples|Fix recommendation|Cross-references)/ { exit } { print }' "$1"
  echo
}

for generic in "$SKILL_DIR"/rules/generic/[0-9]*.md; do
  id="$(rule_id "$generic")"
  overlays=()
  all_covered=1
  for lang in ${LANGS[@]+"${LANGS[@]}"}; do
    o="$(overlay_for "$lang" "$id")"
    if [ -n "$o" ]; then overlays+=("$o"); else all_covered=0; fi
  done
  [ ${#LANGS[@]} -eq 0 ] && all_covered=0
  [ "$all_covered" -eq 1 ] || print_detection "$generic"
  for o in ${overlays[@]+"${overlays[@]}"}; do print_detection "$o"; done
done
