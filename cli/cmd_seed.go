package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tamnd/mathreflect-cli/mathreflect"
)

func newSeedCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Fetch archive index and enqueue all issue PDF URLs",
		Long: `Fetch the Mathematical Reflections archive page at awesomemath.org,
discover all issue PDF URLs, and add unseen ones to the crawl queue.

Run this once before the first crawl. Re-running is safe: already-visited
issues are skipped.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := buildConfig(gf)

			state, err := mathreflect.OpenState(cfg.StatePath)
			if err != nil {
				return fmt.Errorf("open state: %w", err)
			}
			defer state.Close()

			client := mathreflect.NewClient(cfg.Delay, cfg.Timeout)
			task := &mathreflect.SeedTask{Config: cfg, Client: client, StateDB: state}

			fmt.Fprintln(cmd.OutOrStdout(), "seeding Mathematical Reflections issues...")
			m, err := task.Run(cmd.Context(), func(s *mathreflect.SeedState) {
				mathreflect.PrintSeedProgress(s)
			})
			fmt.Println()
			if err != nil {
				return fmt.Errorf("seed: %w", err)
			}
			fmt.Printf("seed done: %d issues discovered, %d enqueued in %s\n",
				m.Discovered, m.Enqueued, m.Duration.Round(time.Millisecond))
			return nil
		},
	}
}
