// Package kb provides keyword retrieval over the embedded support policy notes.
package kb

import (
	"embed"
	"math"
	"path"
	"regexp"
	"sort"
	"strings"
)

//go:embed data/*.md
var files embed.FS

var tokenPattern = regexp.MustCompile(`[a-z0-9][a-z0-9\-']+`)

// Doc is one policy note.
type Doc struct {
	ID    string
	Title string
	Body  string
}

// Hit is a document plus its relevance score.
type Hit struct {
	Doc
	Score float64
}

// Base is an in-memory index over the embedded policy documents.
type Base struct {
	docs []Doc
	df   map[string]int
}

// Load parses every embedded Markdown file.
func Load() (*Base, error) {
	entries, err := files.ReadDir("data")
	if err != nil {
		return nil, err
	}

	base := &Base{df: map[string]int{}}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		raw, err := files.ReadFile(path.Join("data", entry.Name()))
		if err != nil {
			return nil, err
		}
		body := string(raw)
		title := strings.ReplaceAll(strings.TrimSuffix(entry.Name(), ".md"), "-", " ")
		for _, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(line, "# ") {
				title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
				break
			}
		}
		base.docs = append(base.docs, Doc{
			ID:    strings.TrimSuffix(entry.Name(), ".md"),
			Title: title,
			Body:  strings.TrimSpace(body),
		})
		for term := range uniqueTerms(body) {
			base.df[term]++
		}
	}
	return base, nil
}

// Documents returns the indexed documents.
func (b *Base) Documents() []Doc { return b.docs }

// Search returns the top-k documents matching the query.
func (b *Base) Search(query string, k int) []Hit {
	if k <= 0 {
		k = 3
	}
	terms := uniqueTerms(query)
	hits := make([]Hit, 0, len(b.docs))

	for _, doc := range b.docs {
		docTerms := uniqueTerms(doc.Body)
		score := 0.0
		for term := range terms {
			if _, ok := docTerms[term]; !ok {
				continue
			}
			score += math.Log(float64(len(b.docs)+1)/float64(b.df[term]+1)) + 1
		}
		if score > 0 {
			hits = append(hits, Hit{Doc: doc, Score: score})
		}
	}

	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > k {
		hits = hits[:k]
	}
	return hits
}

func uniqueTerms(text string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, token := range tokenPattern.FindAllString(strings.ToLower(text), -1) {
		out[token] = struct{}{}
	}
	return out
}
