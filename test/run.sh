#!/usr/bin/env bash
set -euo pipefail
root=$(cd "$(dirname "$0")/.." && pwd)
bash "$root/test/unit.sh"
python3 "$root/test/pty_test.py"
if command -v shellcheck >/dev/null 2>&1; then
  shellcheck -S warning "$root/gh-sweep" "$root/test/unit.sh" "$root/test/mock-gh"
fi
