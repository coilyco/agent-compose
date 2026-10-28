#!/bin/sh
# Attach every built artifact to the GitHub release for ${TAG}.
#
# The set is read off dist/ rather than restated here. A hardcoded list drifted
# once: the roster and bundles tarballs joined release-build.sh and the rendered
# formula in v2.90.0 but never this script, so v2.90.0 through v2.94.0 each
# published a formula naming two assets no release carried.
set -eu
: "${TAG:?TAG is required}"
for asset in dist/*
do
  [ -f "$asset" ] || continue
  gh release upload "$TAG" "$asset" --clobber
  echo "uploaded $(basename "$asset")"
done
