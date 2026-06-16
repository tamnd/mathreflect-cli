package mathreflect

import (
	"os"
	"path/filepath"
	"time"
)

const (
	// ArchiveURL is the archive listing page for all MR issues.
	ArchiveURL = "https://www.awesomemath.org/mathematical-reflections/archives/"

	// SeedFromYear is the earliest year to probe when the archive page is incomplete.
	SeedFromYear = 2006

	// DefaultDelay is the default wait between requests.
	DefaultDelay = 1000 * time.Millisecond

	// DefaultWorkers is the default number of parallel PDF downloaders.
	DefaultWorkers = 2

	// DefaultTimeout is the default HTTP timeout.
	DefaultTimeout = 30 * time.Second
)

// Config holds constructor parameters for the CLI.
type Config struct {
	DataDir   string
	DBPath    string
	StatePath string
	ExportDir string
	Workers   int
	Delay     time.Duration
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults rooted in $HOME/data/mathreflect.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "mathreflect")
	return Config{
		DataDir:   dataDir,
		DBPath:    filepath.Join(dataDir, "mathreflect.db"),
		StatePath: filepath.Join(dataDir, "state.db"),
		ExportDir: filepath.Join(dataDir, "export"),
		Workers:   DefaultWorkers,
		Delay:     DefaultDelay,
		Timeout:   DefaultTimeout,
	}
}
