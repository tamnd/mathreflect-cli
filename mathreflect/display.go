package mathreflect

import "fmt"

// PrintSeedProgress prints a single-line seed progress update.
func PrintSeedProgress(s *SeedState) {
	fmt.Printf("\r  discovered=%-4d  enqueued=%-4d    ", s.Discovered, s.Enqueued)
}

// PrintCrawlProgress prints a single-line crawl progress update.
func PrintCrawlProgress(s *CrawlState) {
	fmt.Printf("\r  done=%-6d  pending=%-6d  failed=%-4d  exported=%-6d  rps=%.2f   ",
		s.Done, s.Pending, s.Failed, s.Exported, s.RPS)
}

// PrintExportProgress prints a single-line export progress update.
func PrintExportProgress(s *ExportState) {
	fmt.Printf("\r  written=%-6d  current=%-40s", s.Written, s.Current)
}
