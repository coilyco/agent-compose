# Supply-chain audit // basic-memory (basicmachines-co/basic-memory)

Audited 2026-10-02 07:52 UTC (`date -u`). Read-only. Nothing installed or executed. Source read from the v0.23.2 release tarball (`curl -sL https://github.com/basicmachines-co/basic-memory/archive/refs/tags/v0.23.2.tar.gz`, extracted and grepped only). Raw API responses saved beside this file under `raw/`. `$R` = `basicmachines-co/basic-memory`, `$A` = `https://api.github.com`, `$SRC` = the extracted tarball.

## Verdict: yellow

A real, active, well-adopted project with clean advisory history. It does NOT run fully local by default: the stdio MCP server checks PyPI and can self-upgrade, and the default semantic search downloads an embedding model from HuggingFace. All of it can be switched off with config plus env vars, listed below.

## Network calls (the fully-local question)

* **Auto-update, ON by default, fires from the MCP server** - MEASURED. `auto_update: bool = Field(default=True ...)` (`grep -n auto_update $SRC/src/basic_memory/config_models.py`, line 696). `cli/commands/mcp.py` starts `threading.Thread(target=_run_background_auto_update, daemon=True)` whenever `transport == "stdio"` (`sed -n '85,99p' cli/commands/mcp.py`). `cli/auto_update.py` fetches `https://pypi.org/pypi/basic-memory/json` (line 22) and, for a `uv tool` or Homebrew install, runs `uv tool upgrade basic-memory` or `brew upgrade basic-memory` silently (lines 401-412). Interval 86400 s (line 702). Skipped entirely for `uvx` (line 280). For an unknown install source (plain venv/pip) it checks PyPI but does not install (lines 365-377).
  * Disable: `{"auto_update": false}` in `~/.basic-memory/config.json` (README lines 596-606), or env `BASIC_MEMORY_AUTO_UPDATE=false` (EXPECTED from `env_prefix="BASIC_MEMORY_"`, config_models.py line 886, pydantic-settings convention, not run).
  * `uv tool upgrade` respects the constraint given at install, so a `==0.23.2` pin makes the upgrade a no-op (uv docs, `curl -s https://raw.githubusercontent.com/astral-sh/uv/main/docs/concepts/tools.md | grep -n respect`, line 134). The daily PyPI GET still happens unless `auto_update` is false.
* **FastEmbed model download, ON by default** - MEASURED in code, download itself EXPECTED (not run). `semantic_search_enabled` defaults to true whenever `fastembed` and `sqlite_vec` import (`_default_semantic_search_enabled`, config_models.py lines 64-69), and both are hard dependencies. Provider defaults to `fastembed`, model `bge-small-en-v1.5` aliased to `BAAI/bge-small-en-v1.5` (config_models.py 292-297, fastembed_provider.py 40-42). FastEmbed maps that model to HF repo `Qdrant/bge-small-en-v1.5-onnx-Q` (`curl -s https://raw.githubusercontent.com/qdrant/fastembed/main/fastembed/text/onnx_embedding.py`, line 77). Cache dir: `FASTEMBED_CACHE_PATH` or `~/.basic-memory/fastembed_cache` (config_models.py 91-99).
  * Disable: `"semantic_search_enabled": false` (keyword/FTS search only), or pre-seed the cache once and then set `HF_HUB_OFFLINE=1` (EXPECTED, huggingface_hub convention, not run).
  * Reranker is off by default (`reranker_enabled` default False, config_models.py 430).
* **Umami analytics, NOT on the MCP path** - MEASURED. `cli/analytics.py` POSTs to `https://api-gateway.umami.dev/api/send` with a baked-in site id. Callers (`grep -rn 'track(' src/basic_memory`): only `cli/promo.py` (promo shown) and `cli/commands/cloud/core_commands.py` (cloud login, promo opt-out). Promo returns early for subcommand `mcp` and for non-interactive sessions (promo.py ~line 100-104). Disable: `BASIC_MEMORY_NO_PROMOS=1` (analytics.py lines 41-44, README "Telemetry").
* **Logfire, OFF by default** - MEASURED. `logfire_enabled` and `logfire_send_to_logfire` both `default=False` (config_models.py 213-218), and `configure_telemetry` returns before `logfire.configure` when disabled (telemetry.py).
* **Cloud sync, opt-in** - MEASURED. Cloud routing needs `cloud_api_key` or an OAuth login (`has_cloud_credentials`, config.py line 240). Stdio honors `BASIC_MEMORY_FORCE_CLOUD` if set externally (mcp.py comments). Pin local with `BASIC_MEMORY_FORCE_LOCAL=true` (the same var the HTTP transports set, mcp.py ~line 58).
* **Other outbound sites** - `grep -rnE 'urlopen|httpx\.|requests\.' src/basic_memory`: only `cli/auth.py` (cloud login), `cli/auto_update.py`, `cli/analytics.py`, and `ci/project_updates.py` (only via the `bm ci` GitHub Actions command). litellm and openai are imported lazily inside their providers only (`repository/litellm_provider.py:97`, `openai_provider.py:52`), so the default fastembed path does not import litellm (inference from import sites, not traced at runtime).

## Org / maintainers

* Org: real. `gh api orgs/basicmachines-co` -> Organization "Basic Machines", created 2021-08-08, blog `https://www.basicmachines.co/`, email `hello@basicmachines.co`, 21 public repos.
* Top contributor: `phernandez` (Paul Hernandez, company "Basic Memory"), account created 2009-03-07, 47 followers, 1319 contributions (`gh api repos/$R/contributors`, `gh api users/phernandez`). Second: `groksrc` (Drew Cain), created 2014-10-09, 128 contributions.
* Famous-named credentials verified: n/a, none claimed.

## Repo health

