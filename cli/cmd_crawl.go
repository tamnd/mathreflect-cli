package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tamnd/mathreflect-cli/mathreflect"
)

func newCrawlCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "crawl",
		Short: "Download issue PDFs and extract problems into the database",
		Long: `Download each queued issue PDF from awesomemath.org, extract problem
statements using pdftotext, and store the results in the local SQLite database.

Requires pdftotext from poppler-utils:
  macOS:  brew install poppler
  Linux:  apt install poppler-utils

Run 'mathreflect seed' first to populate the queue.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := buildConfig(gf)

			db, err := mathreflect.OpenDB(cfg.DBPath)
			if err != nil {
				return fmt.Errorf("open db: %w", err)
			}
			defer db.Close()

			state, err := mathreflect.OpenState(cfg.StatePath)
			if err != nil {
				return fmt.Errorf("open state: %w", err)
			}
			defer state.Close()

			if state.PendingCount() == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "queue is empty -- run 'mathreflect seed' first")
				return nil
			}

			client := mathreflect.NewClient(cfg.Delay, cfg.Timeout)
			task := &mathreflect.CrawlTask{Config: cfg, Client: client, DB: db, StateDB: state}

			fmt.Fprintln(cmd.OutOrStdout(), "crawling Mathematical Reflections issues...")
			m, err := task.Run(cmd.Context(), func(s *mathreflect.CrawlState) {
				mathreflect.PrintCrawlProgress(s)
			})
			fmt.Println()
			if err != nil {
				if errors.Is(err, mathreflect.ErrPDFtotextNotFound) {
					return err
				}
				return fmt.Errorf("crawl: %w", err)
			}
			fmt.Printf("crawl done: %d fetched, %d failed, %d problems exported in %s\n",
				m.Done, m.Failed, m.Exported, m.Duration.Round(time.Second))
			return nil
		},
	}
}
