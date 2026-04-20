package ragbot

import (
	"bufio"
	"strings"
)

// Chunk is one retrievable unit of markdown.
type Chunk struct {
	SourcePath string
	Heading    string
	Content    string
	TokenCount int
}

// ChunkMarkdown splits a markdown document on H2 (## ) headings. Sections
// longer than maxChars are sub-split at paragraph boundaries with overlap
// so context isn't lost at the seam.
func ChunkMarkdown(sourcePath, body string, maxChars, overlapChars int) []Chunk {
	if maxChars <= 0 {
		maxChars = 1600 // ~400 tokens at ~4 chars/token
	}
	if overlapChars < 0 || overlapChars >= maxChars {
		overlapChars = 200
	}

	type section struct {
		heading string
		lines   []string
	}
	var sections []section
	cur := section{}

	scanner := bufio.NewScanner(strings.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") {
			if cur.heading != "" || len(cur.lines) > 0 {
				sections = append(sections, cur)
			}
			cur = section{heading: strings.TrimSpace(strings.TrimPrefix(line, "## "))}
			continue
		}
		cur.lines = append(cur.lines, line)
	}
	if cur.heading != "" || len(cur.lines) > 0 {
		sections = append(sections, cur)
	}

	var chunks []Chunk
	for _, s := range sections {
		text := strings.TrimSpace(strings.Join(s.lines, "\n"))
		if text == "" && s.heading == "" {
			continue
		}
		for _, piece := range splitWithOverlap(text, maxChars, overlapChars) {
			chunks = append(chunks, Chunk{
				SourcePath: sourcePath,
				Heading:    s.heading,
				Content:    piece,
				TokenCount: approxTokens(piece),
			})
		}
	}
	return chunks
}

func splitWithOverlap(text string, maxChars, overlap int) []string {
	if len(text) <= maxChars {
		return []string{text}
	}
	var out []string
	for start := 0; start < len(text); {
		end := start + maxChars
		if end >= len(text) {
			out = append(out, text[start:])
			break
		}
		cut := end
		if i := strings.LastIndex(text[start:end], "\n\n"); i > maxChars/2 {
			cut = start + i
		} else if i := strings.LastIndex(text[start:end], ". "); i > maxChars/2 {
			cut = start + i + 1
		}
		out = append(out, strings.TrimSpace(text[start:cut]))
		start = cut - overlap
		if start < 0 {
			start = 0
		}
	}
	return out
}

func approxTokens(s string) int { return len(s) / 4 }
