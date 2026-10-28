#!/bin/sh
# Fixture coverage for the asset check: matching bytes pass, a missing or
# different asset fails, and a late asset is retried.
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
verifier="$script_dir/ci/release-verify-assets.sh"
fixture_root=$(mktemp -d)
cleanup() {
  chmod -R u+w "$fixture_root" 2>/dev/null || true
  rm -rf "$fixture_root"
}
trap cleanup EXIT HUP INT TERM

mkdir -p "$fixture_root/bin" "$fixture_root/dist" "$fixture_root/served"
printf 'binary\n' >"$fixture_root/dist/agent-compose-linux-amd64"
printf 'sums\n' >"$fixture_root/dist/agent-compose-roster.tar.gz"
base=https://github.com/example-org/example-repo/releases/download/v9.9.9
printf '  url "%s/agent-compose-linux-amd64"\n  url "%s/agent-compose-roster.tar.gz"\n' "$base" "$base" >"$fixture_root/dist/agent-compose.rb"
printf '{"homepage": "https://github.com/example-org/example-repo"}\n' >"$fixture_root/dist/agent-compose.json"

# A curl that serves $FIXTURE_SERVED/<name> and fails the first FIXTURE_MISSES calls.
cat >"$fixture_root/bin/curl" <<'STUB'
#!/bin/sh
for arg in "$@"; do last=$arg; done
while [ "$#" -gt 1 ]; do
  [ "$1" = -o ] && out=$2
  shift
done
n=$(cat "$FIXTURE_COUNT" 2>/dev/null || echo 0)
printf '%s' $((n + 1)) >"$FIXTURE_COUNT"
[ "$n" -lt "${FIXTURE_MISSES:-0}" ] && exit 22
cp "$FIXTURE_SERVED/${last##*/}" "$out" 2>/dev/null || exit 22
STUB
chmod +x "$fixture_root/bin/curl"

serve() { cp "$fixture_root/dist/$1" "$fixture_root/served/$1"; }
run_verifier() {
  rm -f "$fixture_root/count"
  (cd "$fixture_root" && PATH="$fixture_root/bin:$PATH" FIXTURE_SERVED="$fixture_root/served" \
    FIXTURE_COUNT="$fixture_root/count" VERIFY_DELAY=0 "$@" sh "$verifier" >/dev/null 2>&1)
}

serve agent-compose-linux-amd64
if run_verifier env VERIFY_ATTEMPTS=2; then
  echo "release-verify-assets-test: a release missing an asset must fail" >&2
  exit 1
fi
serve agent-compose-roster.tar.gz
run_verifier env VERIFY_ATTEMPTS=2 || { echo "release-verify-assets-test: matching assets must pass" >&2; exit 1; }
run_verifier env VERIFY_ATTEMPTS=3 FIXTURE_MISSES=2 || { echo "release-verify-assets-test: a late asset must be retried" >&2; exit 1; }
printf 'tampered\n' >"$fixture_root/served/agent-compose-roster.tar.gz"
if run_verifier env VERIFY_ATTEMPTS=2; then
  echo "release-verify-assets-test: an asset with other bytes must fail" >&2
  exit 1
fi

echo "release-verify-assets-test: ok"
