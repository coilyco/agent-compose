# Graphiti MCP server - supply-chain audit

Read-only audit, as of 2026-10-02T07:52Z (`date -u`). Source pinned at `getzep/graphiti@3c427640` (`gh api repos/getzep/graphiti/commits/main`). Nothing was installed or run. All source reads are `curl https://raw.githubusercontent.com/getzep/graphiti/<ref>/<path>`. Verdict and fully-local answer come from Jev (`jev-1.13.0`, verdict yellow p=0.84 / red p=0.16 / green p=0, fully-local "no" p=1.0).

## Verdict: yellow

The maintainers and code are trustworthy (MEASURED). Running it fully local behind Agent Proxy does not work today (MEASURED): the server has no embedder that runs in-process, and Agent Proxy has no `/v1/embeddings`.

## Fully-local feasibility: NO (blocking: embeddings)

* **Embedder (blocker)** - `EmbedderFactory` in `mcp_server/src/services/factories.py` L313-410 accepts only `openai`, `azure_openai`, `gemini`, `voyage`. `openai` calls `client.embeddings.create` (`graphiti_core/embedder/openai.py`) against `embedder.providers.openai.api_url`, which means `POST {base}/embeddings`. The `graphiti_core/embedder/` tree holds only those four providers (`gh api .../git/trees/<sha>?recursive=1`). There is no sentence-transformers or other in-process embedder. Ingest and search both need embeddings. Ways to close the gap:
  1. Add a `/v1/embeddings` route to Agent Proxy that resolves to Ollama (for example `nomic-embed-text` or `bge-m3`). This is Platform Engineer work, and the clean option.
  2. Point the embedder `api_url` straight at Ollama's `/v1/embeddings`. This bypasses Agent Proxy, and AGENTS.md does not allow that outside named exceptions.
  3. Patch in a local sentence-transformers embedder. This is a fork, and upstream issue #437 ("Add Hugging Face Embedder Support") has been open since 2025-05.
