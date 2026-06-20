package mathreflect

import (
	"context"
	"fmt"
	"time"
)

// SeedTask fetches the archive index and enqueues all issue PDF URLs.
type SeedTask struct {
	Config  Config
	Client  *Client
	StateDB *State
}

// Run executes the seed task. It emits progress via emit.
func (t *SeedTask) Run(ctx context.Context, emit func(*SeedState)) (SeedMetric, error) {
	start := time.Now()
	state := &SeedState{}

	archiveResult, err := t.Client.FetchArchiveIndex(ctx)
	if err != nil {
		return SeedMetric{}, fmt.Errorf("fetch archive index: %w", err)
	}

	for _, pdfURL := range archiveResult.PDFURLs {
		if ctx.Err() != nil {
			break
		}
		state.Discovered++
		if !t.StateDB.IsVisited(pdfURL) {
			t.StateDB.Enqueue(pdfURL, EntityIssue, PriorityIssue)
			state.Enqueued++
		}
		emit(state)
	}

	return SeedMetric{
		Discovered: state.Discovered,
		Enqueued:   state.Enqueued,
		Duration:   time.Since(start),
	}, nil
}
