package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tamnd/mathreflect-cli/mathreflect"
)

func newExportCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Write all problems from the database to Markdown files",
		Long: `Read all problems from the local SQLite database and write each one
to a Markdown file under the export directory.

Output structure:
  $export-dir/
    README.md           -- index table of all problems
    problems/
      2006-1-J-1.md
      2006-1-J-2.md
      ...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := buildConfig(gf)

			db, err := mathreflect.OpenDB(cfg.DBPath)
			if err != nil {
				return fmt.Errorf("open db: %w", err)
			}
			defer db.Close()

			task := &mathreflect.ExportTask{Config: cfg, DB: db}

			fmt.Fprintf(cmd.OutOrStdout(), "exporting to %s...\n", cfg.ExportDir)
			m, err := task.Run(cmd.Context(), func(s *mathreflect.ExportState) {
				mathreflect.PrintExportProgress(s)
			})
			fmt.Println()
			if err != nil {
				return fmt.Errorf("export: %w", err)
			}
			fmt.Printf("export done: %d files in %s\n", m.Files, m.Duration.Round(time.Millisecond))
			return nil
		},
	}
}
