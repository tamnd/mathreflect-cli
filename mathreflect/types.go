package mathreflect

import (
	"fmt"
	"time"
)

const (
	// EntityIssue is the entity type for issue PDF queue items.
	EntityIssue = "issue"

	// PriorityIssue is the queue priority for issues.
	PriorityIssue = 5
)

// Problem is one row in the problems table.
type Problem struct {
	ID         string
	IssueYear  int
	IssueNum   int
	Section    string
	ProblemNum int
	ContentMD  string
	URL        string
	FetchedAt  time.Time
}

// QueueItem is one item in the crawl queue.
type QueueItem struct {
	URL        string
	EntityType string
	Priority   int
}

// DBStats summarises the DB for the info command.
type DBStats struct {
	Total    int64
	WithBody int64
	DBSize   int64
}

// SeedState is the live progress struct emitted during seed.
type SeedState struct {
	Discovered int
	Enqueued   int
}

// SeedMetric is returned by SeedTask.Run.
type SeedMetric struct {
	Discovered int
	Enqueued   int
	Duration   time.Duration
}

// CrawlState is the live progress struct emitted during crawl.
type CrawlState struct {
	Done     int64
	Pending  int64
	Failed   int64
	Exported int64
	RPS      float64
}

// CrawlMetric is returned by CrawlTask.Run.
type CrawlMetric struct {
	Done     int64
	Failed   int64
	Exported int64
	Duration time.Duration
}

// ExportState is the live progress struct emitted during export.
type ExportState struct {
	Written int
	Current string
}

// ExportMetric is returned by ExportTask.Run.
type ExportMetric struct {
	Files    int
	Duration time.Duration
}

// ProblemID builds the canonical problem identifier.
// Format: {year}-{issueNum}-{section}-{problemNum}
func ProblemID(year, issueNum int, section string, problemNum int) string {
	return fmt.Sprintf("%d-%d-%s-%d", year, issueNum, section, problemNum)
}
