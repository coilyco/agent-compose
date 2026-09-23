#!/bin/sh
# Exercise the shipped acompose entry point without touching the operator's home.
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
smoke_root=$(mktemp -d "${TMPDIR:-/tmp}/agent-compose-smoke.XXXXXX")

cleanup() {
  rm -rf -- "$smoke_root"
}
trap cleanup EXIT HUP INT TERM

fail() {
  printf 'smoke: %s\n' "$1" >&2
  exit 1
}

verbose=false
case "${1:-}" in
  "")
    ;;
  --verbose)
    verbose=true
    ;;
  *)
    fail "usage: scripts/smoke.sh [--verbose]"
    ;;
esac

show_transcript() {
  if "$verbose"; then
    printf 'smoke: %s transcript\n' "$1"
    sed 's/^/  /' "$2"
  fi
}

assert_file() {
  test -f "$1" || fail "missing file $1"
}

assert_contains() {
  grep -F "$2" "$1" >/dev/null || fail "$1 does not contain: $2"
}

assert_missing() {
  grep -F "$2" "$1" >/dev/null && fail "$1 unexpectedly contains: $2"
  return 0
}

snapshot_file() {
  cp "$1" "$snapshot_dir/$2"
}

assert_unchanged() {
  cmp -s "$1" "$snapshot_dir/$2" || fail "$1 changed during the second convergence"
}

goos=$(go env GOOS)
case "$goos" in
  windows)
    command -v cygpath >/dev/null 2>&1 || fail "cygpath is required by the Windows sh environment"
    native_root=$(cygpath -m "$smoke_root")
    binary="$native_root/bin/acompose.exe"
    binary_exec="$binary"
    ;;
  *)
    native_root=$(CDPATH= cd -- "$smoke_root" && pwd -P)
    binary="$smoke_root/bin/acompose"
    binary_exec="$binary"
    ;;
esac

fixture_home="$smoke_root/home"
state_dir="$fixture_home/.agent-compose"
projects_dir="$smoke_root/projects"
provider_dir="$projects_dir/coilyco-flight-deck/agentic-os"
skill_dir="$provider_dir/.agents/skills/coding-go"
composed_dir="$provider_dir/.agents/composed/design-method"
load_points="$smoke_root/load-points"
fixtures="$smoke_root/fixtures"
snapshot_dir="$smoke_root/snapshot"
launch_target="$smoke_root/native-launch"

mkdir -p "$smoke_root/bin" "$state_dir" "$skill_dir" "$composed_dir" \
  "$load_points" "$fixtures" "$snapshot_dir" "$launch_target"

cat >"$fixtures/AGENTS.COMPOSE.md" <<'EOF'
# Smoke doctrine

The smoke fixture exercises host convergence.
EOF

cat >"$skill_dir/SKILL.md" <<'EOF'
# Go

Use Go for the smoke fixture.
EOF

cat >"$composed_dir/COMPOSED.md" <<'EOF'
# Design method

Shape the complete interaction before implementation.
EOF

cat >"$provider_dir/.agents/roles.kdl" <<'EOF'
roles {
    role frontend-eng {
        composed-skill design-method
    }
}
EOF

# git -C wants a native path, and an MSYS mktemp path is not one, so each call
# changes directory in a subshell instead.
provider_git() {
  (cd "$provider_dir" && git "$@")
}

provider_git init -q
provider_git config user.name Fixture
provider_git config user.email fixture@example.test
provider_git config commit.gpgSign false
provider_git add .
provider_git commit -q -m fixture

cat >"$smoke_root/bin/codex" <<'EOF'
#!/bin/sh
printf 'fake codex'
printf ' <%s>' "$@"
printf '\n'
EOF
chmod +x "$smoke_root/bin/codex"

cat >"$state_dir/agent-compose.yaml" <<EOF
sources:
  - $native_root/fixtures/AGENTS.COMPOSE.md
roots:
  - $native_root/home/.agent-compose/sources
roster_sources:
  - $native_root/projects/coilyco-flight-deck/agentic-os
operating_context:
  - coilyco-flight-deck/agentic-os
