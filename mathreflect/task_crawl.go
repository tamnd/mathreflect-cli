package mathreflect

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	crawlFlushEvery    = 50
	crawlFlushInterval = 15 * time.Second
	crawlStaleAfter    = 10 * time.Minute
)

// CrawlTask downloads issue PDFs and extracts problems into the DB.
type CrawlTask struct {
	Config  Config
	Client  *Client
	DB      *DB
	StateDB *State
}

type workResult struct {
	item     QueueItem
	problems []*Problem
	code     int
	err      error
}

type writeBuffer struct {
	problems  []*Problem
	done      []struct{ url, typ string; code int }
	failed    []struct{ url, msg string }
	size      int
	lastFlush time.Time
}

func (wb *writeBuffer) add(r workResult) {
	wb.problems = append(wb.problems, r.problems...)
	if r.err != nil {
		wb.failed = append(wb.failed, struct{ url, msg string }{r.item.URL, r.err.Error()})
	} else {
		wb.done = append(wb.done, struct{ url, typ string; code int }{r.item.URL, r.item.EntityType, r.code})
	}
	wb.size++
}

func (wb *writeBuffer) ready() bool {
	return wb.size >= crawlFlushEvery || (wb.size > 0 && time.Since(wb.lastFlush) >= crawlFlushInterval)
}

func (wb *writeBuffer) flush(db *DB, state *State, exportDir string) (int, error) {
	if wb.size == 0 {
		return 0, nil
	}
	if len(wb.problems) > 0 {
		batch := make([]Problem, len(wb.problems))
		for i, p := range wb.problems {
			batch[i] = *p
		}
		if err := db.BatchUpsert(batch); err != nil {
			return 0, err
		}
	}
	for _, d := range wb.done {
		_ = state.Done(d.url, d.code, d.typ)
	}
	for _, f := range wb.failed {
		_ = state.Fail(f.url, f.msg)
	}
	exported := 0
	if exportDir != "" {
		for _, p := range wb.problems {
			if p.ContentMD != "" {
				if err := writeProblemFile(exportDir, *p); err == nil {
					exported++
				}
			}
		}
	}
	*wb = writeBuffer{lastFlush: time.Now()}
	return exported, nil
}

// Run executes the crawl task.
func (t *CrawlTask) Run(ctx context.Context, emit func(*CrawlState)) (CrawlMetric, error) {
	start := time.Now()
	var done atomic.Int64
	var failed atomic.Int64
	var exported atomic.Int64
	var lastPending atomic.Int64
	var inFlight atomic.Int64

	workers := t.Config.Workers
	if workers <= 0 {
		workers = DefaultWorkers
	}

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				elapsed := time.Since(start).Seconds()
				rps := 0.0
				if elapsed > 0 {
					rps = float64(done.Load()) / elapsed
				}
				emit(&CrawlState{
					Done:     done.Load(),
					Pending:  lastPending.Load(),
					Failed:   failed.Load(),
					Exported: exported.Load(),
					RPS:      rps,
				})
			}
		}
	}()

	results := make(chan workResult, workers*8)
	writerDone := make(chan struct{})

	go func() {
		defer close(writerDone)
		buf := &writeBuffer{lastFlush: time.Now()}
		doFlush := func() {
			n, err := buf.flush(t.DB, t.StateDB, t.Config.ExportDir)
			if err != nil {
				fmt.Printf("\nflush error: %v\n", err)
			}
			exported.Add(int64(n))
		}
		for r := range results {
			if r.err != nil {
				failed.Add(1)
			} else {
				done.Add(1)
			}
			buf.add(r)
			if buf.ready() {
				doFlush()
			}
			inFlight.Add(-1)
		}
		doFlush()
	}()

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	popBatch := workers * 4
	if popBatch < 4 {
		popBatch = 4
	}

	for {
		if ctx.Err() != nil {
			break
		}
		items, _ := t.StateDB.Pop(popBatch)
		if len(items) == 0 {
			if inFlight.Load() == 0 {
				break
			}
			time.Sleep(50 * time.Millisecond)
			continue
		}
		inFlight.Add(int64(len(items)))
		pending, _, _, _ := t.StateDB.QueueStats()
		lastPending.Store(pending)

		for _, it := range items {
			sem <- struct{}{}
			wg.Add(1)
			go func(item QueueItem) {
				defer wg.Done()
				defer func() { <-sem }()
				results <- t.process(ctx, item)
			}(it)
		}
	}

	wg.Wait()
	close(results)
	<-writerDone

	return CrawlMetric{
		Done:     done.Load(),
		Failed:   failed.Load(),
		Exported: exported.Load(),
		Duration: time.Since(start),
	}, nil
}

func (t *CrawlTask) process(ctx context.Context, item QueueItem) workResult {
	res := workResult{item: item, code: 200}

	year, num := ExtractIssueYearNum(item.URL)
	if year == 0 {
		res.err = fmt.Errorf("cannot parse year/num from URL: %s", item.URL)
		return res
	}

	pdfBytes, code, err := t.Client.FetchPDF(ctx, item.URL)
	res.code = code
	if err != nil {
		res.err = err
		return res
	}
	if code == 404 || pdfBytes == nil {
		// Expected for issues that don't exist; mark done silently.
		return res
	}

	text, err := ExtractTextFromPDF(pdfBytes)
	if err != nil {
		res.err = fmt.Errorf("extract text: %w", err)
		return res
	}

	parsed := ParseProblemsFromText(text)
	now := time.Now()
	for _, pp := range parsed {
		id := ProblemID(year, num, pp.Section, pp.ProblemNum)
		res.problems = append(res.problems, &Problem{
			ID:         id,
			IssueYear:  year,
			IssueNum:   num,
			Section:    pp.Section,
			ProblemNum: pp.ProblemNum,
			ContentMD:  pp.Text,
			URL:        item.URL,
			FetchedAt:  now,
		})
	}

	return res
}
