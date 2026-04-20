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
