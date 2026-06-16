package mathreflect_test

import (
	"path/filepath"
	"testing"

	"github.com/tamnd/mathreflect-cli/mathreflect"
)

func TestStateLifecycle(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.db")

	state, err := mathreflect.OpenState(statePath)
	if err != nil {
		t.Fatalf("OpenState: %v", err)
	}
	defer state.Close()

	// Initially empty.
	p, ip, d, f := state.QueueStats()
	if p != 0 || ip != 0 || d != 0 || f != 0 {
		t.Errorf("initial stats = (%d,%d,%d,%d), want all 0", p, ip, d, f)
	}

	// Enqueue.
	url1 := "https://example.com/mr_3_2026_problems.pdf"
	url2 := "https://example.com/mr_2_2026_problems.pdf"
	state.Enqueue(url1, mathreflect.EntityIssue, mathreflect.PriorityIssue)
	state.Enqueue(url2, mathreflect.EntityIssue, mathreflect.PriorityIssue)

	// Duplicate enqueue should be a no-op.
	state.Enqueue(url1, mathreflect.EntityIssue, mathreflect.PriorityIssue)

	p, _, _, _ = state.QueueStats()
	if p != 2 {
		t.Errorf("pending = %d, want 2", p)
	}

	// Pop 1 item.
	items, err := state.Pop(1)
	if err != nil {
		t.Fatalf("Pop: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Pop returned %d items, want 1", len(items))
	}

	_, ip, _, _ = state.QueueStats()
	if ip != 1 {
		t.Errorf("in_progress = %d, want 1", ip)
	}

	// Mark done.
	if err := state.Done(items[0].URL, 200, mathreflect.EntityIssue); err != nil {
		t.Fatalf("Done: %v", err)
	}

	_, _, d, _ = state.QueueStats()
	if d != 1 {
		t.Errorf("done = %d, want 1", d)
	}

	// IsVisited.
	if !state.IsVisited(items[0].URL) {
		t.Errorf("IsVisited(%q) = false, want true", items[0].URL)
	}

	// Pop and fail.
	items2, _ := state.Pop(1)
	if len(items2) != 1 {
		t.Fatalf("Pop returned %d items (second), want 1", len(items2))
	}
	if err := state.Fail(items2[0].URL, "timeout"); err != nil {
		t.Fatalf("Fail: %v", err)
	}
	// After first failure (attempts=1 < 3), item goes back to pending.
	p2, _, _, _ := state.QueueStats()
	if p2 != 1 {
		t.Errorf("after Fail, pending = %d, want 1", p2)
	}
}
