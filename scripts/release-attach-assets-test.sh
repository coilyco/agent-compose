#!/bin/sh
# Fixture coverage for the asset attach: the set comes off dist/, files only.
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
attach="$script_dir/ci/release-attach-assets.sh"
fixture_root=$(mktemp -d)
cleanup() {
  chmod -R u+w "$fixture_root" 2>/dev/null || true
  rm -rf "$fixture_root"
}
trap cleanup EXIT HUP INT TERM

mkdir -p "$fixture_root/bin" "$fixture_root/dist/bundles"
for name in agent-compose-linux-amd64 agent-compose.rb SHA256SUMS; do
  printf 'x\n' >"$fixture_root/dist/$name"
done
# A gh that records its arguments, one call per line.
cat >"$fixture_root/bin/gh" <<'STUB'
#!/bin/sh
printf '%s\n' "$*" >>"$FIXTURE_CALLS"
STUB
chmod +x "$fixture_root/bin/gh"

(
  cd "$fixture_root"
  PATH="$fixture_root/bin:$PATH" FIXTURE_CALLS="$fixture_root/calls" TAG=v9.9.9 sh "$attach" >/dev/null
)

# Glob order follows the locale, so compare the sorted calls.
expected="release upload v9.9.9 dist/SHA256SUMS --clobber
release upload v9.9.9 dist/agent-compose-linux-amd64 --clobber
release upload v9.9.9 dist/agent-compose.rb --clobber"
if [ "$(LC_ALL=C sort "$fixture_root/calls")" != "$expected" ]; then
  echo "release-attach-assets-test: wanted one clobbering upload per file in dist/, got:" >&2
  cat "$fixture_root/calls" >&2
  exit 1
fi
if (cd "$fixture_root" && PATH="$fixture_root/bin:$PATH" FIXTURE_CALLS="$fixture_root/calls" sh "$attach" 2>/dev/null); then
  echo "release-attach-assets-test: a missing TAG must fail" >&2
  exit 1
fi

echo "release-attach-assets-test: ok"
