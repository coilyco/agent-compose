#!/bin/sh
# Grade the filtered dataset by hand. Projects the roster into entities first,
# so the header carries purpose, owned and scoped attributes, and adjacency.
set -e
dataset=${EVAL_DATASET:-.evalkit/dataset.yaml}
out=${EVAL_ANNOTATIONS:-.evalkit/annotations.yaml}

if [ ! -f "$dataset" ]; then
  echo "no dataset at $dataset. Run evalkit-filter first." >&2
  exit 2
fi

render_dir=$(mktemp -d)
cleanup() { rm -rf "$render_dir"; }
trap cleanup EXIT HUP INT TERM
agent-compose catalog snapshot --out "$render_dir/person.json"
uv run python -m evalkit.roster \
  --person "$render_dir/person.json" \
  --out "$render_dir/entities.json"

# Handed over as YAML because the grading half never imports evalkit. Without
# it the three-type default raises KeyError on the first voice case.
uv run python -m evalkit.profile --out "$render_dir/profile.yaml"

uv run housecast grade annotate \
  --dataset "$dataset" \
  --out "$out" \
  --profile "$render_dir/profile.yaml" \
  --entities "/entities.json" \
  "$@"
