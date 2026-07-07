package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tamnd/mathreflect-cli/mathreflect"
)

func newInfoCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show database and queue statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := buildConfig(gf)

			db, err := mathreflect.OpenDB(cfg.DBPath)
			if err != nil {
				return fmt.Errorf("open db: %w", err)
			}
			defer db.Close()

			stats, err := db.Stats()
			if err != nil {
				return fmt.Errorf("db stats: %w", err)
			}
			fmt.Printf("DB: %s (%.1f MB)\n", cfg.DBPath, float64(stats.DBSize)/1e6)
			fmt.Printf("  problems total:     %d\n", stats.Total)
			fmt.Printf("  problems with body: %d\n", stats.WithBody)

			if _, err := os.Stat(cfg.StatePath); err == nil {
				state, err := mathreflect.OpenState(cfg.StatePath)
				if err != nil {
					return fmt.Errorf("open state: %w", err)
				}
				defer state.Close()
				pending, inProg, done, failed := state.QueueStats()
				fmt.Printf("\nQueue: pending=%d in_progress=%d done=%d failed=%d\n",
					pending, inProg, done, failed)
			}
			return nil
		},
	}
}
