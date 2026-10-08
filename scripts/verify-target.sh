#!/usr/bin/env bash
# Checks one step of the clean-architecture experiment (docs/design/target/README.md).
#
#   scripts/verify-target.sh           guardrails; must pass after every step
#   scripts/verify-target.sh --target  guardrails plus the target architecture; passing it is "done"
#
# This script, internal/archtest/target_test.go and docs/design/target/ are
# frozen. The script fails if any of them differs from the commit that added
# target_test.go.
set -uo pipefail
cd "$(git rev-parse --show-toplevel)" || exit 1

base=$(git log --diff-filter=A --format=%H -- internal/archtest/target_test.go | tail -n 1)
if [ -z "$base" ]; then
	echo "FAIL: no commit adds internal/archtest/target_test.go" >&2
	exit 1
fi

failures=()
fail() { echo "FAIL: $*" >&2; failures+=("$*"); }
step() { printf '\n== %s\n' "$*"; }
testfiles() { find cmd internal -name '*_test.go' -not -path 'internal/archtest/target_test.go'; }

step "frozen spec and checks match ${base:0:12}"
if ! git diff --quiet "$base" -- docs/design/target internal/archtest/target_test.go scripts/verify-target.sh; then
	fail "the spec, target_test.go or this script changed. Restore them: git checkout $base -- docs/design/target internal/archtest/target_test.go scripts/verify-target.sh"
fi

step "build, vet, format, tidy"
go build ./... || fail "go build"
go vet ./... || fail "go vet"
unformatted=$(gofmt -l cmd internal)
[ -z "$unformatted" ] || fail "gofmt: $unformatted"
go mod tidy -diff >/dev/null || fail "go mod tidy -diff"

step "tests"
go test -race -count=1 ./... || fail "go test -race ./..."

step "lint"
make lint || fail "make lint"

step "CLI behavior unchanged: regenerated docs and the testscript goldens match the base"
go run ./internal/docs/gen >/dev/null || fail "go run ./internal/docs/gen"
if ! git diff --quiet "$base" -- cmd/invox/testdata docs/cli share/man; then
	git diff --stat "$base" -- cmd/invox/testdata docs/cli share/man >&2
	fail "testscript goldens, docs/cli or share/man differ from the base; output must not change"
fi

step "every baseline test function still exists, or is listed in docs/design/target-removed-tests.md"
current=$(testfiles | xargs grep -hoE '^func (Test|Fuzz)[A-Za-z0-9_]*' | sed 's/^func //' | sort -u)
ledger=docs/design/target-removed-tests.md
missing=()
while IFS= read -r name; do
	if ! grep -qx "$name" <<<"$current" && ! grep -q -- "^- $name:" "$ledger" 2>/dev/null; then
		missing+=("$name")
	fi
done <docs/design/target/baseline-tests.txt
[ ${#missing[@]} -eq 0 ] || fail "${#missing[@]} test functions disappeared without a ledger entry: ${missing[*]}"

step "no new t.Skip"
skips=$(testfiles | xargs grep -ho 't\.Skip' | wc -l | tr -d ' ')
baseline_skips=$(cat docs/design/target/baseline-skips.txt)
[ "$skips" -le "$baseline_skips" ] || fail "t.Skip count rose from $baseline_skips to $skips"

step "test fixtures unchanged (files may move, contents may not)"
have=$(find cmd internal -path '*/testdata/*' -type f -exec sha256sum {} + | cut -d' ' -f1 | sort -u)
changed=()
while read -r sum path; do
	grep -qx "$sum" <<<"$have" || changed+=("$path")
done <docs/design/target/baseline-fixtures.txt
[ ${#changed[@]} -eq 0 ] || fail "${#changed[@]} baseline fixtures changed or vanished: ${changed[*]}"

if [ "${1:-}" = "--target" ]; then
	step "target architecture"
	out=$(go test -tags target -count=1 ./internal/archtest -run TestTarget -v 2>&1)
	status=$?
	passed=$(grep -c -- '--- PASS: TestTarget' <<<"$out")
	failed=$(grep -c -- '--- FAIL: TestTarget' <<<"$out")
	grep -E -- '--- FAIL|target_test.go' <<<"$out"
	echo "target checks: $passed passed, $failed failed"
	[ "$status" -eq 0 ] || fail "target architecture not reached ($failed checks failing)"
fi

echo
if [ ${#failures[@]} -gt 0 ]; then
	echo "verify-target: ${#failures[@]} failure(s)" >&2
	exit 1
fi
echo "verify-target: OK"