skill_load_points:
  codex: $native_root/load-points/skills
load_points:
  claude: $native_root/load-points/CLAUDE.md
  codex: null
EOF

cd "$repo_root"
if ! go build -o "$binary" ./cmd/agent-compose; then
  fail "build failed"
fi
assert_file "$binary"
printf 'smoke: build real acompose binary... ok\n'

# A successful run says nothing since c5f84bb, so the convergence stages this
# asserts on are only emitted under --verbose.
first_output="$smoke_root/first.txt"
if ! env HOME="$native_root/home" USERPROFILE="$native_root/home" \
  PROJECTS_ROOT="$native_root/projects" "$binary_exec" --verbose >"$first_output" 2>&1; then
  cat "$first_output" >&2
  fail "first acompose convergence failed"
fi

assert_contains "$first_output" "roster  "
assert_contains "$first_output" "wrote"
assert_contains "$first_output" "cascade outputs=1 load-points=1 repository-plan=1 changed="
assert_contains "$first_output" "skills  managed="
printf 'smoke: first isolated convergence... ok\n'
show_transcript "first convergence" "$first_output"

roster_table="$state_dir/sources/personality/AGENTS.COMPOSE.md"
roster_override="$state_dir/sources/personality/AGENTS.claude.md"
roster_body="$state_dir/sources/personality/.agents/skills/personality-tenacious/SKILL.md"
person_snapshot="$state_dir/sources/personality/person.json"
composed="$state_dir/COMPOSED.md"
repository_plan="$state_dir/repository-plan.yaml"
skill_state="$state_dir/skill-mounts.json"
for path in "$roster_table" "$roster_override" "$roster_body" "$person_snapshot" "$composed" \
  "$repository_plan" "$skill_state" \
  "$load_points/CLAUDE.md" "$load_points/skills/coding-go/SKILL.md" \
  "$load_points/skills/role-platform-eng/SKILL.md" \
  "$load_points/skills/personality-tenacious/SKILL.md"; do
  assert_file "$path"
done

assert_contains "$composed" "# Smoke doctrine"
assert_contains "$composed" "# Personality invariant"
assert_contains "$composed" "# Agent seats"
# Roster names churn, so anchor on the identity render shape instead. A seat
# rename must not be able to fail this.
assert_contains "$composed" "**Agent // "
assert_contains "$composed" "**Seats // "
assert_contains "$person_snapshot" '"format": "agent-compose.person-snapshot.v3"'
assert_contains "$person_snapshot" '"briefing":'
printf 'smoke: roster, cascade, skill, and load-point artifacts... ok\n'

snapshot_file "$roster_table" roster-table
snapshot_file "$roster_override" roster-override
snapshot_file "$roster_body" roster-body
snapshot_file "$person_snapshot" person-snapshot
snapshot_file "$composed" composed
snapshot_file "$repository_plan" repository-plan
snapshot_file "$skill_state" skill-state
second_output="$smoke_root/second.txt"
if ! env HOME="$native_root/home" USERPROFILE="$native_root/home" \
  PROJECTS_ROOT="$native_root/projects" "$binary_exec" --verbose >"$second_output" 2>&1; then
  cat "$second_output" >&2
  fail "second acompose convergence failed"
fi

if grep -F "wrote" "$second_output" >/dev/null; then
  cat "$second_output" >&2
  fail "second convergence rewrote current state"
fi
assert_contains "$second_output" "cascade outputs=1 load-points=1 repository-plan=1 changed=0"
assert_contains "$second_output" "skills  managed="
assert_contains "$second_output" "linked=0 removed=0 preserved=0"
printf 'smoke: second isolated convergence reports unchanged state... ok\n'
show_transcript "second convergence" "$second_output"

assert_unchanged "$roster_table" roster-table
assert_unchanged "$roster_override" roster-override
assert_unchanged "$roster_body" roster-body
assert_unchanged "$person_snapshot" person-snapshot
assert_unchanged "$composed" composed
assert_unchanged "$repository_plan" repository-plan
assert_unchanged "$skill_state" skill-state
printf 'smoke: representative artifacts remain byte-stable... ok\n'

