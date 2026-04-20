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
docker-compose up -d                  # -d = detached; returns to prompt
docker-compose logs -f ragbot         # watch startup (Ctrl+C to stop watching)
```

Wait ~2 minutes on first start — Ollama has to pull two models (`gemma3:4b` for chat, `nomic-embed-text` for embeddings, ~3.3 GB total). Subsequent starts are instant.

Open `http://localhost:8080`. Click the bubble. Ask "how do I swap the LLM?" and you should see a streamed answer citing `03-swap-the-llm.md`.

**Other useful commands:**

- `docker-compose ps` — what's running.
- `docker-compose logs -f ragbot` — stream one service's logs.
- `docker-compose restart ragbot` — re-run ragbot after editing `docs/`.
- `docker-compose down` — stop everything (volumes kept for next run).
- `docker-compose down -v` — stop and wipe everything, including the vector index.

## Platform-specific setup

The bundled container Ollama is CPU-only unless the host gives it GPU access. Pick the path for your setup.

### macOS (Intel or Apple Silicon)

Docker Desktop on Mac can't reach Metal (Apple's GPU), so the bundled Ollama is painfully slow. Install Ollama on the Mac and point ragbot at it:

```bash
brew install ollama && brew services start ollama
ollama pull gemma4:26b          # or any chat model from docs/03-swap-the-llm.md
ollama pull nomic-embed-text
cp docker-compose.override.yml.example docker-compose.override.yml
docker-compose up -d
```

Mac Ollama gets full Metal acceleration (~60 tok/s on M4 Max for a 26B model).

### Windows with NVIDIA GPU

Docker Desktop + WSL2 + NVIDIA drivers expose the GPU to containers. Enable GPU passthrough on the bundled Ollama:

```bash
cp docker-compose.gpu.yml.example docker-compose.override.yml
docker-compose up -d
```

The override adds `deploy.resources.reservations.devices: [{driver: nvidia, count: all, capabilities: [gpu]}]` to the `ollama` service. No host install needed.

### Windows without a GPU

Same approach as macOS — install Ollama on the host, point ragbot at it:

1. Download the Ollama installer from [ollama.com/download](https://ollama.com/download).
2. `ollama pull gemma3:4b` and `ollama pull nomic-embed-text` in PowerShell.
3. `copy docker-compose.override.yml.example docker-compose.override.yml`.
4. `docker-compose up -d`.

`host.docker.internal` resolves correctly on Windows Docker Desktop.

### Linux with NVIDIA GPU

Install `nvidia-container-toolkit` if you haven't already, then:

```bash
cp docker-compose.gpu.yml.example docker-compose.override.yml
docker-compose up -d
```

Same override as Windows+GPU — works on any Linux where `docker run --gpus all` works.

### Linux without a GPU

The bundled Ollama works out of the box — Linux has no VM overhead, so CPU inference is tolerable for smaller models (`gemma3:4b`, `llama3.2:3b`). Just:

```bash
docker-compose up -d
```

Alternatively, host-install Ollama (`curl -fsSL https://ollama.com/install.sh | sh`) and use the host-Ollama override — avoids the 3.3 GB container and the double model copy.

## Override env vars

Whichever path you pick, the key settings are:

- `OLLAMA_URL` — where ragbot reaches Ollama. `http://host.docker.internal:11434` for host-installed, `http://ollama:11434` for the bundled container.
- `OLLAMA_CHAT_MODEL` — any model name that `ollama list` shows.
- `OLLAMA_EMBED_MODEL` — the embedder; `nomic-embed-text` is the standard default.

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
