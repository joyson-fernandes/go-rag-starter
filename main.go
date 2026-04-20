// go-rag-starter — a self-hosted, self-documenting RAG help bot.
//
// Clone → docker-compose up → visit localhost:8080. Ask the bot about
// itself. Then replace docs/ with your own markdown. See README.md for
// the full walkthrough.
package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joyson-fernandes/go-rag-starter/internal/ragbot"
)

// CorpusFS holds the markdown files under docs/ — both the human-readable
// onboarding guide AND the seed corpus the bot answers from. Adding,
// editing, or removing a file here triggers a content-hash-gated re-index
// on the next service start.
//
//go:embed docs/*.md
var CorpusFS embed.FS

// WebFS holds the built-in demo HTML page + the widget script.
//
//go:embed web/*.html web/*.js
var WebFS embed.FS

func main() {
	cfg := ragbot.Config{
		ProductName:  envOr("PRODUCT_NAME", "this bot"),
		ProductBlurb: envOr("PRODUCT_BLURB", "a self-hosted RAG help bot built with go-rag-starter."),
		RefusePhrase: envOr("REFUSE_PHRASE", "I don't have that in the docs yet — try asking something else, or check the project README."),
	}
	port, _ := strconv.Atoi(envOr("SERVICE_PORT", "8080"))
	dsn := envOr("DB_DSN", "postgres://ragbot:ragbot@localhost:5432/ragbot?sslmode=disable")
	ollamaURL := envOr("OLLAMA_URL", "http://localhost:11434")
	chatModel := envOr("OLLAMA_CHAT_MODEL", "gemma3:4b")
	embedModel := envOr("OLLAMA_EMBED_MODEL", "nomic-embed-text")

	ctx := context.Background()

	// ─── Connect to PostgreSQL ─────────────────────────────
	log.Println("[ragbot] connecting to PostgreSQL...")
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()
	for i := 0; i < 30; i++ {
		if err := pool.Ping(ctx); err == nil {
			break
		}
		if i == 29 {
			log.Fatalf("postgres ping timeout")
		}
		time.Sleep(time.Second)
	}

	// ─── Schema (idempotent) ───────────────────────────────
	if err := ragbot.EnsureSchema(ctx, pool); err != nil {
		log.Fatalf("schema: %v — ensure your Postgres image has pgvector installed (e.g. pgvector/pgvector:pg16)", err)
	}

	// ─── Ollama client ─────────────────────────────────────
	ollama := ragbot.NewOllama(ollamaURL, chatModel, embedModel, &http.Client{})
	log.Printf("[ragbot] ollama at %s (chat=%s, embed=%s)", ollamaURL, chatModel, embedModel)

	// ─── Ingest seed corpus ────────────────────────────────
	store := ragbot.NewStore(pool)
	corpus, _ := fs.Sub(CorpusFS, "docs")
	ingester := &ragbot.Ingester{Corpus: corpus, Store: store, Ollama: ollama}
	ingestCtx, cancelIngest := context.WithTimeout(ctx, 10*time.Minute)
	if err := ingester.Sync(ingestCtx); err != nil {
		log.Printf("[ragbot] warning: corpus sync failed: %v", err)
	}
	cancelIngest()

	// ─── HTTP handler ──────────────────────────────────────
	svc := ragbot.NewService(store, ollama, cfg)
	handler := ragbot.NewHandler(svc, WebFS)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      withLogging(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // SSE streams can last minutes; per-request ctx enforces cancellation
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("[ragbot] listening on %s — open http://localhost:%d", srv.Addr, port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[ragbot] shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("[ragbot] stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// withLogging is the whole "middleware chain" — just request logging.
// No Recoverer/Metrics/Tracing wrappers here to keep the starter simple
// AND to avoid the Flusher-hiding bug that bit the full Linkvolt build.
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if r.URL.Path != "/healthz" { // quiet the probe noise
			log.Printf("[%s] %s %s (%s)", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start).Round(time.Millisecond))
		}
	})
}
