package ragbot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"
)

// Ingester chunks + embeds the seed corpus into the store when the content
// hash changes. Safe to call on every startup.
type Ingester struct {
	Corpus fs.FS
	Store  *Store
	Ollama *Ollama
}

func (in *Ingester) Sync(ctx context.Context) error {
	docs, err := loadDocs(in.Corpus)
	if err != nil {
		return fmt.Errorf("load docs: %w", err)
	}
	if len(docs) == 0 {
		log.Printf("[ragbot] no seed docs found — drop markdown into docs/ and restart")
		return nil
	}

	hash := corpusHash(docs)
	stored, count, err := in.Store.CorpusVersion(ctx)
	if err != nil {
		return fmt.Errorf("read version: %w", err)
	}
	if stored == hash {
		log.Printf("[ragbot] corpus unchanged (%s), %d chunks already indexed", shortHash(hash), count)
		return nil
	}

	log.Printf("[ragbot] corpus hash changed, reindexing %d docs", len(docs))

	var chunks []Chunk
	for _, d := range docs {
		chunks = append(chunks, ChunkMarkdown(d.path, d.body, 1600, 200)...)
	}
	log.Printf("[ragbot] chunked into %d pieces, embedding with %s", len(chunks), in.Ollama.EmbedModel)

	embeds := make([][]float32, len(chunks))
	for i, c := range chunks {
		v, err := in.Ollama.Embed(ctx, chunkInput(c))
		if err != nil {
			return fmt.Errorf("embed chunk %d (%s): %w", i, c.SourcePath, err)
		}
		embeds[i] = v
	}

	if err := in.Store.ReplaceCorpus(ctx, chunks, embeds, hash); err != nil {
		return fmt.Errorf("replace corpus: %w", err)
	}
	log.Printf("[ragbot] indexed %d chunks at hash %s", len(chunks), shortHash(hash))
	return nil
}

type doc struct {
	path string
	body string
}

func loadDocs(f fs.FS) ([]doc, error) {
	var docs []doc
	err := fs.WalkDir(f, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		b, err := fs.ReadFile(f, p)
		if err != nil {
			return err
		}
		docs = append(docs, doc{path: p, body: string(b)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].path < docs[j].path })
	return docs, nil
}

func corpusHash(docs []doc) string {
	h := sha256.New()
	for _, d := range docs {
		h.Write([]byte(d.path))
		h.Write([]byte{0})
		h.Write([]byte(d.body))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func shortHash(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

func chunkInput(c Chunk) string {
	if c.Heading == "" {
		return c.Content
	}
	return c.Heading + "\n\n" + c.Content
}
