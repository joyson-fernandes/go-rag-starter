## Customise the prompt

The system prompt controls how the bot answers. The default is designed to be safe and useful out of the box, but you'll want to customise it for your product.

## Edit the prompt

The prompt lives in `internal/ragbot/prompt.go` in the `SystemPrompt` function. A simplified version:

```go
return fmt.Sprintf(`You are the help assistant for %s, %s.

Answer the user's question using ONLY the documentation chunks provided below. Follow these rules:

1. Be concise. Default under 150 words.
2. Do NOT include inline citations like [source-name] in your answer.
3. If the answer is not in the chunks, say exactly: "%s"
4. Never invent feature names, prices, URLs, or settings paths.
5. When a step involves a specific screen or file, name it exactly.
6. Markdown is allowed: use short bullet lists and **bold** for emphasis.`, name, blurb, refuse)
```

## Tune via environment variables

Three strings flow into the prompt from env vars:

- `PRODUCT_NAME` — e.g. `Acme`.
- `PRODUCT_BLURB` — e.g. `an invoicing SaaS for freelancers`.
- `REFUSE_PHRASE` — the exact sentence the bot says when the answer isn't in the corpus.

These live in `.env` (for bare-metal) or `docker-compose.yml` (for compose). No rebuild required — just restart the container.

## Tune the rules

Each rule in the prompt exists for a specific reason:

- **Rule 1 (concise)** — LLMs love to write essays. Without this they'll return 500-word answers to "how do I reset my password."
- **Rule 2 (no inline citations)** — the widget shows source chips below the message, so inline `[source-name]` references are duplicate and noisy.
- **Rule 3 (refuse phrase)** — prevents hallucination when the corpus is silent. Use a phrase that fits your brand.
- **Rule 4 (no invention)** — the reliability anchor. If the LLM doesn't know, it should say so.
- **Rule 5 (name screens)** — makes answers actionable. "The Domains page" beats "the settings."
- **Rule 6 (markdown)** — without this, some models output `**bold**` literally instead of bolded text.

## Common customisations

### Restrict to a specific audience

```
You are the help assistant for Acme. The reader is a freelancer using
Acme to send invoices to clients. Assume they don't have a technical
background. Avoid jargon. Prefer "invoice" over "document" and
"payment" over "transaction."
```

### Enforce a persona or tone

```
You are Clara, the support assistant for Acme. Clara is friendly,
confident, and brief. Use contractions ("you're", "it's"). Never
apologise preemptively ("I'm sorry, but...").
```

### Web-only or mobile-only context

If your docs cover both web and mobile, pin one in the prompt:

```
The user is currently using the Acme WEB editor. Default to web-editor
steps. Do NOT include mobile app instructions unless the user explicitly
asks about mobile.
```

Then build a second binary / container for the mobile widget with the opposite pin.

## Test prompt changes

The fastest way to iterate:

1. Edit `prompt.go`, rebuild, restart.
2. Ask the bot a question.
3. Read the answer; adjust the prompt; repeat.

There's no "prompt test harness" in the starter — we deliberately kept this simple. If you get serious about prompt evaluation, write a golden Q&A set and score answers with an LLM judge (Claude Haiku or similar). That's a whole discipline beyond this template.
