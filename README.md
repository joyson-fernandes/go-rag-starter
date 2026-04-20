# go-rag-starter

A minimal, self-hosted, **self-documenting** RAG help bot. Clone it, run `docker-compose up`, and you have a widget that answers questions about your product — grounded in your own markdown docs, with no API keys and no vendor bills.

MIT licensed.

---

## 5-minute quickstart

```bash
git clone https://github.com/joyson-fernandes/go-rag-starter.git
cd go-rag-starter
docker-compose up -d          # -d = detached; come back to your prompt
docker-compose logs -f ragbot # watch startup + indexing (Ctrl+C to stop watching)
```

First start takes ~2 minutes — Ollama pulls two models (~3.3 GB). Subsequent starts are instant.

Open **http://localhost:8080** in a browser. Click the purple bubble. Ask:

> *How do I swap Ollama for OpenAI?*

You'll see tokens stream in with a source chip pointing at `03-swap-the-llm.md`. The bot is answering using *its own docs* — that's the self-documenting part.

When you're done: `docker-compose down` (stops + removes containers; volumes kept for the next run).

### Faster setup (matters on macOS and any no-GPU host)

The bundled container Ollama is CPU-only unless the host gives it GPU access. Two overrides ship with the repo to cover every case:

| Platform | Command | Why |
|---|---|---|
| macOS (any M-series) | `cp docker-compose.override.yml.example docker-compose.override.yml` + install host Ollama | Docker can't reach Metal on Mac |
| Windows (no GPU) | same as macOS, download Ollama from ollama.com | Same pain without the GPU |
| Windows (NVIDIA) | `cp docker-compose.gpu.yml.example docker-compose.override.yml` | GPU passthrough on the bundled container |
| Linux (NVIDIA) | `cp docker-compose.gpu.yml.example docker-compose.override.yml` | Same GPU passthrough |
| Linux (no GPU) | just `docker-compose up -d` | CPU is OK without VM overhead |

Full per-platform walkthrough in [`docs/00-getting-started.md`](docs/00-getting-started.md).

---

## What's in the box

| Piece | Notes |
|---|---|
| Go binary | ~500 lines. Stateless. Binds `:8080`. |
| pgvector Postgres | Stores chunks, vectors, conversations. Via Docker Compose. |
| Ollama | Local LLM runtime. `gemma3:4b` for chat, `nomic-embed-text` for embeddings. |
| Widget | One vanilla-JS file served at `/widget.js`. Embed anywhere with `<script src>`. |
| Demo page | `GET /` renders a minimal page with the widget already loaded. |
| 9 seed docs | In `docs/`. Both human-readable AND the bot's corpus. Replace with your own. |

No Kubernetes. No Vault. No Helm. No React toolchain. One `docker-compose.yml`, one `.env`, done.

---

## Replace the seed corpus

1. Drop your own markdown files into `docs/`. Use H2 (`## `) headings — the chunker splits on them.
2. `docker-compose build ragbot && docker-compose up -d ragbot`.
3. On startup the service notices the corpus hash changed and re-embeds everything. ~40 seconds for ~60 chunks.

Either add alongside the starter's own docs (so the bot answers both "how does go-rag-starter work" and "how does *my product* work"), or wipe `docs/` entirely and add only yours.

See `docs/01-replace-the-corpus.md` for the format + writing tips.

---

## Embed the widget on any site

```html
<script src="http://localhost:8080/widget.js"
        data-api="http://localhost:8080"
        data-title="Ask Acme"
        data-subtitle="Answers from the Acme docs"></script>
```

Framework-agnostic. No build step. Works on Next.js / Nuxt / plain HTML / Wordpress / anything.

See `docs/04-embed-the-widget.md`.

---

## Customise

| Want to… | See |
|---|---|
| Change how the bot writes | `docs/02-customise-the-prompt.md` |
| Use OpenAI / Anthropic / Bedrock instead of Ollama | `docs/03-swap-the-llm.md` |
| Gate access with auth | `docs/05-add-authentication.md` |
| Run on Kubernetes | `docs/06-deploy-to-kubernetes.md` |
| Debug something weird | `docs/07-troubleshooting.md` |
| Understand how it works | `docs/08-architecture.md` |

**Or just run the bot and ask it.** The 9 docs above are the bot's own corpus.

---

## Architecture in one diagram

```
┌──── browser ──────────────────────────────────────────────┐
│  fetch() + ReadableStream  ──POST→  /api/query → SSE      │
└───────────────────────┬───────────────────────────────────┘
            ┌───────────▼───────────┐
            │  ragbot  :8080        │  single Go process
            │  1. embed user query  │──→ Ollama
            │  2. hybrid retrieve   │──→ Postgres + pgvector
            │  3. stream LLM tokens │──→ Ollama
            └───────────────────────┘
```

- **Retrieval** is hybrid: HNSW vector similarity + `tsvector` full-text search, fused with Reciprocal Rank Fusion in a single SQL CTE.
- **Generation** streams from Ollama via SSE; the widget renders tokens as they arrive.
- **Ingest** is idempotent + content-hash-gated: restarts are instant unless the corpus changed.

Details in `docs/08-architecture.md`.

---

## Why this exists

Proprietary support bots (Intercom Fin, Zendesk AI) are expensive and your data has to leave your network. Managed RAG services (kapa.ai, Inkeep) start at hundreds of dollars a month. Meanwhile, a small Postgres + any local LLM + a few hundred lines of Go does the same thing well enough for most support-widget use cases — and costs nothing per query.

This template is the minimum viable version of that. Half the effort in the original build was spent on 10 specific gotchas (SSE middleware, HTTP/2 buffering, `http.Server.WriteTimeout` killing streams, pgvector missing from the default Postgres image, etc.). Those gotchas are captured up front in `docs/07-troubleshooting.md` so you don't have to hit them.

---

## React widget?

This starter ships a vanilla-JS widget only. It embeds on any page with one `<script>` tag, no build step. For a React component version, adapt the vanilla JS in `web/widget.js` — the protocol (POST to `/api/query`, read SSE frames, render tokens as they arrive) is the same regardless of framework.

---

## License

MIT. See `LICENSE`. Credit appreciated but not required.
