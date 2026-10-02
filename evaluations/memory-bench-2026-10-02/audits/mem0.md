# Supply-chain audit // Mem0 / OpenMemory MCP

Read-only audit, 2026-10-02 UTC (`date -u` -> Fri Oct 2 07:52:03 UTC 2026). mem0ai/mem0 main at abb81c88e1 (`curl api.github.com/repos/mem0ai/mem0/commits/main`). Nothing installed or executed. Verdict and feasibility picks came from Jev (jev-1.13.0).

## Verdict: yellow

Jev: verdict yellow (p=0.71, red 0.29, green 0.00, confidence 0.56). The mem0ai library is a real, maintained, provenance-signed package. The OpenMemory MCP server Kai asked about is archived and unmaintained, and its successor (`server/`) ships no MCP surface and hardcodes cloud providers. A fully-local MCP store is feasible only with own build work (Jev: yes_with_own_build p=0.93, confidence 0.90).

## Where OpenMemory went (MEASURED)

* Removed from the monorepo - commit ea2ee075, 2026-07-29, "chore: remove OpenMemory from the monorepo (#6530)" (`curl "api.github.com/repos/mem0ai/mem0/commits?path=openmemory"`). This is why the prior openmemory path fetch 404'd.
* Snapshot lives at github.com/mem0ai/openmemory under `openmemory-archive/` - commit b0bfce4b message: "Snapshot of the OpenMemory self-hosted MCP server + dashboard ... Read-only, no longer maintained." (`curl api.github.com/repos/mem0ai/openmemory/commits/b0bfce4b`)
* The archive README carries a CAUTION banner "This project has been archived" and points to the Mem0 self-hosted server (`curl raw.githubusercontent.com/mem0ai/openmemory/main/openmemory-archive/README.md`).
* The `mem0ai/openmemory` repo name now belongs to a different product: "Open-source CLI & TUI to port AI coding sessions across Claude Code, Codex, and OpenCode", MIT, created 2026-07-06 (`curl api.github.com/repos/mem0ai/openmemory`).
* Docker image `mem0/openmemory-mcp:latest` last pushed 2025-06-17, digest sha256:b665d382a94fdc18c7bb84a647d4bebdc98d9e7fc1146fc5e559ca7f5f7f9211, the only tag (`curl hub.docker.com/v2/repositories/mem0/openmemory-mcp/tags`). It predates fixes made in the monorepo through 2026-06-18.
* `mem0ai/mem0-mcp` repo is archived=true (`curl api.github.com/orgs/mem0ai/repos`).

## Feasibility answers

### 1. Local in-process embedders (MEASURED from source at main)

* `fastembed` - mem0/embeddings/fastembed.py, ONNX runtime via `fastembed.TextEmbedding`, default model `thenlper/gte-large`, dims taken from the model. No network beyond the first weight download.
* `huggingface` - mem0/embeddings/huggingface.py. With no `huggingface_base_url` it loads `SentenceTransformer(model)` in-process, default `multi-qa-MiniLM-L6-cos-v1`. Setting `huggingface_base_url` switches it to an OpenAI-style `/embeddings` HTTP client, so leave that unset.
* Both are optional extras: pyproject.toml `extras` lists `sentence-transformers>=5.2.0` and `fastembed>=0.3.1`. Base deps are httpx, openai, posthog, protobuf, pydantic, pytz, qdrant-client, sqlalchemy (`curl pypi.org/pypi/mem0ai/json | jq .info.requires_dist`).
* `ollama` and `lmstudio` embedders also exist, but they need an embeddings HTTP endpoint, and Agent Proxy has none.
* Caveat - no `embedding_dims` propagation into the vector-store config was found in mem0/memory/main.py, mem0/utils/factory.py, or mem0/configs/base.py (grep, 0 hits). The Qdrant config default is `embedding_model_dims: 1536` (mem0/configs/vector_stores/qdrant.py line 12). Set `vector_store.config.embedding_model_dims` by hand to the embedder's size (384 for MiniLM-L6, 1024 for gte-large). EXPECTED: a mismatch fails at insert.

### 2. Custom OpenAI-compatible LLM plus local embedder

