package kb

import "testing"

func TestLoadIndexesEmbeddedDocuments(t *testing.T) {
	base, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(base.Documents()) < 2 {
		t.Fatalf("expected at least two documents, got %d", len(base.Documents()))
	}
}

func TestSearchRanksRelevantDocument(t *testing.T) {
	base, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	hits := base.Search("enterprise first response target", 2)
	if len(hits) == 0 {
		t.Fatal("expected hits")
	}
	if hits[0].ID != "customer-tiers" {
		t.Fatalf("top hit = %q", hits[0].ID)
	}
}

func TestSearchWithoutTermsReturnsNothing(t *testing.T) {
	base, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if hits := base.Search("zzzz", 3); len(hits) != 0 {
		t.Fatalf("expected no hits, got %d", len(hits))
	}
}
