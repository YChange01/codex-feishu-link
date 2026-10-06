#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
python3 - "${ROOT_DIR}" <<'PY'
from pathlib import Path
import sys

root = Path(sys.argv[1])
violations = []
for tree in ("cmd", "internal", "testkit"):
    base = root / tree
    if not base.is_dir():
        continue
    for path in base.rglob("*.go"):
        line_count = path.read_bytes().count(b"\n")
        is_test = path.name.endswith("_test.go")
        limit = 2000 if is_test else 1000
        if line_count > limit:
            kind = "test" if is_test else "business"
            violations.append((line_count, limit, kind, path.relative_to(root)))

if violations:
    print("Go file length limits exceeded. Split the files locally before committing.", file=sys.stderr)
    print("Limits: business files <= 1000 lines, test files <= 2000 lines.\n", file=sys.stderr)
    for lines, limit, kind, path in sorted(violations, reverse=True):
        print(f"  - {path}: {lines} lines (limit {limit}, {kind} file)", file=sys.stderr)
    raise SystemExit(1)
PY