role_output="$smoke_root/role-launch.txt"
if ! (
  cd "$launch_target"
  unset AGENT_COMPOSE_LAUNCH
  env HOME="$native_root/home" USERPROFILE="$native_root/home" \
    PROJECTS_ROOT="$native_root/projects" PATH="$smoke_root/bin:$PATH" \
    "$binary_exec" frontend codex --version
) >"$role_output" 2>&1; then
  cat "$role_output" >&2
  fail "assigned native role launch failed"
fi

assert_contains "$role_output" "role metadata"
assert_contains "$role_output" "role: frontend-eng"
assert_contains "$role_output" "personality: imaginative"
assert_contains "$role_output" "fake codex <--version>"
for routine in \
  "agent-compose: assigned frontend to codex" \
  "cascade outputs=" \
  "skills  managed=" \
  "sources:" \
  "decisions:" \
  "trace:"; do
  assert_missing "$role_output" "$routine"
done
for path in \
  "$launch_target/AGENTS.md" \
  "$launch_target/.agents/skills/role-frontend-eng/SKILL.md" \
  "$launch_target/.agents/skills/personality-imaginative/SKILL.md" \
  "$launch_target/.agents/skills/design-method/SKILL.md"; do
  assert_file "$path"
done
assert_contains "$launch_target/AGENTS.md" '`frontend-eng` role'
printf 'smoke: assigned native role and composed skill projection... ok\n'
show_transcript "native role launch" "$role_output"

verbose_output="$smoke_root/role-launch-verbose.txt"
if ! (
  cd "$launch_target"
  unset AGENT_COMPOSE_LAUNCH
  env HOME="$native_root/home" USERPROFILE="$native_root/home" \
    PROJECTS_ROOT="$native_root/projects" PATH="$smoke_root/bin:$PATH" \
    AGENT_COMPOSE_VERBOSE=1 \
    "$binary_exec" frontend codex --version
) >"$verbose_output" 2>&1; then
  cat "$verbose_output" >&2
  fail "verbose native role launch failed"
fi

assert_contains "$verbose_output" "agent-compose: assigned frontend to codex"
assert_contains "$verbose_output" "cascade outputs="
assert_contains "$verbose_output" "sources:"
assert_contains "$verbose_output" "trace:"
assert_contains "$verbose_output" "role: frontend-eng"
printf 'smoke: AGENT_COMPOSE_VERBOSE restores the routine status... ok\n'
show_transcript "verbose native role launch" "$verbose_output"

intro_output="$smoke_root/role-introduction.txt"
if ! (
  cd "$launch_target"
  unset AGENT_COMPOSE_LAUNCH
  env HOME="$native_root/home" USERPROFILE="$native_root/home" \
    PROJECTS_ROOT="$native_root/projects" PATH="$smoke_root/bin:$PATH" \
    "$binary_exec" frontend codex
) >"$intro_output" 2>&1; then
  cat "$intro_output" >&2
  fail "bare Codex introduction launch failed"
fi
assert_contains "$intro_output" "fake codex <"
assert_contains "$intro_output" "Introduce yourself now as the active Codex seat"
printf 'smoke: bare Codex launch supplies its introduction prompt... ok\n'
show_transcript "bare Codex introduction" "$intro_output"

third_output="$smoke_root/third.txt"
if ! env HOME="$native_root/home" USERPROFILE="$native_root/home" \
  PROJECTS_ROOT="$native_root/projects" "$binary_exec" \
  --reapply --verbose >"$third_output" 2>&1; then
  cat "$third_output" >&2
  fail "verbose reapply convergence failed"
fi

assert_contains "$third_output" "layout  $native_root/fixtures/AGENTS.COMPOSE.md => "
assert_contains "$third_output" " => $native_root/load-points/CLAUDE.md"
assert_contains "$third_output" "wrote"
assert_contains "$third_output" "cascade outputs=1 load-points=1 repository-plan=1 changed=3"
printf 'smoke: verbose reapply traces and recreates the compose layout... ok\n'
show_transcript "verbose reapply" "$third_output"

printf 'smoke: acompose host convergence is healthy and idempotent\n'
