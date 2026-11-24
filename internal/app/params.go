package app

import (
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

// CLIParams holds command-line flags and environment overrides.
type CLIParams struct {
	RTK              bool
	RTKSet           bool
	Caveman          bool
	CavemanSet       bool
	Ponytail         bool
	PonytailSet      bool
	ADHD             bool
	ADHDSet          bool
	AutoUpdate       bool
	NoInjectionGuard bool
	IdleTimeout      time.Duration
}

// NewCLIParams creates CLIParams from a urfave/cli Context.
func NewCLIParams(cCtx *cli.Context) CLIParams {
	if cCtx == nil {
		return DefaultCLIParams()
	}
	idle := cCtx.Duration("idle-timeout")
	if idle <= 0 {
		idle = ParseIdleTimeoutEnv()
	}
	return CLIParams{
		RTK:              cCtx.Bool("rtk"),
		RTKSet:           cCtx.IsSet("rtk"),
		Caveman:          cCtx.Bool("caveman"),
		CavemanSet:       cCtx.IsSet("caveman"),
		Ponytail:         cCtx.Bool("ponytail"),
		PonytailSet:      cCtx.IsSet("ponytail"),
		ADHD:             cCtx.Bool("adhd"),
		ADHDSet:          cCtx.IsSet("adhd"),
		AutoUpdate:       cCtx.Bool("auto-update"),
		NoInjectionGuard: cCtx.Bool("no-injection-guard"),
		IdleTimeout:      idle,
	}
}

// DefaultCLIParams creates CLIParams using default environment variables.
func DefaultCLIParams() CLIParams {
	return CLIParams{
		RTK:              os.Getenv("RTK_ENABLED") != "false",
		RTKSet:           os.Getenv("RTK_ENABLED") != "",
		Caveman:          os.Getenv("CAVEMAN_ENABLED") == "true",
		CavemanSet:       os.Getenv("CAVEMAN_ENABLED") != "",
		Ponytail:         os.Getenv("PONYTAIL_ENABLED") == "true",
		PonytailSet:      os.Getenv("PONYTAIL_ENABLED") != "",
		ADHD:             os.Getenv("ADHD_ENABLED") == "true",
		ADHDSet:          os.Getenv("ADHD_ENABLED") != "",
		AutoUpdate:       os.Getenv("AUTO_UPDATE") == "true",
		NoInjectionGuard: os.Getenv("INJECTION_GUARD_DISABLED") == "true",
		IdleTimeout:      ParseIdleTimeoutEnv(),
	}
}

// ParseIdleTimeoutEnv parses IDLE_TIMEOUT as a Go duration (e.g. "15m").
// Empty or invalid values return 0 (disabled).
func ParseIdleTimeoutEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("IDLE_TIMEOUT"))
	if raw == "" || raw == "0" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0
	}
	return d
}
