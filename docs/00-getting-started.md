# Getting Started

**go-rag-starter** is a self-hosted RAG (Retrieval-Augmented Generation) help bot template. It answers questions about your product by reading the markdown docs you give it, not by guessing. Everything runs on your own hardware — no OpenAI key required.

## What you get when you run it

- A Go service listening on port `8080`.
- A built-in demo HTML page at `http://localhost:8080`.
- A chat widget (the bubble in the bottom-right) that streams answers token-by-token.
- Answers include "source chips" showing which doc each answer came from.
- On first start, the service indexes every markdown file under `docs/` into a pgvector database.

## Quickstart

```bash
git clone https://github.com/joyson-fernandes/go-rag-starter.git
cd go-rag-starter
docker-compose up
```

Wait ~2 minutes on first start — Ollama has to pull two models (`gemma3:4b` for chat, `nomic-embed-text` for embeddings, ~3.3 GB total). Subsequent starts are instant.

Open `http://localhost:8080`. Click the bubble. Ask "how do I swap the LLM?" and you should see a streamed answer citing `03-swap-the-llm.md`.

## Point at your own host Ollama (recommended on macOS)

Docker Desktop on macOS runs containers in a Linux VM with no access to Metal (Apple's GPU), so the bundled `ollama` service runs on CPU only — roughly 5-10× slower than a host-installed Ollama. Fix: install Ollama on the Mac and point the bot at it.

```bash
# 1. Install Ollama on the host
brew install ollama && brew services start ollama

# 2. Pull the models you want
ollama pull gemma4:26b          # or any chat model from docs/03-swap-the-llm.md
ollama pull nomic-embed-text

# 3. Activate the override — ragbot will talk to the host Ollama
cp docker-compose.override.yml.example docker-compose.override.yml

# 4. Boot the stack
docker-compose up
```

The override sets:

- `OLLAMA_URL=http://host.docker.internal:11434` — points ragbot at your Mac's Ollama.
- `OLLAMA_CHAT_MODEL=gemma4:26b` — default chat model (edit the override file to pick a different one).
- `OLLAMA_EMBED_MODEL=nomic-embed-text` — embedding model.

The bundled container Ollama is skipped (the override removes the dependency on it).

**On Linux with an NVIDIA GPU** you don't need the override — install `nvidia-container-toolkit`, add `deploy.resources.reservations.devices` to the `ollama` service in `docker-compose.yml`, and the bundled container gets GPU access directly.

**On Linux without a GPU** the bundled Ollama works fine for small models (`gemma3:4b`, `llama3.2:3b`) — CPU is faster on Linux than inside Docker-on-Mac because there's no VM overhead.

## What just happened

1. Docker Compose started three containers: `postgres` (with pgvector), `ollama`, and `ragbot` (the Go service).
2. The service connected to Postgres, created its schema, and enabled the `vector` extension.
3. It hashed the `docs/` directory and compared it to the database. First run → no match → embeds every chunk.
4. Each markdown file was split into ~400-token chunks at H2 headings.
5. Each chunk was sent to Ollama's `/api/embeddings` endpoint, producing a 768-dimensional vector.
6. Vectors + text were stored in the `ragbot_chunks` table with an HNSW index.

## Next steps

- **Replace the corpus** — drop your own markdown into `docs/` and restart. See `01-replace-the-corpus.md`.
- **Customise the prompt** — change how the bot answers. See `02-customise-the-prompt.md`.
- **Swap the LLM** — use OpenAI / Anthropic / Bedrock instead of local Ollama. See `03-swap-the-llm.md`.
- **Embed on your site** — add the widget to any webpage with one `<script>` tag. See `04-embed-the-widget.md`.
- **Add authentication** — gate access to the bot. See `05-add-authentication.md`.
- **Deploy to production** — Kubernetes / Helm notes. See `06-deploy-to-kubernetes.md`.

## Troubleshooting

If something breaks on first run, `07-troubleshooting.md` covers the common issues: Postgres missing pgvector, Ollama not reachable, SSE streaming not working.
