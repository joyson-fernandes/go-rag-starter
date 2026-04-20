<div align="center">

# go-rag-starter

**A self-hosted, self-documenting RAG help bot. Clone it, run one command, get a working chat widget grounded in your own markdown docs.**

[![License](https://img.shields.io/github/license/joyson-fernandes/go-rag-starter?color=blue)](LICENSE)
[![Go version](https://img.shields.io/github/go-mod/go-version/joyson-fernandes/go-rag-starter)](go.mod)
[![Last commit](https://img.shields.io/github/last-commit/joyson-fernandes/go-rag-starter)](https://github.com/joyson-fernandes/go-rag-starter/commits/main)
[![Code size](https://img.shields.io/github/languages/code-size/joyson-fernandes/go-rag-starter)](https://github.com/joyson-fernandes/go-rag-starter)
[![Stars](https://img.shields.io/github/stars/joyson-fernandes/go-rag-starter?style=flat&logo=github)](https://github.com/joyson-fernandes/go-rag-starter/stargazers)

No API keys. No vendor bills. No Kubernetes. ~500 lines of Go, one `docker-compose.yml`, done in 5 minutes.

</div>

---

## What you get

```
┌──── browser ──────────────────────────────────────────────┐
│  fetch() + ReadableStream                                  │
│  POST /api/query  →  reads SSE frames                     │
└───────────────────────┬────────────────────────────────────┘
            ┌───────────▼───────────┐
            │  ragbot  :8080        │  single Go process
            │  1. embed user query  │──→ Ollama
            │  2. hybrid retrieve   │──→ Postgres + pgvector
            │  3. stream LLM tokens │──→ Ollama
            └───────────────────────┘
```

A floating chat bubble appears on a built-in demo page. You click it, type a question, and tokens stream in — with a "source chip" pointing at the markdown file each claim came from. The bot is indexed from its own docs, so you can ask "how do I swap Ollama for OpenAI?" and get a real answer the moment it starts.

Replace `docs/*.md` with your own product docs, rebuild, and you have a support widget for your own project.

---

## Quick start

```bash
git clone https://github.com/joyson-fernandes/go-rag-starter.git
cd go-rag-starter
docker-compose up -d
```

First run pulls ~3.6 GB (Ollama models + container images). Subsequent starts are instant. Then open **http://localhost:8080**.

> **On macOS?** The bundled Ollama runs CPU-only (Docker on Mac can't access Metal) and will feel painfully slow. Install Ollama on the host with `brew install ollama && brew services start ollama`, then `cp docker-compose.override.yml.example docker-compose.override.yml` before `docker-compose up -d`. Full per-platform guide in [docs/00-getting-started.md](docs/00-getting-started.md).

---

## Features

- **Self-hosted** — runs on your hardware, data never leaves your network.
- **Self-documenting** — the `docs/` directory is both human onboarding AND the bot's corpus. Replace it to make the bot about your product.
- **Hybrid retrieval** — pgvector (HNSW) + full-text search (BM25-ish), fused with Reciprocal Rank Fusion in a single SQL CTE.
- **Streaming UI** — Server-Sent Events; tokens appear as the model generates them.
- **Framework-agnostic widget** — one `<script src>` tag, no build step, works on plain HTML, Next.js, Nuxt, WordPress, anywhere.
- **Swappable LLM** — ships with Ollama; adapter shape for OpenAI / Anthropic / Bedrock documented.
- **Production-hardened** — the 10 real gotchas from the original build are captured up front so you don't hit them.
- **Single binary** — no gateway, no microservices, no Kubernetes. Go 1.25 + Postgres + Ollama.

---

## Platform setup

| Platform | Command | Why |
|---|---|---|
| macOS (any M-series) | Install host Ollama + `cp docker-compose.override.yml.example docker-compose.override.yml` | Docker can't reach Metal |
| Windows (NVIDIA) | `cp docker-compose.gpu.yml.example docker-compose.override.yml` | GPU passthrough on bundled Ollama |
| Windows (no GPU) | Install host Ollama + override (same as macOS) | Same pain without the GPU |
| Linux (NVIDIA) | `cp docker-compose.gpu.yml.example docker-compose.override.yml` | GPU passthrough on bundled Ollama |
| Linux (no GPU) | `docker-compose up -d` | Bundled Ollama is fine without VM overhead |

Full walkthrough: [docs/00-getting-started.md](docs/00-getting-started.md).

---

## Customise

All nine docs below are indexed into the bot's corpus — open them in the chat widget or read them on GitHub.

| Task | Doc |
|---|---|
| Replace the seed corpus with your own docs | [01-replace-the-corpus.md](docs/01-replace-the-corpus.md) |
| Tune the system prompt / bot persona | [02-customise-the-prompt.md](docs/02-customise-the-prompt.md) |
| Swap Ollama for OpenAI / Anthropic / Bedrock | [03-swap-the-llm.md](docs/03-swap-the-llm.md) |
| Embed the widget on your own site | [04-embed-the-widget.md](docs/04-embed-the-widget.md) |
| Add authentication (JWT, API key, X-User-ID) | [05-add-authentication.md](docs/05-add-authentication.md) |
| Deploy to Kubernetes | [06-deploy-to-kubernetes.md](docs/06-deploy-to-kubernetes.md) |
| Debug a common issue | [07-troubleshooting.md](docs/07-troubleshooting.md) |
| Understand the architecture + glossary | [08-architecture.md](docs/08-architecture.md) |

---

## Embed on any site

Three attributes on one `<script>` tag:

```html
<script src="https://your-bot-host/widget.js"
        data-api="https://your-bot-host"
        data-title="Ask Acme"
        data-subtitle="Answers from the Acme docs"></script>
```

Framework-agnostic. No build step. Matches Intercom / Stripe Dashboard / GitHub Copilot UX: floating bubble bottom-right, non-modal side panel, streaming tokens with source chips, keyboard-navigable.

---

## Project layout

```
go-rag-starter/
├── main.go                             # service entrypoint (~140 lines)
├── internal/ragbot/                    # the RAG service
│   ├── chunker.go                      # markdown H2 splitter
│   ├── ingest.go                       # hash-gated corpus indexer
│   ├── ollama.go                       # LLM client (Embed + ChatStream)
│   ├── store.go                        # pgvector hybrid-search SQL
│   ├── service.go                      # embed → retrieve → generate pipeline
│   ├── handler.go                      # /api/query (SSE), /widget.js, /
│   ├── prompt.go                       # system prompt template
│   └── schema.go                       # DDL (pgvector + tables + indexes)
├── web/
│   ├── widget.js                       # vanilla-JS chat widget (~380 lines)
│   └── index.html                      # built-in demo page served at /
├── docs/                               # seed corpus (human-readable + bot-queryable)
│   └── 00..08-*.md
├── docker-compose.yml                  # postgres + ollama + ragbot
├── docker-compose.override.yml.example # opt-in: point at host Ollama (macOS)
├── docker-compose.gpu.yml.example      # opt-in: NVIDIA GPU passthrough
└── Dockerfile                          # single-stage Go build, distroless runtime
```

Total: ~1,200 lines of Go, ~380 lines of widget JS, nine seed docs.

---

## FAQ

<details>
<summary><b>Why Go?</b></summary>

Fast to build, fast to run, trivial to ship as a single static binary. The whole service is one `docker build` away from a ~15 MB distroless image. If you prefer TypeScript/Python/Rust, the pattern (`embed query → hybrid search → stream LLM`) is the same — this starter just makes Go the path of least resistance.

</details>

<details>
<summary><b>Why not use [LangChain / LlamaIndex / another framework]?</b></summary>

Frameworks add abstractions that usually obscure what's happening. RAG is three pipe-separated steps — chunking, retrieval, generation. You can read the whole service in 30 minutes and know exactly what it does. When something breaks (and it will), you debug actual code, not framework internals.

</details>

<details>
<summary><b>Does it scale?</b></summary>

pgvector + Postgres handles millions of chunks comfortably. The bottleneck is usually the LLM, not the retrieval. For many concurrent users, front the service with something that does connection pooling and horizontal-scale multiple ragbot replicas behind a load balancer — it's stateless.

</details>

<details>
<summary><b>Can I use this commercially?</b></summary>

Yes, MIT licensed. Keep the copyright notice. No warranty.

</details>

<details>
<summary><b>React component version?</b></summary>

Not shipped in v1 — the vanilla-JS widget works everywhere including React apps via `<script>` in your layout. If you want a true React component, adapt `web/widget.js`; the protocol (POST to `/api/query`, read SSE frames, render tokens as they arrive) is framework-independent.

</details>

---

## Contributing

Issues and pull requests welcome. A few things that would help:

- **Cleaner prompt templates** for specific domains (docs sites, e-commerce support, internal wikis).
- **Adapters for other LLMs** — OpenAI / Anthropic / Bedrock drop-ins in `internal/ragbot/ollama.go` style.
- **Eval harness** — golden Q&A set runner with LLM-judge scoring.
- **React / Vue widget packages** on npm with the same protocol.

Keep PRs small and focused; a 10-line fix merges faster than a 500-line refactor.

---

## License

[MIT](LICENSE) © 2026 Joyson Fernandes.

<div align="center">

If this saves you a weekend, consider [starring the repo](https://github.com/joyson-fernandes/go-rag-starter) — it's the only signal GitHub's algorithm listens to.

</div>
