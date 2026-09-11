#!/bin/sh
# Fixture coverage for the owning release-impact classifier.
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
classifier="$script_dir/release-impact.sh"
fixture_root=$(mktemp -d)
cleanup() {
  chmod -R u+w "$fixture_root" 2>/dev/null || true
  rm -rf "$fixture_root"
}
trap cleanup EXIT HUP INT TERM

# git -C wants a native path, and an MSYS mktemp path is not one, so each call
# changes directory in a subshell instead.
fixture_git() {
  (cd "$fixture_root" && git "$@")
}

fixture_git init -q
fixture_git config user.name Fixture
fixture_git config user.email fixture@example.test
fixture_git config commit.gpgSign false
fixture_git config tag.gpgSign false
mkdir -p "$fixture_root/cmd/tool" "$fixture_root/docs" "$fixture_root/evaluations/latest"
printf 'package main\n' >"$fixture_root/cmd/tool/main.go"
printf '# Guide\n' >"$fixture_root/docs/guide.md"
fixture_git add .
fixture_git commit -q -m base
base=$(fixture_git rev-parse HEAD)
fixture_git tag v1.0.0 "$base"

expect() {
  expected=$1
  event_name=$2
  before=$3
  after=$4
  actual=$(
    cd "$fixture_root"
    RELEASE_EVENT="$event_name" RELEASE_BASE="$before" RELEASE_HEAD="$after" \
      "$classifier"
  )
  if [ "$actual" != "$expected" ]; then
    echo "release-impact-test: got $actual, want $expected" >&2
    exit 1
  fi
}

printf 'More docs.\n' >>"$fixture_root/docs/guide.md"
fixture_git add docs/guide.md
fixture_git commit -q -m docs
docs_revision=$(fixture_git rev-parse HEAD)
expect false push "$base" "$docs_revision"

printf 'format: fixture\n' >"$fixture_root/evaluations/latest/result.yaml"
fixture_git add evaluations/latest/result.yaml
fixture_git commit -q -m results
results_revision=$(fixture_git rev-parse HEAD)
expect false push "$docs_revision" "$results_revision"

# The seed ships inside the package, so a roster-only revision is product
# impact even though nothing rebuilds. agent-compose#7397.
mkdir -p "$fixture_root/seed/roster/data/role-fixture"
printf 'role: fixture\n' >"$fixture_root/seed/roster/data/role-fixture/role.yaml"
fixture_git add seed
fixture_git commit -q -m seed
seed_revision=$(fixture_git rev-parse HEAD)
expect true push "$results_revision" "$seed_revision"

printf '// product change\n' >>"$fixture_root/cmd/tool/main.go"
fixture_git add cmd/tool/main.go
fixture_git commit -q -m product
product_revision=$(fixture_git rev-parse HEAD)
expect true push "$seed_revision" "$product_revision"
expect true push 0000000000000000000000000000000000000000 "$product_revision"

printf 'format: recovery\n' >"$fixture_root/evaluations/latest/result.yaml"
fixture_git add evaluations/latest/result.yaml
fixture_git commit -q -m recovery-results
recovery_revision=$(fixture_git rev-parse HEAD)
expect true push "$product_revision" "$recovery_revision"
fixture_git tag v1.1.0 "$recovery_revision"

printf 'Released docs.\n' >>"$fixture_root/docs/guide.md"
fixture_git add docs/guide.md
fixture_git commit -q -m released-docs
released_docs_revision=$(fixture_git rev-parse HEAD)
expect false push "$recovery_revision" "$released_docs_revision"

printf 'v2.0.0\n' >"$fixture_root/.release-major"
printf '// held product change\n' >>"$fixture_root/cmd/tool/main.go"
fixture_git add .release-major cmd/tool/main.go
fixture_git commit -q -m held
held_revision=$(fixture_git rev-parse HEAD)
expect false push "$released_docs_revision" "$held_revision"
expect true workflow_dispatch "$held_revision" "$held_revision"

echo "release-impact-test: ok"
