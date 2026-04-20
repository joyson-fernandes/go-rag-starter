package ragbot

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Ollama is a minimal client for the Ollama HTTP API.
//
// To use a different LLM backend (OpenAI / Anthropic / Bedrock), replace
// this file. See docs/03-swap-the-llm.md for the shape to match.
type Ollama struct {
	BaseURL    string
	ChatModel  string
	EmbedModel string
	HTTP       *http.Client
}

func NewOllama(baseURL, chatModel, embedModel string, httpClient *http.Client) *Ollama {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Ollama{
		BaseURL:    baseURL,
		ChatModel:  chatModel,
		EmbedModel: embedModel,
		HTTP:       httpClient,
	}
}

type embedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type embedResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (o *Ollama) Embed(ctx context.Context, text string) ([]float32, error) {
	body, _ := json.Marshal(embedRequest{Model: o.EmbedModel, Prompt: text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embed: status %d", resp.StatusCode)
	}
	var out embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ollama embed decode: %w", err)
	}
	if len(out.Embedding) == 0 {
		return nil, fmt.Errorf("ollama embed: empty vector")
	}
	return out.Embedding, nil
}

// ChatMessage is one turn in a chat conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Options  *chatOptions  `json:"options,omitempty"`
}

type chatOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumCtx      int     `json:"num_ctx,omitempty"`
}

type chatChunk struct {
	Message ChatMessage `json:"message"`
	Done    bool        `json:"done"`
}

// ChatStream streams tokens from Ollama's /api/chat endpoint.
func (o *Ollama) ChatStream(ctx context.Context, msgs []ChatMessage) (<-chan string, <-chan error) {
	tokens := make(chan string, 32)
	errCh := make(chan error, 1)

	go func() {
		defer close(tokens)
		defer close(errCh)

		body, _ := json.Marshal(chatRequest{
			Model:    o.ChatModel,
			Messages: msgs,
			Stream:   true,
			Options:  &chatOptions{Temperature: 0.2, NumCtx: 8192},
		})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/api/chat", bytes.NewReader(body))
		if err != nil {
			errCh <- err
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := o.HTTP.Do(req)
		if err != nil {
			errCh <- fmt.Errorf("ollama chat: %w", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			errCh <- fmt.Errorf("ollama chat: status %d", resp.StatusCode)
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			var ch chatChunk
			if err := json.Unmarshal(line, &ch); err != nil {
				errCh <- fmt.Errorf("ollama chat decode: %w", err)
				return
			}
			if ch.Message.Content != "" {
				select {
				case tokens <- ch.Message.Content:
				case <-ctx.Done():
					return
				}
			}
			if ch.Done {
				return
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		}
	}()

	return tokens, errCh
}