* Library: yes. mem0/llms/openai.py line 51: `base_url = self.config.openai_base_url or os.getenv("OPENAI_BASE_URL") or "https://api.openai.com/v1"`. The call sends `response_format` and, for some paths, `tools` (lines 136-139). EXPECTED: Agent Proxy must pass these through to the Ollama route.
* Egress footgun 1 - an empty base_url falls back to api.openai.com. Footgun 2 - lines 42-46: if `OPENROUTER_API_KEY` is present in the environment, the client goes to OpenRouter whatever the config says. Keep both out of the container env and set `OPENAI_BASE_URL` explicitly as a second guard.
* Successor `server/` (REST, not MCP): no as shipped. server/main.py line 62-63: `BUNDLED_LLM_PROVIDERS = ("openai", "anthropic", "gemini")`, `BUNDLED_EMBEDDER_PROVIDERS = ("openai", "gemini")`. `POST /configure` calls `_validate_bundled_providers` and returns 400 for any other provider (lines 232-258). DEFAULT_CONFIG wires the openai embedder to api key + model only. A local embedder requires editing that tuple, adding sentence-transformers or fastembed to requirements.txt, and rebuilding. The LLM can reach Agent Proxy through `OPENAI_BASE_URL` (library reads it) with no code change. Routes: /configure, /memories, /search, /reset (grep `@app.` in server/main.py). No MCP endpoint.
* Archived OpenMemory MCP: partially. openmemory-archive/api/app/utils/memory.py builds the LLM config from env, `openai` provider accepts `base_url` -> `openai_base_url` (lines 145-151). Unknown embedder providers fall through to a generic `{"model": ...}` config (lines 209-223), so `EMBEDDER_PROVIDER=huggingface` is wired. But api/requirements.txt carries neither sentence-transformers nor fastembed, so the image must be rebuilt. It is unmaintained.
* Official MCP surfaces are hosted-only: docs/platform/mem0-mcp.mdx uses `https://mcp.mem0.ai/mcp` and requires a Platform API key. integrations/agent-plugin-core/python/memory_core.py line 31 `DEFAULT_API_URL = "https://api.mem0.ai"`, overridable by `MEM0_API_URL`, but it calls `/v3/memories/add/` and `/v3/memories/search/` (lines 1996, 2335), which server/main.py does not serve. That mismatch is my inference from comparing the two route sets. Nobody has run it against the self-hosted server.

### 3. Telemetry (MEASURED from source)

* Library mem0/memory/telemetry.py: PostHog, host `https://us.i.posthog.com`, `MEM0_TELEMETRY` default "True" (line 14), read at import time. Events carry class names, collection name, vector dims, and md5 of user_id/agent_id/run_id (mem0/memory/utils.py line 235+). No memory content was seen in the payloads.
* Server server/telemetry.py: default on, sends `admin_registered` and `onboarding_completed` with email domain and a freeform use-case string.
* Agent plugins integrations/agent-plugin-core/python/telemetry.py: PostHog batch, same opt-out.
* Off switch for all three: `MEM0_TELEMETRY=false`. It must be set in the container env before the Python process starts. Optional belt: block egress to us.i.posthog.com.

### 4. Vector store and images (MEASURED, Docker Hub API)

* The archived OpenMemory path defaults to Qdrant (`mem0_store`, port 6333) with an unpinned `qdrant/qdrant` image (openmemory-archive/docker-compose.yml). Pin `qdrant/qdrant:v1.19.1`, digest sha256:12364fe851b9f17356fc88189fc06d1b521262e04659ec7345975b00c9246a10, pushed 2026-09-03 (`curl hub.docker.com/v2/repositories/qdrant/qdrant/tags/v1.19.1`). It matches GitHub release v1.19.1 published 2026-09-04.
* The successor `server/` uses pgvector: `pgvector/pgvector:pg17` (server/docker-compose.yaml). Current pinned equivalent `pgvector/pgvector:0.8.7-pg17-trixie`, digest sha256:7a7e9f22015b67edb4bef5c59daeebcd7e74bfa570df6ce60ae01237c8648a84, pushed 2026-10-01.
* The library alone can run Qdrant embedded on disk (`path`, `on_disk: true` in QdrantConfig), so a thin MCP wrapper needs no vector container at all. EXPECTED, not run.
* `mem0/mem0-api-server:latest` was last pushed 2025-09-10, 342497 pulls. It is stale against main and has no MCP surface.
* PyPI pin: `mem0ai==2.2.1`, uploaded 2026-09-25. Wheel sha256 fe91bb91ac8926231993a4aa58df00a60c6c74c709e6a338fe399500776eef4d, sdist sha256 099a7d58368908d0a05a633c5da3920f607a57db00d4f168800de05966b2a4a8.

