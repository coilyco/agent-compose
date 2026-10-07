#!/bin/sh
# Fail the release unless the GitHub mirror actually carries the tag.
#
# GitHub is the module origin: `go.mod` names github.com, so proxy.golang.org
# fetches the tag from there and a tag that never arrives is a release no Go
# consumer can resolve. Nothing else in this pipeline looks at the mirror, so
# without this a failed sync is green everywhere and invisible.
#
# Verifies rather than pushes. Something already syncs refs to the mirror
# outside this workflow, and a second pusher would race it. A read proves the
# property either way and needs no credential, because the mirror is public.
# See docs/release.md.

set -eu

: "${TAG:?TAG is required}"
# go.mod owns the slug: a restated default went stale when the module moved
# orgs, and GitHub answers the old slug with a 301 that no 200 check passes.
module=$(sed -n 's|^module github\.com/\([^/]*/[^/]*\).*|\1|p' "${GO_MOD:-go.mod}")
MIRROR="${MIRROR_REPO:-$module}"
: "${MIRROR:?MIRROR_REPO is unset and go.mod names no github.com module}"
API="${GITHUB_API:-https://api.github.com}"
# Tags reach the mirror 8.5 to 9 minutes late (v2.200.0, v2.201.0), so a first
# miss means nothing. The 800s ceiling clears that, under the job timeout.
ATTEMPTS="${MIRROR_ATTEMPTS:-40}"
DELAY="${MIRROR_DELAY:-10}"
CURL_TIMEOUT="${MIRROR_CURL_TIMEOUT:-10}"

url="$API/repos/$MIRROR/git/ref/tags/$TAG"
attempt=1

while [ "$attempt" -le "$ATTEMPTS" ]; do
  code=$(curl -s -o /dev/null -w '%{http_code}' -m "$CURL_TIMEOUT" "$url") && rc=0 || rc=$?
  if [ "$code" = "200" ]; then
    echo "release-verify-mirror: $MIRROR carries $TAG after ${attempt} attempt(s)."
    exit 0
  fi
  # 403 is the anonymous rate limit rather than a missing tag, so it is worth
  # saying out loud: the retry is fine, a wall of them is the real problem.
  [ "$code" = "403" ] && echo "release-verify-mirror: rate limited, retrying." >&2
  [ "$code" = "301" ] && echo "release-verify-mirror: $MIRROR moved on GitHub, fix its slug." >&2
  # Every attempt took the full timeout in the 10-02 to 10-07 failures, which the
  # old message hid: 000 with curl exit 28 is no response, not a missing tag.
  [ "$code" != "404" ] && echo "release-verify-mirror: attempt $attempt got HTTP $code, curl exit $rc." >&2
  attempt=$((attempt + 1))
  sleep "$DELAY"
done

echo "::error::release-verify-mirror: $MIRROR does not carry $TAG after $((ATTEMPTS * (DELAY + CURL_TIMEOUT)))s at worst." >&2
echo "Last attempt: HTTP $code, curl exit $rc (000 and 28 mean no response in ${CURL_TIMEOUT}s)." >&2
echo "GitHub is the module origin, so this release is unresolvable to Go consumers." >&2
echo "The tag exists on Forgejo and the packages are already published. Check the" >&2
echo "push mirror, then re-run this job." >&2
exit 1
