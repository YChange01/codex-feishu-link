#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${root_dir}"

# Keep the retired identifiers assembled so this guard does not flag itself.
legacy_repo="codex-re""mote-feishu"
legacy_app="codex-re""mote"
legacy_brand="Codex Re""mote"
legacy_env="CODEX_""REMOTE"
legacy_go="Codex""Remote"
legacy_spaced="codex ""remote"

failed=0
for identifier in "${legacy_repo}" "${legacy_app}" "${legacy_brand}" "${legacy_env}" "${legacy_go}" "${legacy_spaced}"; do
  if rg -n --hidden --fixed-strings \
    --glob '!**/.git/**' \
    --glob '!**/node_modules/**' \
    --glob '!NOTICE' \
    --glob '!scripts/check/no-legacy-names.sh' \
    "${identifier}" .; then
    failed=1
  fi
done

if (( failed )); then
  echo "Found a retired project identifier." >&2
  exit 1
fi
