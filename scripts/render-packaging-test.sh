#!/bin/sh
# Render package metadata at a fixture major tag and inspect its public contract.
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
renderer="$script_dir/render-packaging.sh"
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
mkdir -p "$fixture_root/dist"
printf 'fixture\n' >"$fixture_root/release-input"
fixture_git add release-input
fixture_git commit -q -m release
fixture_git -c tag.gpgSign=false tag v2.0.0

for artifact in \
  agent-compose-darwin-arm64 \
  agent-compose-linux-amd64 \
  agent-compose-linux-arm64 \
  agent-compose-windows-amd64.exe
do
  printf '%s\n' "$artifact" >"$fixture_root/dist/$artifact"
done

# Real archives, tarred the way release-build.sh tars them, because the install
# contract depends on their top-level shape rather than on their contents.
mkdir -p "$fixture_root/seed/roster/data" "$fixture_root/seed/bundles/one"
printf 'fixture\n' >"$fixture_root/seed/roster/data/role.yaml"
printf 'fixture\n' >"$fixture_root/seed/bundles/one/content.md"
tar -czf "$fixture_root/dist/agent-compose-roster.tar.gz" -C "$fixture_root/seed" roster
tar -czf "$fixture_root/dist/agent-compose-bundles.tar.gz" -C "$fixture_root/seed" bundles

(
  cd "$fixture_root"
  sh "$renderer"
)

formula="$fixture_root/dist/agent-compose.rb"
manifest="$fixture_root/dist/agent-compose.json"
for expected in \
  'Core Roster context composition for native agent harnesses' \
  'v2.0.0/agent-compose' \
  '2.0.0'
do
  if ! grep -F "$expected" "$formula" "$manifest" >/dev/null; then
    echo "render-packaging-test: missing $expected" >&2
    exit 1
  fi
done
if ! grep -F '"agent-compose-windows-amd64.exe", "acompose", "compose"' "$manifest" >/dev/null; then
  echo "render-packaging-test: Scoop alias contract is missing" >&2
  exit 1
fi
for expected in 'agent-compose-roster.tar.gz' 'agent-compose-bundles.tar.gz' 'share/"agent-compose"'
do
  if ! grep -F "$expected" "$formula" "$manifest" >/dev/null; then
    echo "render-packaging-test: seed roster is not installed: $expected" >&2
    exit 1
  fi
done

# A stage block naming its own lone top-level directory raises ENOENT, which
# shipped in v2.90.0 through v2.95.0 unread above: agentic-os#6835.
for resource_name in roster bundles
do
  archive="$fixture_root/dist/agent-compose-$resource_name.tar.gz"
  roots=$(tar tzf "$archive" | awk -F/ 'NF { print $1 }' | sort -u)
  if [ "$(printf '%s\n' "$roots" | wc -l)" -ne 1 ]; then
    continue
  fi
  if grep -F "install \"$roots\"" "$formula" >/dev/null; then
    echo "render-packaging-test: $resource_name stage block names \"$roots\", the directory Homebrew has already chdir'd into" >&2
    exit 1
  fi
  if ! grep -F "resource(\"$resource_name\").stage(share/\"agent-compose\"/\"$resource_name\")" "$formula" >/dev/null; then
    echo "render-packaging-test: $resource_name must stage to an explicit target, not a block" >&2
    exit 1
  fi
done

# Outsiders fail fast with one sentence: every download uses the tailnet strategy
# and the Scoop manifest probes the host. teable:coilyco/homebrew-tap#8682
urls=$(grep -c '^ *url "' "$formula")
strategies=$(grep -c 'using: TailnetCurlDownloadStrategy' "$formula")
if [ "$urls" -ne "$strategies" ] || [ "$urls" -lt 5 ]; then
  echo "render-packaging-test: $strategies of $urls formula downloads use the tailnet strategy" >&2
  exit 1
fi
if ! grep -F 'require_relative "../lib/tailnet_download_strategy"' "$formula" >/dev/null; then
  echo "render-packaging-test: the formula does not load the tailnet strategy" >&2
  exit 1
fi
if ! grep -F "only available on the coilyco tailnet" "$manifest" >/dev/null ||
  ! grep -F "ConnectAsync('forgejo.coilysiren.me', 443)" "$manifest" >/dev/null; then
  echo "render-packaging-test: the Scoop manifest does not probe the tailnet host" >&2
  exit 1
fi

echo "render-packaging-test: ok"
