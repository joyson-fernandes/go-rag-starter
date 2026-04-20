package ragbot

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Ping checks the database connection.
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// CorpusVersion returns the stored hash + chunk count, or ("", 0, nil) if none.
func (s *Store) CorpusVersion(ctx context.Context) (string, int, error) {
	var hash string
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT content_hash, chunk_count FROM ragbot_corpus_version WHERE id = 1`).
		Scan(&hash, &count)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", 0, nil
		}
		return "", 0, err
	}
	return hash, count, nil
}

// ReplaceCorpus truncates ragbot_chunks + upserts the version row + bulk
// inserts the new chunks with their embeddings, atomically.
func (s *Store) ReplaceCorpus(ctx context.Context, chunks []Chunk, embeddings [][]float32, contentHash string) error {
	if len(chunks) != len(embeddings) {
		return fmt.Errorf("chunks/embeddings length mismatch")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `TRUNCATE ragbot_chunks`); err != nil {
		return fmt.Errorf("truncate chunks: %w", err)
	}
	for i, c := range chunks {
		if _, err := tx.Exec(ctx,
			`INSERT INTO ragbot_chunks (source_path, heading, content, token_count, embedding)
			 VALUES ($1, $2, $3, $4, $5::vector)`,
			c.SourcePath, c.Heading, c.Content, c.TokenCount, vectorLiteral(embeddings[i]),
		); err != nil {
			return fmt.Errorf("insert chunk %d: %w", i, err)
		}
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO ragbot_corpus_version (id, content_hash, chunk_count, indexed_at)
		 VALUES (1, $1, $2, NOW())
		 ON CONFLICT (id) DO UPDATE
		   SET content_hash = EXCLUDED.content_hash,
		       chunk_count  = EXCLUDED.chunk_count,
		       indexed_at   = EXCLUDED.indexed_at`,
		contentHash, len(chunks),
	); err != nil {
		return fmt.Errorf("upsert version: %w", err)
	}
	return tx.Commit(ctx)
}

type Retrieved struct {
	SourcePath string
	Heading    string
	Content    string
	Score      float64
}

// HybridSearch runs vector similarity + BM25-style full-text search, fuses
// with Reciprocal Rank Fusion (k=60), returns top-k.
func (s *Store) HybridSearch(ctx context.Context, queryText string, queryEmbedding []float32, k int) ([]Retrieved, error) {
	if k <= 0 {
		k = 6
	}
	const fetch = 20
	rows, err := s.pool.Query(ctx,
		`WITH vec AS (
		     SELECT id, source_path, heading, content,
		            row_number() OVER (ORDER BY embedding <=> $1::vector) AS r
		     FROM ragbot_chunks
		     ORDER BY embedding <=> $1::vector
		     LIMIT $3
		 ),
		 fts AS (
		     SELECT id, source_path, heading, content,
		            row_number() OVER (ORDER BY ts_rank(to_tsvector('english', content), plainto_tsquery('english', $2)) DESC) AS r
		     FROM ragbot_chunks
		     WHERE to_tsvector('english', content) @@ plainto_tsquery('english', $2)
		     LIMIT $3
		 ),
		 fused AS (
		     SELECT COALESCE(v.id, f.id) AS id,
		            COALESCE(v.source_path, f.source_path) AS source_path,
		            COALESCE(v.heading, f.heading) AS heading,
		            COALESCE(v.content, f.content) AS content,
		            COALESCE(1.0 / (60 + v.r), 0) + COALESCE(1.0 / (60 + f.r), 0) AS score
		     FROM vec v FULL OUTER JOIN fts f ON v.id = f.id
		 )
		 SELECT source_path, heading, content, score
		 FROM fused ORDER BY score DESC LIMIT $4`,
		vectorLiteral(queryEmbedding), queryText, fetch, k,
	)
	if err != nil {
		return nil, fmt.Errorf("hybrid search: %w", err)
	}
	defer rows.Close()
	var out []Retrieved
	for rows.Next() {
		var r Retrieved
		if err := rows.Scan(&r.SourcePath, &r.Heading, &r.Content, &r.Score); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) CreateConversation(ctx context.Context, userID string) (uuid.UUID, error) {
	var id uuid.UUID
	var ptr *string
	if userID != "" {
		ptr = &userID
	}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO ragbot_conversations (user_id) VALUES ($1) RETURNING id`, ptr).Scan(&id)
	return id, err
}

func (s *Store) InsertMessage(ctx context.Context, convID uuid.UUID, role, content string, sources []string) (int64, error) {
	var srcJSON any
	if len(sources) > 0 {
		b, _ := json.Marshal(sources)
		srcJSON = b
	}
	var id int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO ragbot_messages (conversation_id, role, content, sources)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		convID, role, content, srcJSON,
	).Scan(&id)
	return id, err
}

func (s *Store) UpdateMessageContent(ctx context.Context, id int64, content string) error {
	_, err := s.pool.Exec(ctx, `UPDATE ragbot_messages SET content = $1 WHERE id = $2`, content, id)
	return err
}

func (s *Store) UpdateFeedback(ctx context.Context, messageID int64, feedback int16) error {
	_, err := s.pool.Exec(ctx, `UPDATE ragbot_messages SET feedback = $1 WHERE id = $2`, feedback, messageID)
	return err
}

func vectorLiteral(v []float32) string {
	var b strings.Builder
	b.Grow(len(v) * 10)
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(x), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}
