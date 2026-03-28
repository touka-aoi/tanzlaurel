#!/usr/bin/env bash
# PostToolUse hook: run Go linters and formatters on edited .go files
set -euo pipefail

# Read file path from stdin JSON
f=$(jq -r '.tool_input.file_path // .tool_response.filePath')

# Skip non-Go files
echo "$f" | grep -q '\.go$' || exit 0

# Find the go.mod directory
dir=$(cd "$(dirname "$f")" && while [ ! -f go.mod ] && [ "$(pwd)" != "/" ]; do cd ..; done && pwd)
[ -n "$dir" ] && [ -f "$dir/go.mod" ] || exit 0

cd "$dir"

# Run Go tools
go fix ./... >/dev/null 2>&1 || true
golangci-lint run --fix ./... >/dev/null 2>&1 || true
gofumpt -w "$f" 2>/dev/null || true
