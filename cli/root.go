// Package cli assembles the mathreflect command tree.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tamnd/mathreflect-cli/mathreflect"
)

// Build metadata, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// exit codes.
const (
	exitError  = 1
	exitUsage  = 2
	exitNoData = 3
	exitNet    = 5
)

// globalFlags holds the parsed global flags shared across subcommands.
type globalFlags struct {
	DBPath    string
	StatePath string
	ExportDir string
	DelayMs   int
	TimeoutS  int
	Workers   int
}

// Root builds and returns the root cobra command for the mathreflect binary.
func Root() *cobra.Command {
	gf := &globalFlags{}
	cfg := mathreflect.DefaultConfig()

	root := &cobra.Command{
		Use:   "mathreflect",
		Short: "Competition math problems from Mathematical Reflections (awesomemath.org)",
		Long: `mathreflect fetches competition math problems from the Mathematical Reflections
journal published by AwesomeMath. Problems are extracted from PDFs and stored
locally as structured Markdown files.

Pipeline:
  1. seed   -- fetch archive index, enqueue all issue PDF URLs
  2. crawl  -- download each PDF, extract problems, store in DB
  3. export -- write all problems to markdown files`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&gf.DBPath, "db", cfg.DBPath, "Path to problems SQLite database")
	root.PersistentFlags().StringVar(&gf.StatePath, "state", cfg.StatePath, "Path to crawl-queue SQLite database")
	root.PersistentFlags().StringVar(&gf.ExportDir, "export-dir", cfg.ExportDir, "Markdown export directory")
	root.PersistentFlags().IntVar(&gf.DelayMs, "delay", int(cfg.Delay.Milliseconds()), "Delay between requests (ms)")
	root.PersistentFlags().IntVar(&gf.TimeoutS, "timeout", int(cfg.Timeout.Seconds()), "HTTP timeout (seconds)")
	root.PersistentFlags().IntVar(&gf.Workers, "workers", cfg.Workers, "Parallel PDF download workers")

	root.AddCommand(newSeedCmd(gf))
	root.AddCommand(newCrawlCmd(gf))
	root.AddCommand(newExportCmd(gf))
	root.AddCommand(newInfoCmd(gf))
	root.AddCommand(newVersionCmd())

	return root
}

// run is the entry point called from main.
func run(root *cobra.Command, args []string) int {
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitError
	}
	return 0
}
