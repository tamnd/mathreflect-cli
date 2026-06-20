package mathreflect

import (
	"container/heap"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	stateFlushCount    = 200
	stateFlushInterval = 30 * time.Second
)

type memEntry struct {
	URL        string
	EntityType string
	Priority   int
	Status     string // pending/in_progress/done/failed
	Attempts   int
	Err        string
	CreatedAt  time.Time
}

type memPQ []*memEntry

func (pq memPQ) Len() int { return len(pq) }
func (pq memPQ) Less(i, j int) bool {
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority > pq[j].Priority
	}
	return pq[i].CreatedAt.Before(pq[j].CreatedAt)
}
func (pq memPQ) Swap(i, j int)  { pq[i], pq[j] = pq[j], pq[i] }
func (pq *memPQ) Push(x any)    { *pq = append(*pq, x.(*memEntry)) }
func (pq *memPQ) Pop() any {
	old := *pq
	n := len(old)
	x := old[n-1]
	*pq = old[:n-1]
	return x
}

// State is the in-memory crawl queue backed by a SQLite database.
type State struct {
	mu      sync.Mutex
	entries map[string]*memEntry
	pq      memPQ

	pendingCnt int64
	inProgCnt  int64
	doneCnt    int64
	failedCnt  int64

	db        *sql.DB
	path      string
	dirtyDone int
	lastFlush time.Time
}

// OpenState opens (or creates) the state database at path.
func OpenState(path string) (*State, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open state sqlite %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)

	s := &State{
		entries:   make(map[string]*memEntry),
		pq:        make(memPQ, 0, 1024),
		db:        db,
		path:      path,
		lastFlush: time.Now(),
	}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	if err := s.loadFromDB(); err != nil {
		db.Close()
		return nil, err
	}
	heap.Init(&s.pq)
	return s, nil
}

func (s *State) initSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS queue (
			url         TEXT PRIMARY KEY,
			entity_type TEXT NOT NULL,
			priority    INTEGER DEFAULT 0,
			status      TEXT DEFAULT 'pending',
			attempts    INTEGER DEFAULT 0,
			error       TEXT DEFAULT '',
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS visited (
			url         TEXT PRIMARY KEY,
			fetched_at  DATETIME,
			status_code INTEGER,
			entity_type TEXT
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("init state schema: %w", err)
		}
	}
	return nil
}

func (s *State) loadFromDB() error {
	rows, err := s.db.Query(`SELECT url, entity_type FROM visited`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var u, typ string
		if rows.Scan(&u, &typ) == nil {
			e := &memEntry{URL: u, EntityType: typ, Status: "done", CreatedAt: time.Now()}
			s.entries[u] = e
			s.doneCnt++
		}
	}
	rows.Close()

	rows, err = s.db.Query(`SELECT url, entity_type, priority, status, attempts, COALESCE(error,''), created_at FROM queue`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var e memEntry
		var createdAt string
		if err := rows.Scan(&e.URL, &e.EntityType, &e.Priority, &e.Status, &e.Attempts, &e.Err, &createdAt); err != nil {
			continue
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if e.Status == "in_progress" {
			e.Status = "pending"
		}
		if _, exists := s.entries[e.URL]; exists {
			continue
		}
		s.entries[e.URL] = &e
		switch e.Status {
		case "pending":
			s.pendingCnt++
			heap.Push(&s.pq, &e)
		case "failed":
			s.failedCnt++
		}
	}
	rows.Close()
	return nil
}

// Enqueue adds url to the queue if not already present.
func (s *State) Enqueue(url, entityType string, priority int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.entries[url]; ok {
		return
	}
	e := &memEntry{
		URL:        url,
		EntityType: entityType,
		Priority:   priority,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}
	s.entries[url] = e
	heap.Push(&s.pq, e)
	s.pendingCnt++
}

// Pop returns up to n pending items, marking them in_progress.
func (s *State) Pop(n int) ([]QueueItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var items []QueueItem
	for s.pq.Len() > 0 && len(items) < n {
		e := heap.Pop(&s.pq).(*memEntry)
		if s.entries[e.URL] != e {
			continue
		}
		e.Status = "in_progress"
		e.Attempts++
		s.pendingCnt--
		s.inProgCnt++
		items = append(items, QueueItem{URL: e.URL, EntityType: e.EntityType, Priority: e.Priority})
	}
	return items, nil
}

// Done marks an item as successfully completed.
func (s *State) Done(url string, code int, entityType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[url]
	if !ok {
		return nil
	}
	if e.Status == "in_progress" {
		s.inProgCnt--
	}
	e.Status = "done"
	s.doneCnt++
	s.dirtyDone++
	if s.dirtyDone >= stateFlushCount || time.Since(s.lastFlush) >= stateFlushInterval {
		go s.Flush()
	}
	return nil
}

// Fail marks an item as failed, re-enqueuing it if attempts remain.
func (s *State) Fail(url, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[url]
	if !ok {
		return nil
	}
	if e.Status == "in_progress" {
		s.inProgCnt--
	}
	e.Err = errMsg
	if e.Attempts >= 3 {
		e.Status = "failed"
		s.failedCnt++
	} else {
		e.Status = "pending"
		s.pendingCnt++
		captured := e
		go func() {
			time.Sleep(time.Duration(captured.Attempts*captured.Attempts) * 10 * time.Second)
			s.mu.Lock()
			heap.Push(&s.pq, captured)
			s.mu.Unlock()
		}()
	}
	return nil
}

// IsVisited reports whether url has been marked done.
func (s *State) IsVisited(url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[url]
	return ok && e.Status == "done"
}

// QueueStats returns counts by status.
func (s *State) QueueStats() (pending, inProgress, done, failed int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pendingCnt, s.inProgCnt, s.doneCnt, s.failedCnt
}

// PendingCount returns the number of pending items.
func (s *State) PendingCount() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pendingCnt
}

// Flush persists in-memory state to SQLite.
func (s *State) Flush() error {
	s.mu.Lock()
	var done []*memEntry
	var queue []*memEntry
	for _, e := range s.entries {
		switch e.Status {
		case "done":
			done = append(done, e)
		default:
			queue = append(queue, e)
		}
	}
	s.dirtyDone = 0
	s.lastFlush = time.Now()
	s.mu.Unlock()

	return s.persistToDB(done, queue)
}

func (s *State) persistToDB(done, queueRows []*memEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, e := range done {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO visited (url, fetched_at, status_code, entity_type) VALUES (?, ?, ?, ?)`,
			e.URL, time.Now().UTC().Format(time.RFC3339), 200, e.EntityType); err != nil {
			return err
		}
	}
	for _, e := range queueRows {
		status := e.Status
		if status == "in_progress" {
			status = "pending"
		}
		if _, err := tx.Exec(`INSERT OR REPLACE INTO queue (url, entity_type, priority, status, attempts, error, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			e.URL, e.EntityType, e.Priority, status, e.Attempts, e.Err,
			e.CreatedAt.UTC().Format(time.RFC3339)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Close flushes and closes the state database.
func (s *State) Close() error {
	_ = s.Flush()
	return s.db.Close()
}
