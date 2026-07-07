package mathreflect

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite database holding problem records.
type DB struct {
	db   *sql.DB
	path string
}

// OpenDB opens (or creates) the problems database at path.
func OpenDB(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	d := &DB{db: db, path: path}
	if err := d.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return d, nil
}

// OpenReadOnly opens the database in read-only mode.
func OpenReadOnly(path string) (*DB, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return &DB{db: db, path: path}, nil
}

// Close closes the database.
func (d *DB) Close() error { return d.db.Close() }

func (d *DB) initSchema() error {
	_, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS problems (
		id           TEXT PRIMARY KEY,
		issue_year   INTEGER DEFAULT 0,
		issue_num    INTEGER DEFAULT 0,
		section      TEXT DEFAULT '',
		problem_num  INTEGER DEFAULT 0,
		content_md   TEXT DEFAULT '',
		url          TEXT DEFAULT '',
		fetched_at   DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

// BatchUpsert inserts or replaces a batch of problems in a single transaction.
func (d *DB) BatchUpsert(problems []Problem) error {
	if len(problems) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO problems
		(id, issue_year, issue_num, section, problem_num, content_md, url, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range problems {
		if _, err := stmt.Exec(p.ID, p.IssueYear, p.IssueNum, p.Section, p.ProblemNum,
			p.ContentMD, p.URL, p.FetchedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListAll returns all problems ordered by year, issue, section, number.
func (d *DB) ListAll() ([]Problem, error) {
	rows, err := d.db.Query(`SELECT id, issue_year, issue_num, section, problem_num,
		content_md, url, fetched_at
		FROM problems ORDER BY issue_year, issue_num, section, problem_num`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var problems []Problem
	for rows.Next() {
		var p Problem
		var fetchedAt string
		if err := rows.Scan(&p.ID, &p.IssueYear, &p.IssueNum, &p.Section, &p.ProblemNum,
			&p.ContentMD, &p.URL, &fetchedAt); err != nil {
			return nil, err
		}
		p.FetchedAt, _ = time.Parse(time.RFC3339, fetchedAt)
		problems = append(problems, p)
	}
	return problems, rows.Err()
}

// Stats returns aggregate statistics about the problems database.
func (d *DB) Stats() (DBStats, error) {
	var s DBStats
	_ = d.db.QueryRow(`SELECT COUNT(*) FROM problems`).Scan(&s.Total)
	_ = d.db.QueryRow(`SELECT COUNT(*) FROM problems WHERE content_md != ''`).Scan(&s.WithBody)
	if fi, err := os.Stat(d.path); err == nil {
		s.DBSize = fi.Size()
	}
	return s, nil
}