## Org / maintainers

* Org - mem0ai, Organization, created 2023-06-19, blog https://mem0.ai, 12 public repos, is_verified false (`curl api.github.com/orgs/mem0ai`).
* Top contributors - Dev-Khant 453, kartik-mem0 227, deshraj 219, taranjeet 206. deshraj created 2012-12-02, 1262 followers, company @mem0ai. taranjeet created 2013-04-30, 931 followers, company @mem0ai.
* Famous-named credentials verified - n/a.

## Repo health

* Created 2023-06-20, pushed 2026-10-01, 66455 stars, not archived, Apache-2.0.
* CI - 34 workflow files incl. ci.yml, cd.yml, release.yml, pr-gate.yml. SECURITY.md and .pre-commit-config.yaml present. No dependabot.yml or renovate config in .github (`curl .../contents/.github`), though commits cite Dependabot/Vanta alerts.
* Release provenance - PyPI attestation for the 2.2.1 wheel names publisher GitHub mem0ai/mem0 cd.yml (`curl pypi.org/integrity/mem0ai/2.2.1/.../provenance`), and cd.yml uses `id-token: write` + `pypa/gh-action-pypi-publish`.
* Build scripts - hatchling build, no setup.py hooks seen in pyproject.toml.

## Supply chain

* Advisories (GHSA REST + OSV): GHSA-jfv9-68m5-gjjr high, GHSA-cgx8-qgvr-f7vf medium, GHSA-gq6f-qwv9-rf4j medium, all "mem0 server lacks authentication", range `<= 1.0.0`, no patched version recorded. GHSA-xqxw-r767-67m7 low, patched 2.0.0b2. Mirrored as PYSEC-2026-2633..2636.
* Recent 40 commits by email domain: 16 noreply.github.com, 14 gmail.com, 9 mem0.ai, 1 qq.com. External contributions arrive through a PR gate plus a vouch system (vouch-check-pr.yml, VOUCHED.td). No hijack pattern seen in the first page.
* External engagement - 105 issues opened since 2026-09-01 (search API).

## Yellow flags worth naming honestly

* The OpenMemory MCP server is archived (2026-07-23/29) and the only image is 15 months old.
* `server/` has no MCP surface and rejects non-cloud embedders without a code change and rebuild.
* Telemetry is on by default in library, server, and plugins.
* The LLM client silently falls back to api.openai.com, and honors `OPENROUTER_API_KEY` over config. Either one breaks the no-third-party constraint on a misconfiguration.
* The server auth advisories have no recorded patched version. Bind to 127.0.0.1 only and keep `AUTH_DISABLED=false`.
* `embedding_model_dims` defaults to 1536 and must be set by hand.

## Recommendation

Allow with caveats: adopt the `mem0ai==2.2.1` library, not the archived OpenMemory image or the stock server.

1. Build a thin local MCP wrapper (or fork openmemory-archive) on `mem0ai==2.2.1` + `fastembed` or `sentence-transformers`. This is Platform Engineer scope, not mine.
2. Config: `llm.provider=openai`, `openai_base_url=<Agent Proxy /v1>`, `embedder.provider=huggingface` (no base_url) or `fastembed`, Qdrant `qdrant/qdrant:v1.19.1` or embedded path, with `embedding_model_dims` matched to the embedder.
3. Container env: `MEM0_TELEMETRY=false`, `OPENAI_BASE_URL` set, `OPENROUTER_API_KEY` and a real `OPENAI_API_KEY` absent.
4. Verify with an egress-denied network (only Agent Proxy reachable after the weight download) before trusting it. EXPECTED, not run here.
5. Recheck on any mem0ai minor bump, since the server providers and telemetry defaults move.