* Created 2024-12-02, last push 2026-10-02 02:15 UTC, 4076 stars, 299 forks, not archived (`gh api repos/$R`).
* Latest stable release v0.23.2, published 2026-08-25 20:44 UTC (`gh api repos/$R/releases`). PyPI `info.version` = 0.23.2, wheel uploaded 2026-08-25T20:44:42, sha256 `a1679a16319d8a7fb9c0486033551a47dedc0fbae7f5da81444eb3c4bf0ccecb` (`curl -s https://pypi.org/pypi/basic-memory/json`). Dev builds up to 0.23.3.dev8 exist. A v0.24.0 changelog was cut on main 2026-09-29 but is not released.
* License: AGPL-3.0 (GitHub `license.spdx_id`, PyPI `AGPL-3.0-or-later`, LICENSE header).
* CI/hygiene: `.github/dependabot.yml`, 11 workflows incl. `test.yml`, `release.yml`, `mcp-registry-publish.yml` (`gh api repos/$R/contents/.github/workflows`); latest main runs Tests / SQLite lifecycle probes success at 2026-10-02 02:15 (`curl -s "$A/repos/$R/actions/runs?per_page=10"`). `SECURITY.md`, `CHANGELOG.md`, `uv.lock` present (`gh api repos/$R/contents`).
* Build scripts: none. hatchling backend, no `setup.py`, `[project.scripts]` only `basic-memory` and `bm` -> `basic_memory.cli.main:app` (`sed -n '78,86p' $SRC/pyproject.toml`).
* Self-modifying behaviour: the auto-updater shells out to `uv`/`brew` at runtime (see above). Not malicious, but it is code that changes the installed version underneath a pin-unaware install.

## Supply chain

* Advisory DB: clean. `curl -s "$A/advisories?ecosystem=pip&affects=basic-memory"` -> 0. `gh api repos/$R/security-advisories` -> 0.
* MCP registry: listed as `io.github.basicmachines-co/basic-memory`, versions 0.17.7 through 0.23.2 (`curl -s "https://registry.modelcontextprotocol.io/v0/servers?search=basic-memory"`). `server.json` declares pypi `basic-memory` 0.23.2, transport stdio.
* External engagement: 407 issues and 170 merged PRs from authors other than phernandez (`$A/search/issues?q=repo:$R+is:issue+-author:phernandez`, `...+is:pr+is:merged+-author:phernandez`).
* Hijack check: last 60 commits (`gh api "repos/$R/commits?per_page=60"`) come from phernandez noreply, `paul@basicmachines.co`, `joe@basicmemory.com`, plus external contributors (`subp@wachtel.us`, a `163.com` address, `georgeatparallel`, `masteragentsiri@gmail.com`) whose changes are docs/fixes landed by the maintainer. No manifest or build-script touch from a new identity seen.
* Dependency tree red flags: 43 non-extra runtime requirements (`jq -r '.info.requires_dist[]' raw/pypi.json | grep -vc extra`).
  * `fastmcp==4.0.0b1`, a beta, hard-pinned (requires_dist, uv.lock line 1105).
  * `litellm>=1.60.0,<1.92.0`. That range spans 1.82.7-1.82.8, flagged as credential-harvesting malware (GHSA-5mg7-485q-xm76, `curl -s "$A/advisories?ecosystem=pip&affects=litellm&per_page=100"`), and several critical proxy-server advisories below 1.84.0. Lockfile pins 1.91.3 (uv.lock line 1969), but `uv tool install` resolves fresh and ignores uv.lock. A fresh resolve picks the newest allowed release (EXPECTED).
  * Test/dev tools shipped as runtime deps: `pyright`, `pytest-aio`, `pytest-asyncio`.
  * Server drivers pulled for every install: `asyncpg`, `psycopg==3.3.1`, `fastapi[standard]`, `openai`.

## Yellow flags worth naming honestly

* Stdio MCP server self-updates by default when installed via `uv tool` or Homebrew. This is the main finding against the fully-local requirement.
* Default semantic search downloads model weights from HuggingFace on first indexing.
* AGPL-3.0. Fine for running it as a local tool. It contaminates only if its code is linked into or redistributed with a non-AGPL project.
* Heavy transitive tree (litellm, openai, Postgres drivers, pyright) for a local markdown store, plus a beta `fastmcp` pin.
* PyPI author email domain `basic-machines.co` (hyphenated) differs from the org's `basicmachines.co` (`jq .info.author_email raw/pypi.json`). Likely a typo. Not verified.
* Umami analytics ships a baked-in site id and is opt-out. It does not fire on the MCP path.

## Recommendation

Allow with caveats. Mitigations:

1. Pinned install: `uv tool install basic-memory==0.23.2`
2. `~/.basic-memory/config.json`: `{"auto_update": false}`. Add `"semantic_search_enabled": false` for zero model download, or keep it on, let the first index pull the ~bge-small ONNX weights once, then run with `HF_HUB_OFFLINE=1`.
3. Env on every harness's MCP entry: `BASIC_MEMORY_NO_PROMOS=1`, `BASIC_MEMORY_FORCE_LOCAL=true`, `BASIC_MEMORY_AUTO_UPDATE=false` (env form EXPECTED, config-file form is documented).
4. Stdio launch, the same for Claude Code, Codex, and OpenCode: command `basic-memory`, args `["mcp"]` (add `--project <name>` to constrain). The README form is `uvx basic-memory mcp`. `uvx basic-memory@0.23.2 mcp` also skips the auto-updater, but it resolves dependencies at runtime, so the pinned `uv tool` binary is the stable choice.
5. Before trusting "no network", verify with one egress-blocked run (for example Little Snitch, or `HF_HUB_OFFLINE=1` plus a firewall deny) and record the result. That run has not been done.
