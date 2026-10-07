#!/bin/sh
# Fixture coverage for the mirror check: the slug it reads comes from go.mod.
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
verifier="$script_dir/ci/release-verify-mirror.sh"
fixture_root=$(mktemp -d)
cleanup() {
  chmod -R u+w "$fixture_root" 2>/dev/null || true
  rm -rf "$fixture_root"
}
trap cleanup EXIT HUP INT TERM

printf 'module github.com/example-org/example-repo/v2\n\ngo 1.25\n' >"$fixture_root/go.mod"
mkdir -p "$fixture_root/bin"
# A curl that records the URL it was asked for and answers 200.
cat >"$fixture_root/bin/curl" <<'STUB'
#!/bin/sh
for arg in "$@"; do last=$arg; done
printf '%s\n' "$last" >"$FIXTURE_URL_FILE"
printf '200'
STUB
chmod +x "$fixture_root/bin/curl"

run_verifier() {
  PATH="$fixture_root/bin:$PATH" FIXTURE_URL_FILE="$fixture_root/url" \
    GO_MOD="$fixture_root/go.mod" GITHUB_API=https://api.example.test TAG=v9.9.9 \
    MIRROR_DELAY=0 "$@" sh "$verifier" >/dev/null
}

expect_url() {
  expected=$1
  shift
  run_verifier "$@"
  actual=$(cat "$fixture_root/url")
  if [ "$actual" != "$expected" ]; then
    echo "release-verify-mirror-test: wanted $expected, got $actual" >&2
    exit 1
  fi
}

expect_url https://api.example.test/repos/example-org/example-repo/git/ref/tags/v9.9.9 env
expect_url https://api.example.test/repos/other/slug/git/ref/tags/v9.9.9 env MIRROR_REPO=other/slug

printf 'module example.test/not-github\n' >"$fixture_root/go.mod"
if run_verifier env 2>/dev/null; then
  echo "release-verify-mirror-test: a go.mod without a github.com module must fail" >&2
  exit 1
fi
echo "release-verify-mirror-test: ok"
