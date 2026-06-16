package mathreflect_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tamnd/mathreflect-cli/mathreflect"
)

func TestDBRoundtrip(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := mathreflect.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	problems := []mathreflect.Problem{
		{
			ID:         "2026-3-O-5",
			IssueYear:  2026,
			IssueNum:   3,
			Section:    "O",
			ProblemNum: 5,
			ContentMD:  "Find all integers...",
			URL:        "https://example.com/mr_3_2026_problems.pdf",
			FetchedAt:  time.Now().UTC().Truncate(time.Second),
		},
		{
			ID:         "2026-3-J-1",
			IssueYear:  2026,
			IssueNum:   3,
			Section:    "J",
			ProblemNum: 1,
			ContentMD:  "Let n be a positive integer...",
			URL:        "https://example.com/mr_3_2026_problems.pdf",
			FetchedAt:  time.Now().UTC().Truncate(time.Second),
		},
	}

	if err := db.BatchUpsert(problems); err != nil {
		t.Fatalf("BatchUpsert: %v", err)
	}

	got, err := db.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListAll returned %d problems, want 2", len(got))
	}

	stats, err := db.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Total != 2 {
		t.Errorf("Stats.Total = %d, want 2", stats.Total)
	}
	if stats.WithBody != 2 {
		t.Errorf("Stats.WithBody = %d, want 2", stats.WithBody)
	}

	// Upsert again -- should not duplicate.
	if err := db.BatchUpsert(problems); err != nil {
		t.Fatalf("second BatchUpsert: %v", err)
	}
	got2, _ := db.ListAll()
	if len(got2) != 2 {
		t.Errorf("after re-upsert, got %d problems, want 2", len(got2))
	}

	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("DB file not found: %v", err)
	}
}
