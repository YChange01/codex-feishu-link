#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

escape_regex() {
  printf '%s' "$1" | sed -e 's/[.[\*^$()+?{|\\]/\\&/g'
}

repo_root_pattern="$(escape_regex "${ROOT_DIR}")"
repo_parent="$(dirname "${ROOT_DIR}")"
repo_parent_pattern="$(escape_regex "${repo_parent}")"
home_pattern=""
if [[ -n "${HOME:-}" ]]; then
  home_pattern="$(escape_regex "${HOME}")"
fi

patterns=(
  "${repo_root_pattern}"
  "${repo_parent_pattern}/fschannel[[:alnum:]_-]*"
  '(^|[^[:alnum:]_])/(home|Users)/[^/"'"'"'[:space:]]+/'
  '(^|[^[:alnum:]_])/private/var/folders/[^/"'"'"'[:space:]]+/[^/"'"'"'[:space:]]+/'
  '(^|[^[:alnum:]_])[A-Za-z]:\\\\Users\\\\[^\\/"'"'"'[:space:]]+\\\\'
)
if [[ -n "${home_pattern}" ]]; then
  patterns+=("${home_pattern}")
fi

files=()
while IFS= read -r file; do
  case "${file}" in
    scripts/check/no-local-paths.sh|scripts/check/no-legacy-names.sh)
      continue
      ;;
  esac
  files+=("${file#./}")
done < <(
  find . -type f \
    -not -path './.git/*' \
    -not -path '*/node_modules/*' \
    \( -name '*.go' -o -name '*.md' -o -name '*.yml' -o -name '*.yaml' \
       -o -name '*.json' -o -name '*.sh' -o -name '*.ps1' -o -name '*.ts' \
       -o -name '*.tsx' -o -name '*.js' -o -name '*.jsx' -o -name '*.css' \
       -o -name '*.html' -o -name '*.txt' \) -print
)

if [[ ${#files[@]} -eq 0 ]]; then
  exit 0
fi

allowlisted_demo_pattern='/(home|Users)/(demo|example|sample|test)(/|$)'

raw_matches=""
if command -v rg >/dev/null 2>&1; then
  args=()
  for pattern in "${patterns[@]}"; do
    args+=(-e "${pattern}")
  done
  raw_matches="$(rg -n "${args[@]}" -- "${files[@]}" || true)"
else
  combined_pattern="$(IFS='|'; printf '%s' "${patterns[*]}")"
  raw_matches="$(grep -nE "${combined_pattern}" "${files[@]}" || true)"
fi

filtered_matches="$(printf '%s\n' "${raw_matches}" | grep -vE "${allowlisted_demo_pattern}" || true)"
filtered_matches="$(printf '%s\n' "${filtered_matches}" | sed '/^$/d')"

if [[ -n "${filtered_matches}" ]]; then
  printf '%s\n' "${filtered_matches}"
  echo "Found machine-local absolute paths in tracked source or docs." >&2
  exit 1
fi
