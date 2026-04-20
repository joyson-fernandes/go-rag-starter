## Swap the LLM

The starter uses Ollama by default because it's free, runs locally, and keeps your data on your own hardware. You can swap it for OpenAI, Anthropic, Bedrock, or any OpenAI-compatible API with small changes.

## What needs to change

You need to replace two methods in `internal/ragbot/ollama.go`:

- `Embed(ctx, text)` — returns a `[]float32` vector for a chunk or query.
- `ChatStream(ctx, msgs)` — returns a channel that yields tokens as they generate.

Everything else (retrieval SQL, handler, widget) stays the same.

## OpenAI

Replace `Embed` to call `https://api.openai.com/v1/embeddings` with `text-embedding-3-small` (1536 dimensions — **also update `vector(768)` → `vector(1536)` in `schema.go`** and re-index).

```go
func (o *Ollama) Embed(ctx context.Context, text string) ([]float32, error) {
    body, _ := json.Marshal(map[string]any{
        "model": "text-embedding-3-small",
        "input": text,
    })
    req, _ := http.NewRequestWithContext(ctx, "POST",
        "https://api.openai.com/v1/embeddings", bytes.NewReader(body))
    req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
    req.Header.Set("Content-Type", "application/json")
    resp, err := o.HTTP.Do(req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    var out struct {
        Data []struct { Embedding []float32 `json:"embedding"` } `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&out)
    return out.Data[0].Embedding, nil
}
```

Replace `ChatStream` to call `https://api.openai.com/v1/chat/completions` with `stream=true`. Parse the SSE frames (they're shaped slightly differently from Ollama's).

## Anthropic

Use the Claude Messages API at `https://api.anthropic.com/v1/messages`. Anthropic doesn't offer an embedding model — you'd still use Ollama's `nomic-embed-text` (or another provider) for `Embed`, and Claude for `ChatStream`.

## AWS Bedrock

Bedrock exposes Titan for embeddings (`amazon.titan-embed-text-v2`) and Claude/Llama/Mistral for chat. Use the AWS SDK for Go v2's `bedrockruntime` package. `InvokeModelWithResponseStream` for the chat side.

## Local alternatives to Ollama

- **llama.cpp server** — binary-compatible with Ollama's API on most endpoints.
- **vLLM** — OpenAI-compatible server for running models on GPU. Much faster than Ollama for concurrent requests.
- **LM Studio** — desktop app with an HTTP server; good for development.

## Model choice

For a support bot, you want:

- **Cheap** — you'll send every chunk through the embedder once, and the chat model fires on every question.
- **Fast tokens/sec** — users hate waiting.
- **Good at "read this and summarise"** — which is exactly what RAG is.

Some defaults that work well:

- Local (free, ~40-60 tok/s on a modern Mac): Gemma 3 4B, Qwen 2.5 7B, Llama 3.2 3B.
- OpenAI API: `gpt-4o-mini` is usually the sweet spot.
- Anthropic: `claude-haiku-4-5` or the newest Haiku equivalent — fastest + cheapest.

## Cost estimate

A reasonable help-bot usage pattern: 1,000 queries/month, average answer ~300 tokens, average retrieved context ~2,000 tokens.

- Ollama locally: **£0/month**, electricity only.
- OpenAI `gpt-4o-mini`: ~$0.50/month at that volume.
- Anthropic `claude-haiku`: ~$1/month.

Scale up 100× (100k queries) and you're still under $100/month on the managed APIs. RAG is cheap at the query layer; the expensive part is usually the embedding of your corpus, which is one-time.
