package cli

import (
	"time"

	"github.com/tamnd/mathreflect-cli/mathreflect"
)

// buildConfig constructs a mathreflect.Config from global flags.
func buildConfig(gf *globalFlags) mathreflect.Config {
	cfg := mathreflect.DefaultConfig()
	if gf.DBPath != "" {
		cfg.DBPath = gf.DBPath
	}
	if gf.StatePath != "" {
		cfg.StatePath = gf.StatePath
	}
	if gf.ExportDir != "" {
		cfg.ExportDir = gf.ExportDir
	}
	if gf.DelayMs > 0 {
		cfg.Delay = time.Duration(gf.DelayMs) * time.Millisecond
	}
	if gf.TimeoutS > 0 {
		cfg.Timeout = time.Duration(gf.TimeoutS) * time.Second
	}
	if gf.Workers > 0 {
		cfg.Workers = gf.Workers
	}
	return cfg
}
