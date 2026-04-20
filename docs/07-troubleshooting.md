## Troubleshooting

Ten real issues that bit during the original build. Each one is a `Symptom → Root cause → Fix`.

## 0. Ollama in Docker on macOS is painfully slow

**Symptom.** You ran `docker-compose up` on a Mac, the bundled Ollama pulled `gemma3:4b` fine, but every chat response takes tens of seconds per token. A 300-token answer takes several minutes.

**Root cause.** Docker Desktop on macOS runs containers in a Linux VM. That VM **cannot access Metal** (Apple's GPU). So the `ollama` container is CPU-only, and CPU inference on a 4B-param model tops out around 5–10 tok/s.

**Fix.** Install Ollama directly on the Mac (`brew install ollama && brew services start ollama`), pull your models, and point the ragbot container at the host Ollama. Copy `docker-compose.override.yml.example` to `docker-compose.override.yml` — that's the one-liner override. The host Ollama gets full Metal acceleration (60+ tok/s on M-series Macs) and the rest of the stack stays in Docker.

This issue is macOS-specific. On Linux with nvidia-container-toolkit, the in-container Ollama can use the host GPU directly.

## 1. `could not access file "$libdir/vector"` on startup

**Symptom.** Every Postgres-using service crashes on startup with this error from `CREATE EXTENSION vector`.

**Root cause.** The Postgres image doesn't ship the pgvector shared library. Stock `postgres:16` does not include it. Stock `ghcr.io/cloudnative-pg/postgresql:17.4` does not include it either.

**Fix.** Use `pgvector/pgvector:pg16` (community image, works out of the box — this is what the starter's `docker-compose.yml` uses). Or build a custom image: `FROM postgres:16` + `apt-get install postgresql-16-pgvector`.

## 2. SSE returns `{"error":"streaming not supported"}`

**Symptom.** Every POST to `/api/query` returns a JSON error instead of streaming.

**Root cause.** The server's handler calls `w.(http.Flusher)` but a middleware wrapper embeds `http.ResponseWriter` without exposing `Flush()` — Go's interface assertion checks against the concrete type, not the embedded field.

**Fix.** Any middleware that wraps `ResponseWriter` must also expose `Flush()`:
```go
func (rw *myWrapper) Flush() {
    if f, ok := rw.ResponseWriter.(http.Flusher); ok {
        f.Flush()
    }
}
```
The starter avoids this by keeping the middleware chain trivial (one logging middleware that doesn't wrap the writer).

## 3. Stream cuts off after 10 seconds with `net/http: abort Handler`

**Symptom.** Tokens start flowing, then stop abruptly around 10s. The browser reports `network error`.

**Root cause.** `http.Server.WriteTimeout` is a *total* deadline, not an idle timeout. Any streaming response that takes longer gets killed mid-flight.

**Fix.** Set `WriteTimeout: 0` for any server that streams. Per-request context handles cancellation. The starter's `main.go` sets `WriteTimeout: 0`.

## 4. Reverse-proxying SSE drops every frame after the first

**Symptom.** When you put the bot behind a reverse proxy (Gateway service, Nginx, Traefik), only the first SSE frame reaches the browser.

**Root cause.** A manual Read/Write/Flush proxy loop can buffer at multiple layers. `http.Client` buffers response bodies; `ResponseWriter` wrappers may not flush.

**Fix.** Use Go's `httputil.ReverseProxy` with `FlushInterval: -1`. That's the canonical SSE-safe proxy:
```go
proxy := &httputil.ReverseProxy{
    Rewrite: func(pr *httputil.ProxyRequest) {
        pr.SetURL(upstreamURL)
    },
    FlushInterval: -1,
}
```

## 5. "Connection refused" to Ollama from a container

**Symptom.** Your containerised service can't reach Ollama at `http://host.docker.internal:11434` (Mac) or the host's IP (Linux). `ping` works, `curl` from the host works, but from inside a container fails.

**Root cause.** Ollama defaults to binding `127.0.0.1:11434`, which only accepts connections from localhost. Containers are on a different network interface.

**Fix.** Set `OLLAMA_HOST=0.0.0.0:11434` in the Ollama process environment. On macOS Homebrew, edit `~/Library/LaunchAgents/homebrew.mxcl.ollama.plist` to add the env var, then `launchctl unload ... && launchctl load ...`. The starter's `docker-compose.yml` runs Ollama inside the same compose network so this isn't an issue there.

## 6. `$\rightarrow$` instead of `→` in answers

**Symptom.** Answers contain LaTeX math-mode markers.

**Root cause.** Some models (Gemma especially) are trained on academic corpora and occasionally emit LaTeX tokens.

**Fix.** A small regex map in the widget:
```js
const LATEX_MAP = { rightarrow: '→', times: '×', cdot: '·', ... };
s.replace(/\$\\([a-zA-Z]+)\$/g, (m, n) => LATEX_MAP[n] || m);
```
The starter's `widget.js` already does this.

## 7. Inline `[source-name]` duplicates the source chips

**Symptom.** Every answer has source chips *and* literal `[00-getting-started]` markers in the text.

**Root cause.** The system prompt originally asked for inline citations. The UI already shows them as chips below — double display.

**Fix.** The starter's prompt tells the model NOT to inline-cite. The widget also strips `[\d+-[a-z0-9-]+]` patterns as a safety net.

## 8. Bot refuses to answer questions that ARE in the docs

**Symptom.** You ask a question clearly covered by a doc, and the bot says "I don't have that in the docs yet."

**Root cause.** Retrieval missed it. Usually because either (a) the question is phrased very differently from the doc ("how do I reset my password?" vs the doc titled "Password recovery"), or (b) the doc is a single wall of text without H2 headings so the chunker produced one giant unhelpful chunk.

**Fix.** Add H2 headings to the doc. Write the question as one of the headings. Re-index.

## 9. Model is too slow (answers take 2+ minutes)

**Symptom.** Ollama takes forever to generate each answer.

**Root cause.** The chat model is too large for your hardware. A dense 27B parameter model on a consumer laptop runs at ~10 tok/s → a 300-token answer takes 30s.

**Fix.** **Benchmark, don't assume.** Try `gemma3:4b`, `llama3.2:3b`, `qwen2.5:7b`. On a Mac M4 we measured `gemma4:26b` at 62 tok/s (fastest), `qwen3.5:35b-a3b` at 41 (MoE, disappointingly slow), `qwen3.5:27b` at 13 (dense, too slow). Smaller models with well-written docs often beat bigger ones with mediocre docs.

## 10. The bot is answering questions, but the answers are vague

**Symptom.** The bot cites the right doc, but its answers read like ChatGPT — general and unspecific.

**Root cause.** Your docs are general and unspecific.

**Fix.** Rewrite your docs. Name exact screens, exact fields, exact commands. "Open Settings → Domain → Enter `links.yourbrand.com`" beats "configure your domain in the settings." RAG amplifies doc quality; it doesn't fix vagueness.
