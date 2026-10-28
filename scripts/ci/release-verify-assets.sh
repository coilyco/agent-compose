#!/bin/sh
# Fail the release unless every URL the rendered package metadata names serves
# the bytes dist/ holds.
#
# This is what brew and scoop do, anonymously, so it runs before the tap and
# bucket bumps: a release whose assets did not arrive never reaches a channel.
# v2.90.0 shipped a formula naming assets no release carried and nothing looked.
# See docs/release.md.

set -eu

ATTEMPTS="${VERIFY_ATTEMPTS:-6}"
DELAY="${VERIFY_DELAY:-5}"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT HUP INT TERM

sha() { { shasum -a 256 "$1" 2>/dev/null || sha256sum "$1"; } | cut -d' ' -f1; }

urls=$(grep -h -o 'https://[^" ,]*/releases/download/[^" ,]*' dist/agent-compose.rb dist/agent-compose.json | sort -u)
if [ -z "$urls" ]; then
  echo "::error::release-verify-assets: the rendered metadata names no release download" >&2
  exit 1
fi

for url in $urls; do
  name="${url##*/}"
  want=$(sha "dist/$name")
  attempt=1
  while :; do
    if curl -fsSL --connect-timeout 10 --max-time 180 -o "$work/$name" "$url" &&
      [ "$(sha "$work/$name")" = "$want" ]; then
      echo "release-verify-assets: $name serves the built bytes after ${attempt} attempt(s)."
      break
    fi
    if [ "$attempt" -ge "$ATTEMPTS" ]; then
      echo "::error::release-verify-assets: $url is missing or differs from dist/$name after $ATTEMPTS attempts." >&2
      exit 1
    fi
    attempt=$((attempt + 1))
    sleep "$DELAY"
  done
done
