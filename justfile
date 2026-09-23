# Per-repo task manifest. Run `just` (or `just --list`) to see every verb.
#
# Recipes take trailing arguments directly: `just test -k parity`, where the
# retired form was `ward exec test -- -k parity`.
#
# One line of comment per recipe on purpose: just reads only the LAST comment
# line above a recipe, so a wrapped description silently truncates to its tail.
# That is agentic-os#1048's finding, kept here rather than rediscovered.
#
# `ward exec` is retired, and so is the `.ward/ward.yaml` that outlived it.

set positional-arguments

# The binary embeds no roster, so every recipe mounts the repository seed the
# way an installed package mounts its own. See docs/person-packages.md.
export AGENT_COMPOSE_ROSTER := justfile_directory() / "seed" / "roster"

# Default target: list every available recipe.
default:
    @just --list --unsorted

# Build acompose and report isolated host convergence stages.
smoke *ARGS:
    @sh scripts/smoke.sh "$@"

# Run the smoke with captured acompose convergence transcripts.
smoke-verbose *ARGS:
    @sh scripts/smoke.sh --verbose "$@"

# Run the Go suite, the parity checks, and the pre-commit suite.
test *ARGS:
    @sh scripts/test.sh "$@"

# Compile every package.
build *ARGS:
    @go build ./... "$@"

# Run Go static analysis.
lint *ARGS:
    @go vet ./... "$@"

# Format Go source files.
fmt *ARGS:
    @gofmt -w . "$@"

# Install the agent-compose binary into GOBIN.
install *ARGS:
    @go install ./cmd/agent-compose "$@"

# Reconcile go.mod and go.sum with the source imports.
tidy *ARGS:
    @go mod tidy "$@"

# Run the complete pre-commit suite explicitly.
pre-commit *ARGS:
    @pre-commit run --all-files "$@"

# Re-extract the vendored Claude Code verb and theme-token lists from the installed binary.
harness-refresh *ARGS:
    @go test ./internal/nativeui -run TestVendoredHarnessData -update -v "$@"

# Generate canonical palette data and build the local explorer.
palette-build *ARGS:
    @sh scripts/palette-web.sh build "$@"

# Type-check, build, and verify the local palette explorer.
palette-test *ARGS:
    @sh scripts/palette-web.sh test "$@"

# Generate canonical palette data and serve the local explorer.
palette-serve *ARGS:
    @sh scripts/palette-web.sh dev "$@"

# Reconcile the local explorer package lock with package.json.
palette-tidy *ARGS:
    @npm --prefix web/personality-palette install --package-lock-only --ignore-scripts --no-audit --no-fund "$@"

# Compose every shipped bundle from the roster into dist/bundles.
compose-bundles *ARGS:
    @uv run python scripts/compose-bundles.py dist/bundles "$@"

# Cross-compile version-stamped release binaries into dist/.
release-build *ARGS:
    @sh scripts/release-build.sh "$@"

# Render the brew formula and scoop manifest from dist/ binaries.
release-package *ARGS:
    @sh scripts/render-packaging.sh "$@"

# Render v2 fixture package metadata and inspect its public contract.
release-package-test *ARGS:
    @sh scripts/render-packaging-test.sh "$@"

# Decide whether one validated revision should publish a product release.
release-impact *ARGS:
    @sh scripts/release-impact.sh "$@"

# Exercise documentation, result, product, hold, and manual release fixtures.
release-impact-test *ARGS:
    @sh scripts/release-impact-test.sh "$@"

# Lint, format-check, type-check, and test evalkit.
check *ARGS:
    @sh scripts/check.sh "$@"

# Re-resolve uv.lock after a pyproject dependency pin moves.
lock *ARGS:
    @uv lock "$@"

# Reconcile the checks virtualenv with pyproject.toml.
sync *ARGS:
    @uv sync "$@"

# Compose one compiled bundle per role as the eval system prompts.
evalkit-prompts *ARGS:
    @sh scripts/eval-prompts.sh "$@"

# One live request through Agent Proxy, before a full board run.
evalkit-smoke *ARGS:
    @sh scripts/eval-smoke.sh "$@"

# Run the board through Inspect against Agent Proxy.
evalkit-run *ARGS:
    @sh scripts/eval-run.sh "$@"

# Open the Inspect log viewer.
evalkit-view *ARGS:
    @uv run inspect view --log-dir .evalkit/logs "$@"

# Project a committed run into a display payload, one way only, under this board's profile.
evalkit-export *ARGS:
    @mkdir -p .evalkit && uv run python -m evalkit.profile --out .evalkit/profile.yaml
    @uv run housecast grade export --profile .evalkit/profile.yaml "$@"

# Read an Inspect eval log and build the dataset the annotator grades.
evalkit-filter *ARGS:
    @uv run python -m evalkit.filter "$@"

# Ask whether response dispersion predicted the grade. `just evalkit-validity RUN/annotations.kai.yaml`.
evalkit-validity ANNOTATIONS:
    @uv run python evaluations/split-candidates-2026-09-08/validity.py "{{ANNOTATIONS}}"

# Cluster annotation critiques into a ranked failure taxonomy.
evalkit-taxonomy *ARGS:
    @uv run housecast grade taxonomy "$@"

# Annotate the eval dataset by hand, one keystroke per challenge.
evalkit-annotate *ARGS:
    @sh scripts/eval-annotate.sh "$@"

# Emit this board's profile as YAML, for a grading surface that takes --profile.
evalkit-profile *ARGS:
    @uv run python -m evalkit.profile "$@"

# Project the roster as entities.json, the --entities a grading surface takes.
evalkit-entities *ARGS:
    @sh scripts/eval-entities.sh "$@"

# Grade a JSONL of {half, response} on stdin with the autonomy grader.
evalkit-autonomy *ARGS:
    @uv run python -m evalkit.autonomy "$@"

# Reshape challenges.yaml into eval_cases.yaml, housecast's EvalCase wire schema.
evalkit-reshape *ARGS:
    @uv run python scripts/reshape_eval_cases.py "$@"

# Grade one committed run in a browser, with this roster's entities and profile. `just grade-serve evaluations/pilot/RUN`.
grade-serve *ARGS:
    @sh scripts/eval-serve.sh "$@"
