#!/usr/bin/env bash
# Checks one step of the clean-architecture work (docs/design/target/README.md).
#
#   scripts/verify-target.sh           guardrails; must pass after every commit
#   scripts/verify-target.sh --target  guardrails plus the target architecture tests
#
# The baseline is the last commit that touched docs/design/target/BASELINE. Only the
# coordinator changes that file, together with any change to the frozen files:
# docs/design/target/, internal/archtest/target_test.go and this script.
#
# Since the baseline, CLI output may change, but only on the record: every changed
# testscript golden, docs/cli or share/man page, and every changed or removed test
# fixture must be named in docs/design/output-changes.md or a file in
# docs/design/output-changes/, and every removed test function, as "- Name: reason",
# in docs/design/target-removed-tests.md or a file in docs/design/removed-tests/.
set -uo pipefail
cd "$(git rev-parse --show-toplevel)" || exit 1

base=$(git log -1 --format=%H -- docs/design/target/BASELINE)
if [ -z "$base" ]; then
	echo "FAIL: no commit touches docs/design/target/BASELINE" >&2
	exit 1
fi
changes=docs/design/output-changes.md
# A unit records its changes in its own file so parallel units don't conflict.
listed() { cat "$changes" docs/design/output-changes/*.md 2>/dev/null | grep -qF -- "$1"; }
removed() { cat docs/design/target-removed-tests.md docs/design/removed-tests/*.md 2>/dev/null | grep -q -- "^- $1:"; }

failures=()
fail() { echo "FAIL: $*" >&2; failures+=("$*"); }
step() { printf '\n== %s\n' "$*"; }
testfiles() { find cmd internal -name '*_test.go' -not -path 'internal/archtest/target_test.go'; }

step "frozen spec and checks match the baseline ${base:0:12}"
if ! git diff --quiet "$base" -- docs/design/target internal/archtest/target_test.go scripts/verify-target.sh; then
	fail "the spec, target_test.go or this script changed since the baseline. Only the coordinator changes them, with docs/design/target/BASELINE. Restore: git checkout $base -- docs/design/target internal/archtest/target_test.go scripts/verify-target.sh"
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

step "generated docs are committed"
go run ./internal/docs/gen >/dev/null || fail "go run ./internal/docs/gen"
if ! git diff --quiet -- docs/cli share/man; then
	git diff --stat -- docs/cli share/man >&2
	fail "docs/cli or share/man are stale; run make docs and commit the result"
fi

step "every output change since ${base:0:12} is listed in $changes"
unlisted=()
while IFS= read -r path; do
	[ -n "$path" ] || continue
	listed "$path" || unlisted+=("$path")
done < <(git diff --name-only "$base" -- cmd/invox/testdata docs/cli share/man)
[ ${#unlisted[@]} -eq 0 ] || fail "${#unlisted[@]} changed output files are not named in docs/design/output-changes/: ${unlisted[*]}"

step "every baseline test function still exists, or is listed as removed"
current=$(testfiles | xargs grep -hoE '^func (Test|Fuzz)[A-Za-z0-9_]*' | sed 's/^func //' | sort -u)
ledger=docs/design/target-removed-tests.md
missing=()
while IFS= read -r name; do
	if ! grep -qx "$name" <<<"$current" && ! removed "$name"; then
		missing+=("$name")
	fi
done <docs/design/target/baseline-tests.txt
[ ${#missing[@]} -eq 0 ] || fail "${#missing[@]} test functions disappeared without an entry in docs/design/removed-tests/: ${missing[*]}"

step "no new t.Skip"
skips=$(testfiles | xargs grep -ho 't\.Skip' | wc -l | tr -d ' ')
baseline_skips=$(cat docs/design/target/baseline-skips.txt)
[ "$skips" -le "$baseline_skips" ] || fail "t.Skip count rose from $baseline_skips to $skips"

step "test fixtures unchanged, moved, or listed in $changes"
have=$(find cmd internal -path '*/testdata/*' -type f -exec sha256sum {} + | cut -d' ' -f1 | sort -u)
changed=()
while read -r sum path; do
	grep -qx "$sum" <<<"$have" || listed "$path" || changed+=("$path")
done <docs/design/target/baseline-fixtures.txt
[ ${#changed[@]} -eq 0 ] || fail "${#changed[@]} baseline fixtures changed or vanished without an entry in $changes: ${changed[*]}"

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
