package mathreflect

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExportTask renders the DB to markdown files.
type ExportTask struct {
	Config Config
	DB     *DB
}

// Run executes the export task.
func (t *ExportTask) Run(ctx context.Context, emit func(*ExportState)) (ExportMetric, error) {
	start := time.Now()
	state := &ExportState{}

	base := t.Config.ExportDir
	if err := os.MkdirAll(filepath.Join(base, "problems"), 0o755); err != nil {
		return ExportMetric{}, fmt.Errorf("mkdir problems: %w", err)
	}

	problems, err := t.DB.ListAll()
	if err != nil {
		return ExportMetric{}, fmt.Errorf("list problems: %w", err)
	}

	written := 0

	state.Current = "README.md"
	emit(state)
	if err := writeFileAt(filepath.Join(base, "README.md"), renderReadme(problems)); err == nil {
		written++
	}

	for _, p := range problems {
		if ctx.Err() != nil {
			break
		}
		state.Current = fmt.Sprintf("problems/%s.md", p.ID)
		emit(state)
		if err := writeProblemFile(base, p); err == nil {
			written++
			state.Written = written
		}
	}

	return ExportMetric{Files: written, Duration: time.Since(start)}, nil
}

func writeProblemFile(baseDir string, p Problem) error {
	return writeFileAt(
		filepath.Join(baseDir, "problems", p.ID+".md"),
		renderProblemMarkdown(p),
	)
}

func renderReadme(problems []Problem) string {
	var b strings.Builder
	b.WriteString("# Mathematical Reflections -- Competition Math Problems\n\n")
	fmt.Fprintf(&b, "Total: %d problems from [awesomemath.org](https://www.awesomemath.org/mathematical-reflections/).\n\n", len(problems))
	b.WriteString("| ID | Issue | Section | # |\n")
	b.WriteString("|----|-------|---------|---|\n")
	for _, p := range problems {
		link := fmt.Sprintf("[%s](problems/%s.md)", p.ID, p.ID)
		fmt.Fprintf(&b, "| %s | %d-%d | %s | %d |\n",
			link, p.IssueYear, p.IssueNum, p.Section, p.ProblemNum)
	}
	return b.String()
}

func renderProblemMarkdown(p Problem) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "id: %s\n", p.ID)
	fmt.Fprintf(&b, "issue_year: %d\n", p.IssueYear)
	fmt.Fprintf(&b, "issue_num: %d\n", p.IssueNum)
	fmt.Fprintf(&b, "section: %s\n", p.Section)
	fmt.Fprintf(&b, "problem_num: %d\n", p.ProblemNum)
	fmt.Fprintf(&b, "url: %s\n", p.URL)
	fmt.Fprintf(&b, "fetched_at: %s\n", p.FetchedAt.UTC().Format(time.RFC3339))
	b.WriteString("---\n\n")

	fmt.Fprintf(&b, "# Mathematical Reflections %d Issue %d -- %s Problem %d\n\n",
		p.IssueYear, p.IssueNum, p.Section, p.ProblemNum)

	if p.ContentMD != "" {
		b.WriteString(p.ContentMD)
		b.WriteString("\n")
	}
	return b.String()
}

func writeFileAt(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
