## Architecture

A one-screen overview of what's running and how the pieces connect.

## Diagram

```
┌──── browser ─────────────────────────────────────────────┐
│  fetch() + ReadableStream                                 │
│  POST /api/query  ────────→  reads SSE frames            │
└───────────────────────┬───────────────────────────────────┘
                        │
            ┌───────────▼───────────┐
            │  ragbot  :8080        │  Go, one process
            │                       │
            │  1. embed user query  │──→ Ollama /api/embeddings
            │  2. hybrid retrieve   │──→ Postgres + pgvector
            │  3. stream from LLM   │──→ Ollama /api/chat
            └───────────────────────┘
```

Three components. Two dependencies (Postgres and Ollama). Everything runs in the same `docker-compose.yml`.

## The RAG pattern in five steps

1. **Chunk** — split markdown into ~400-token pieces at H2 boundaries. `internal/ragbot/chunker.go`.
2. **Embed** — send each chunk to Ollama's embedding model, get back a 768-dim vector. `internal/ragbot/ollama.go` `Embed()`.
3. **Store** — rows of `(source_path, content, embedding)` in pgvector's `vector` column, with an HNSW index for fast nearest-neighbour search. `internal/ragbot/schema.go`.
4. **Retrieve** at query time — embed the user's question, run a hybrid SQL query that combines vector similarity + BM25 keyword search, return top 6 chunks. `internal/ragbot/store.go` `HybridSearch()`.
5. **Generate** — feed the chunks + question + system prompt into the chat LLM. Stream tokens back to the browser over SSE. `internal/ragbot/service.go`.

## Glossary

- **Embedding / vector** — numbers representing the meaning of text. Nearby vectors mean similar meaning. `nomic-embed-text` produces 768 numbers per chunk.
- **pgvector** — PostgreSQL extension that adds a `vector` column type and fast similarity search operators (`<=>` for cosine distance).
- **HNSW** — Hierarchical Navigable Small World. A graph-based vector index. Slow to build, very fast to query. Default for pgvector >= 0.5.
- **BM25** — classical keyword ranking algorithm. Postgres' `tsvector` + `ts_rank` is its cousin. Good at exact word matches (where vector search is weak).
- **Hybrid search** — run both vector and BM25, combine the rankings. Almost always beats either alone.
- **RRF** — Reciprocal Rank Fusion, the formula we use to combine the two ranked lists: `score = 1/(60 + rank_vec) + 1/(60 + rank_bm25)`. The constant 60 is from the original paper; no tuning needed.
- **Chunk** — one retrievable unit of text. For markdown, "one H2 section" is a good default.
- **Token** — roughly 3/4 of a word. LLMs measure context limits in tokens.
- **Context window** — how many tokens the model can read at once. Gemma 3 4B is 8k, GPT-4o is 128k.
- **SSE (Server-Sent Events)** — one-way HTTP streaming. Server sends `event: <name>\ndata: <json>\n\n` frames.
- **Ollama** — a local LLM runtime. Serves HTTP on port 11434. Supports both embedding and chat models.

## The hybrid SQL in full

```sql
WITH vec AS (
    SELECT id, source_path, heading, content,
           row_number() OVER (ORDER BY embedding <=> $1::vector) AS r
    FROM ragbot_chunks
    ORDER BY embedding <=> $1::vector
    LIMIT 20
),
fts AS (
    SELECT id, source_path, heading, content,
           row_number() OVER (ORDER BY ts_rank(to_tsvector('english', content),
                                               plainto_tsquery('english', $2)) DESC) AS r
    FROM ragbot_chunks
    WHERE to_tsvector('english', content) @@ plainto_tsquery('english', $2)
    LIMIT 20
),
fused AS (
    SELECT COALESCE(v.id, f.id) AS id,
           COALESCE(v.source_path, f.source_path) AS source_path,
           COALESCE(v.heading, f.heading) AS heading,
           COALESCE(v.content, f.content) AS content,
           COALESCE(1.0/(60 + v.r), 0) + COALESCE(1.0/(60 + f.r), 0) AS score
    FROM vec v FULL OUTER JOIN fts f ON v.id = f.id
)
SELECT source_path, heading, content, score FROM fused
ORDER BY score DESC LIMIT 6;
```

## What's NOT here (and when to add it)

- **Cross-encoder reranker** — a second scoring pass on the top 20 results for more precision. Adds 100-200ms. Worth it if users complain about irrelevant answers.
- **Multi-turn history** — the starter sends the full message history each turn, which hits the 8k context limit fast. For long chats, summarise old turns.
- **Eval harness** — a golden Q&A set + LLM judge to catch answer regressions. Essential before trusting the bot on high-stakes content.
- **Feedback loop → doc gaps** — the starter stores 👎 in the DB. Periodically query the 👎 messages and rewrite the docs that triggered them.
- **Observability** — tracing, metrics, log aggregation. See `06-deploy-to-kubernetes.md`.

All of these are a weekend of work each, not months. The starter is the 80% solution — the remaining 20% is adapting to your specific context.

## Credits

This starter is an extraction of the help-bot built for **Linkvolt**, a link-in-bio platform. The production implementation is open-source at [github.com/joyson-fernandes/linkvolt](https://github.com/joyson-fernandes/linkvolt) (see `internal/aichat/` and `web/src/components/ai-chat/`).

A long-form blog post on the full build is at [dev.to/joyson-fernandes](https://dev.to/joyson-fernandes) — includes the 10 gotchas from this file expanded, plus the benchmarks, system-prompt anatomy, and lessons learned.
