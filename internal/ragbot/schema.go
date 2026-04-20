package ragbot

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureSchema creates the pgvector extension + all tables + indexes on
// startup. Idempotent. If pgvector is missing, returns an error and the
// caller should log + exit (the bot needs vectors to work).
func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	stmts := []string{
		`CREATE EXTENSION IF NOT EXISTS vector`,

		`CREATE TABLE IF NOT EXISTS ragbot_corpus_version (
            id           INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
            content_hash TEXT NOT NULL,
            chunk_count  INT NOT NULL,
            indexed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )`,

		`CREATE TABLE IF NOT EXISTS ragbot_chunks (
            id          BIGSERIAL PRIMARY KEY,
            source_path TEXT NOT NULL,
            heading     TEXT,
            content     TEXT NOT NULL,
            token_count INT NOT NULL,
            embedding   vector(768) NOT NULL,
            created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )`,

		`CREATE INDEX IF NOT EXISTS ragbot_chunks_hnsw
            ON ragbot_chunks USING hnsw (embedding vector_cosine_ops)`,

		`CREATE INDEX IF NOT EXISTS ragbot_chunks_fts
            ON ragbot_chunks USING GIN (to_tsvector('english', content))`,

		`CREATE TABLE IF NOT EXISTS ragbot_conversations (
            id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id    TEXT,                       -- free-form; nil for anonymous
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )`,

		`CREATE INDEX IF NOT EXISTS ragbot_conversations_user_ts
            ON ragbot_conversations (user_id, created_at DESC)`,

		`CREATE TABLE IF NOT EXISTS ragbot_messages (
            id              BIGSERIAL PRIMARY KEY,
            conversation_id UUID NOT NULL REFERENCES ragbot_conversations(id) ON DELETE CASCADE,
            role            TEXT NOT NULL CHECK (role IN ('user','assistant')),
            content         TEXT NOT NULL,
            sources         JSONB,
            feedback        SMALLINT,
            created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )`,

		`CREATE INDEX IF NOT EXISTS ragbot_messages_conv_ts
            ON ragbot_messages (conversation_id, created_at)`,
	}
	for _, s := range stmts {
		if _, err := pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("ensure schema: %w", err)
		}
	}
	return nil
}
