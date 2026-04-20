package ragbot

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Service struct {
	Store  *Store
	Ollama *Ollama
	Config Config
}

func NewService(store *Store, ollama *Ollama, cfg Config) *Service {
	return &Service{Store: store, Ollama: ollama, Config: cfg}
}

type StreamResult struct {
	Header  *QueryHeader
	Token   string
	Sources []string
	Done    bool
	Err     error
}

type QueryInput struct {
	UserID         string // "" for anonymous
	ConversationID uuid.UUID
	Message        string
}

type QueryHeader struct {
	ConversationID uuid.UUID
	AssistantMsgID int64
	Sources        []string
}

// Query embeds the user's message, runs hybrid retrieval, streams tokens
// from the LLM, and returns the final assembled content via side-effect
// on the Store. Events flow on `out`; the header is always the first one.
func (s *Service) Query(ctx context.Context, in QueryInput, out chan<- StreamResult) error {
	convID := in.ConversationID
	if convID == uuid.Nil {
		id, err := s.Store.CreateConversation(ctx, in.UserID)
		if err != nil {
			return fmt.Errorf("create conversation: %w", err)
		}
		convID = id
	}

	if _, err := s.Store.InsertMessage(ctx, convID, "user", in.Message, nil); err != nil {
		return fmt.Errorf("insert user msg: %w", err)
	}

	queryVec, err := s.Ollama.Embed(ctx, in.Message)
	if err != nil {
		return fmt.Errorf("embed query: %w", err)
	}
	retrieved, err := s.Store.HybridSearch(ctx, in.Message, queryVec, 6)
	if err != nil {
		return fmt.Errorf("retrieve: %w", err)
	}
	sources := UniqueSources(retrieved)

	assistantID, err := s.Store.InsertMessage(ctx, convID, "assistant", "", sources)
	if err != nil {
		return fmt.Errorf("insert assistant msg: %w", err)
	}

	hdr := QueryHeader{
		ConversationID: convID,
		AssistantMsgID: assistantID,
		Sources:        sources,
	}
	out <- StreamResult{Header: &hdr}

	msgs := BuildMessages(s.Config, nil, retrieved, in.Message)

	tokens, errCh := s.Ollama.ChatStream(ctx, msgs)
	full := make([]byte, 0, 2048)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t, ok := <-tokens:
			if !ok {
				_ = s.Store.UpdateMessageContent(ctx, assistantID, string(full))
				out <- StreamResult{Done: true, Sources: sources}
				return nil
			}
			full = append(full, t...)
			out <- StreamResult{Token: t}
		case err := <-errCh:
			if err == nil {
				continue
			}
			return err
		}
	}
}
