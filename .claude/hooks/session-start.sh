#!/bin/bash
# Prepares Claude Code on the web sessions so `go build`, `go vet` and
# `go test` work immediately. The container is cached after this runs, so
# downloaded modules and the warmed build cache carry over to later sessions.
set -euo pipefail

if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

cd "${CLAUDE_PROJECT_DIR:-$(dirname "$0")/../..}"

go mod download
# Compile packages and test binaries without running any tests.
go build ./...
go test -count=1 -run '^$' ./... >/dev/null