* **LLM: OK, with a caveat** - When `OPENAI_API_URL` is not api.openai.com, `is_non_openai_provider()` (L94-108) selects `OpenAIGenericClient`. That client uses Chat Completions (`client.chat.completions.create`, `openai_generic_client.py` L155), not the Responses API. Only the official-OpenAI branch uses `OpenAIClient` with the Responses API (L172-178). The client sends `response_format` = `json_schema` by default, built from the raw pydantic schema with no `strict` (L102-124). Setting `LLM_STRUCTURED_OUTPUT_MODE=json_object` switches it to `json_object` and injects the schema into the prompt (L194-200). It strips code fences and retries 4 times. Open issues #1909 and #1913 report that Ollama/llama.cpp grammars skip optional keys under `json_schema`, so edges lose `valid_at`/`invalid_at`. EXPECTED: extraction will run on `ornith:35b` / `ministral-3:14b`, with some temporal fields missing. Agent Proxy must pass `response_format` through unchanged. I have not verified that it does.
* **Reranker: wired but unused by the MCP tools** - `CrossEncoderFactory` (L413-511) picks the LLM provider's reranker first. With `provider: openai` that is `OpenAIRerankerClient(base_url=<proxy>)`. Its model is unset, so it sends `gpt-4.1-nano` (`openai_reranker_client.py` L31, L87), with `logprobs`, `top_logprobs=2`, and OpenAI-tokenizer `logit_bias` ids. That would fail against an Ollama route. The local `BGERerankerClient` (sentence-transformers `BAAI/bge-reranker-v2-m3`, downloaded from Hugging Face on first run) is only a fallback, and the config cannot select it while the LLM provider is `openai`. The MCP tools only use `NODE_HYBRID_SEARCH_RRF`, `NODE_HYBRID_SEARCH_NODE_DISTANCE`, `EDGE_HYBRID_SEARCH_RRF`, and `EDGE_HYBRID_SEARCH_NODE_DISTANCE` (`graphiti_mcp_server.py` L583-591, `graphiti.py` L1629). Their rerankers are `rrf` and `node_distance` (`search_config_recipes.py`). So the cross-encoder is constructed but never called by MCP search (MEASURED by grep). It does not call OpenAI, because the base_url is the proxy.
* **Leak risk: silent fallback to OpenAI** - `graphiti_mcp_server.py` L247-256 catches LLM and embedder factory exceptions and only logs a warning, then passes `None`. `graphiti.py` L216-227 replaces `None` with the default `OpenAIClient()`, `OpenAIEmbedder()`, and `OpenAIRerankerClient()`, which target api.openai.com unless `OPENAI_BASE_URL` is set. Mitigations: set `OPENAI_BASE_URL=<agent-proxy>/v1` in the container env, and enforce egress deny at the Docker network level.
* **Image 1.0.2 is unsafe** - `factories.py` at tag `mcp-v1.0.2` passes `base_url` only on the embedder (L274). The LLM client ignores `api_url` and calls api.openai.com (issue #1744). Tag `mcp-v1.1.0` fixes this, and its `factories.py` is byte-identical to HEAD (`diff` printed IDENTICAL). Use 1.1.0 or later only.

## Telemetry

* PostHog, enabled by default. `graphiti_core/telemetry/telemetry.py` has `TELEMETRY_ENV_VAR = 'GRAPHITI_TELEMETRY_ENABLED'` with default `'true'` and host `https://us.i.posthog.com`. A single `graphiti_initialized` event fires from `Graphiti.__init__` (`graphiti.py` L247-266). The payload is the LLM/embedder/reranker/driver class names, version, CPU architecture, and an anonymous UUID cached in `~/.cache/graphiti`. It carries no content. `posthog` is a hard dependency (`pyproject.toml` L20).
* No compose file or Dockerfile sets the variable. grep of `mcp_server/` found only `SFW_TELEMETRY_DISABLED`, which belongs to the build-time Socket Firewall.
* **Disable:** `GRAPHITI_TELEMETRY_ENABLED=false`.
* The FalkorDB Browser UI listens on :3000 with `BROWSER=1` by default. Set `BROWSER=0` unless it is needed. I did not audit its telemetry.

## Images and pins (Docker Hub API, `hub.docker.com/v2/repositories/<repo>/tags`)

* **MCP server, separate-DB topology (recommended)** - `zepai/knowledge-graph-mcp:1.1.0-standalone` (alias `1.1.0-graphiti-0.30.1-standalone`), pushed 2026-09-01. Index digest `sha256:52d619bc3c45527dd5e6c5c413f3024f50d79fa672c81f4b384eb04bff85ca84`, arm64 `sha256:ac14b7cf527b67a9abddf99381f728afd8d8588c7e679316c8e961c5a32082b6`. Upstream compose (`docker-compose-falkordb.yml`, `-neo4j.yml`) uses the floating tag `:standalone`.
* **MCP server, combined FalkorDB+MCP** - `zepai/knowledge-graph-mcp:1.1.0` (upstream `docker-compose.yml` uses `:latest`, the same digest), index `sha256:a2536b6d59b4afb359a13aeaa4a9d2f4db195af14231168efda6a3d166228956`, arm64 `sha256:4e0716083edeaafebc363549a40589f382280ea065d8556568ae5ad0639bcb44`. It is built `FROM falkordb/falkordb:latest`. The 2026-09-01 build predates FalkorDB 6.0, so it is the 4.x engine. A local rebuild today would pull 6.x.
* **FalkorDB** - upstream compose uses `falkordb/falkordb:latest`, which now resolves to **6.0.1** (Rust rewrite, pushed 2026-10-01). Open issue #1947 (2026-09-29) reports Graphiti is broken on FalkorDB 6.x (fulltext index procedure). **Pin `falkordb/falkordb:v4.22.0`** (C engine, 2026-09-30), arm64 `sha256:f36b2d00c8b88c0b3a76e56d534638540de549ce8b2a6f66cfa815b6e29d70e4`. License is SSPLv1 (`LICENSE` header). That is fine for local self-hosting and blocks offering it as a service.
* **Neo4j alternative** - `neo4j:5.26.0` (pinned in `docker-compose-neo4j.yml`).
* **Library** - `graphiti-core` 0.30.2 on PyPI (uploaded 2026-09-08, `pypi.org/pypi/graphiti-core/json`). MCP server release `mcp-v1.1.0` (2026-09-01). HEAD carries unreleased MCP SDK 2.x (#1921, 2026-09-25) and group_id routing (#1926).
* **Config env** - `OPENAI_API_URL`, `OPENAI_API_KEY` (any non-empty value, `_validate_api_key`), `MODEL_NAME`, `LLM_STRUCTURED_OUTPUT_MODE`, `EMBEDDER_MODEL`. `embedder.dimensions` is hard-coded at 1536 in `config.yaml`, so mount a custom config to match the local embedding model. LLM and embedder share `OPENAI_API_URL` unless the YAML splits them.

## Org / maintainers

* Org: `getzep`, verified, created 2023-05-07, blog getzep.com, 14 public repos (`gh api orgs/getzep`).
* Top contributors: danielchalef 348 commits, prasmussen15 293, paul-paliychuk 64 (`gh api repos/getzep/graphiti/contributors`). These match the Zep authors in `pyproject.toml` (`@getzep.com` emails).
* Famous-named credentials: n/a.

## Repo health

* Created 2024-08-08, 31,372 stars, 3,218 forks, not archived, last push 2026-09-30 (`gh api repos/getzep/graphiti`).
* License: Apache-2.0.
* Present: CI (unit, lint, typecheck, mcp-server-tests, CodeQL), `dependabot.yml`, `SECURITY.md`, CLA workflow, and Socket Firewall in image builds (`.github/workflows/*`).
* Build scripts: the Dockerfile downloads `sfw` from `SocketDev/firewall-release/releases/latest` with no checksum. That only matters when building locally, and a pull-only setup skips it. The project uses `pyproject.toml`, has no `setup.py`, and declares no suspicious `[project.scripts]`.

## Supply chain

* Advisory: GHSA-gg5m-55jj-8m5g, high, Cypher injection via `node_labels`, affects `<= 0.28.1`. The current 0.30.2 is outside that range (`api.github.com/advisories?ecosystem=pip&affects=graphiti-core`, `gh api repos/getzep/graphiti/security-advisories`). It was disclosed with a fixed release, which is a positive sign.
* Adoption: 178,819 pulls of `zepai/knowledge-graph-mcp`. 563 issues, with active external reporters.
* Hijack check: the last 30 commits are Zep staff (pevans, prasmussen15, jackaldenryan), the CLA bot, and contributor fixes landed through PRs. No anomalous emails touched manifests.

## Yellow flags worth naming honestly

* No local embedder, so Agent Proxy needs a `/v1/embeddings` route or a policy exception.
* Silent `None` falls back to api.openai.com when a factory throws.
* `falkordb:latest` points at 6.x, which breaks Graphiti (#1947).
* `json_schema` on Ollama drops optional fields (#1909).
* Image 1.0.2 sends LLM traffic to OpenAI regardless of config (#1744).
* The reranker would send `gpt-4.1-nano` to the proxy if a cross-encoder recipe is ever used.

## Recommendation

Allow with caveats, once the embeddings route exists:

1. Add `/v1/embeddings` to Agent Proxy, resolving to an Ollama embedding model. File this for the Platform Engineer.
2. Pin `zepai/knowledge-graph-mcp:1.1.0-standalone@sha256:52d619bc…` and `falkordb/falkordb:v4.22.0`, with `BROWSER=0`.
3. Set the env `GRAPHITI_TELEMETRY_ENABLED=false`, `OPENAI_BASE_URL=<proxy>/v1`, `OPENAI_API_URL=<proxy>/v1`, and a dummy `OPENAI_API_KEY`. Try `LLM_STRUCTURED_OUTPUT_MODE=json_schema` first and fall back to `json_object` if extraction fails.
4. Mount a config with `embedder.dimensions` matching the local model.
5. Run the container on a Docker network whose only egress is Agent Proxy, so the OpenAI fallback cannot leave the host.
6. Re-audit on `mcp-v1.2.0` (MCP SDK 2.x) before upgrading.
