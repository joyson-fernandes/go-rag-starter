package ragbot

import (
	"fmt"
	"strings"
)

// Config holds runtime tunables read from environment variables.
// ProductName and ProductBlurb flow into the system prompt so the same
// binary can power different bots.
type Config struct {
	ProductName  string // e.g. "Acme"
	ProductBlurb string // e.g. "an invoicing SaaS for freelancers"
	RefusePhrase string // shown when the answer isn't in the corpus
}

// SystemPrompt returns the prompt string built from the config. Edit this
// function to customise how the bot behaves across every query. See
// docs/02-customise-the-prompt.md for the anatomy.
func SystemPrompt(cfg Config) string {
	name := cfg.ProductName
	if name == "" {
		name = "this product"
	}
	blurb := cfg.ProductBlurb
	if blurb == "" {
		blurb = "a software product"
	}
	refuse := cfg.RefusePhrase
	if refuse == "" {
		refuse = "I don't have that in the docs yet — try asking something else, or check the project README."
	}
	return fmt.Sprintf(`You are the help assistant for %s, %s.

Answer the user's question using ONLY the documentation chunks provided below. Follow these rules:

1. Be concise. Default under 150 words. Step-by-step walkthroughs may be longer when the user asks for one.
2. Do NOT include inline citations like [source-name] in your answer — the source chips are shown separately in the UI.
3. If the answer is not in the chunks, say exactly: "%s"
4. Never invent feature names, prices, URLs, or settings paths.
5. When a step involves a specific screen or file, name it exactly.
6. Markdown is allowed: use short bullet lists and **bold** for emphasis.`, name, blurb, refuse)
}

// BuildMessages composes the message list sent to the chat model.
func BuildMessages(cfg Config, history []ChatMessage, retrieved []Retrieved, userQuery string) []ChatMessage {
	var context strings.Builder
	for i, r := range retrieved {
		src := strings.TrimSuffix(strings.TrimPrefix(r.SourcePath, "docs/"), ".md")
		fmt.Fprintf(&context, "\n--- chunk %d [%s]", i+1, src)
		if r.Heading != "" {
			fmt.Fprintf(&context, " · %s", r.Heading)
		}
		context.WriteString(" ---\n")
		context.WriteString(strings.TrimSpace(r.Content))
		context.WriteString("\n")
	}

	msgs := []ChatMessage{{Role: "system", Content: SystemPrompt(cfg)}}
	msgs = append(msgs, history...)
	msgs = append(msgs, ChatMessage{
		Role:    "user",
		Content: fmt.Sprintf("Documentation chunks:\n%s\n\nQuestion: %s", context.String(), userQuery),
	})
	return msgs
}

// UniqueSources deduplicates source paths, preserving first-appearance order.
func UniqueSources(retrieved []Retrieved) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, r := range retrieved {
		if _, ok := seen[r.SourcePath]; ok {
			continue
		}
		seen[r.SourcePath] = struct{}{}
		out = append(out, r.SourcePath)
	}
	return out
}
