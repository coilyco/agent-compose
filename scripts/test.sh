#!/bin/sh
# Full repository validation: engine tests first, then the hook catalog.
set -e
test_home=$(mktemp -d)
cleanup() {
  chmod -R u+w "$test_home" 2>/dev/null || true
  rm -rf "$test_home"
}
trap cleanup EXIT HUP INT TERM

go test ./...
# The script checks below shell out to agent-compose, so give them this
# checkout's build rather than whatever release happens to be installed.
mkdir -p "$test_home/bin"
go build -o "$test_home/bin/agent-compose" ./cmd/agent-compose
PATH="$test_home/bin:$PATH"
export PATH
sh scripts/release-impact-test.sh
sh scripts/release-publish-package-test.sh
sh scripts/release-verify-assets-test.sh
sh scripts/release-attach-assets-test.sh
sh scripts/render-packaging-test.sh
env HOME="$test_home" sh scripts/palette-web.sh test
env HOME="$test_home" sh scripts/context-budget.sh
sh scripts/check.sh
# The hosted gate cannot clone the Forgejo hook pin yet: docs/release.md.
[ -n "${ACOMPOSE_SKIP_PRECOMMIT:-}" ] || pre-commit run --all-files
