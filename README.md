# mathreflect

Competition math problems from Mathematical Reflections (awesomemath.org)

`mathreflect` is a single pure-Go binary that fetches and locally archives
competition math problems from the Mathematical Reflections journal published
by AwesomeMath. Problems are extracted from PDFs and stored in a local SQLite
database, then exported to structured Markdown files.

No API key required.

## Requirements

`pdftotext` from the `poppler-utils` package is required to extract text from
issue PDFs:

```bash
# macOS
brew install poppler

# Linux (Debian/Ubuntu)
apt install poppler-utils
```

## Install

```bash
go install github.com/tamnd/mathreflect-cli/cmd/mathreflect@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/mathreflect-cli/releases).

## Quick start

```bash
# Step 1: fetch the archive index and enqueue all issue URLs
mathreflect seed

# Step 2: download PDFs and extract problems (requires pdftotext)
mathreflect crawl

# Step 3: write all problems to markdown files
mathreflect export

# Check progress at any time
mathreflect info
```

## Commands

### `mathreflect seed`

Fetch the archive index at `awesomemath.org/mathematical-reflections/archives/`,
discover all issue PDF URLs, and enqueue any not yet visited.

```
mathreflect seed [--db PATH] [--state PATH] [--delay MS] [--timeout S]
```

### `mathreflect crawl`

Download each queued issue PDF, extract problem statements using `pdftotext`,
and store the results in the local SQLite database.

```
mathreflect crawl [--db PATH] [--state PATH] [--delay MS] [--timeout S]
                  [--workers N] [--export-dir PATH]
```

### `mathreflect export`

Read all problems from the database and write each one to a Markdown file.

```
mathreflect export [--db PATH] [--export-dir PATH]
```

Output structure:
```
$HOME/data/mathreflect/export/
  README.md               -- index table of all problems
  problems/
    2006-1-J-1.md
    2006-1-J-2.md
    ...
    2026-3-O-5.md
```

### `mathreflect info`

Show database statistics and queue depth.

```
mathreflect info [--db PATH] [--state PATH]
```

Example output:
```
DB: /Users/alice/data/mathreflect/mathreflect.db (4.2 MB)
  problems total:     2841
  problems with body: 2841

Queue: pending=0 in_progress=0 done=108 failed=0
```

## Problem ID format

Each problem is identified by: `{year}-{issue}-{section}-{number}`

Example: `2026-3-O-5` = Year 2026, Issue 3, Olympiad section, Problem 5.

Sections:
- `J` = Junior
- `S` = Senior
- `O` = Olympiad
- `U` = Undergraduate
- `I` = Individual

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--db` | `$HOME/data/mathreflect/mathreflect.db` | Problems SQLite database |
| `--state` | `$HOME/data/mathreflect/state.db` | Crawl-queue SQLite database |
| `--export-dir` | `$HOME/data/mathreflect/export` | Markdown output directory |
| `--delay` | `1000` | Delay between requests (ms) |
| `--timeout` | `30` | HTTP timeout (seconds) |
| `--workers` | `2` | Parallel PDF download workers |

## Development

```
cmd/mathreflect/   main entry point
cli/               cobra command tree
mathreflect/       library: client, DB, state, PDF parser, tasks
```

```bash
make build      # ./bin/mathreflect
make test       # go test ./...
make vet        # go vet ./...
```

## License

Apache-2.0. See [LICENSE](LICENSE).
