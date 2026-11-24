package main

import (
	"context"
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"9router/proxy/internal/app"
	"9router/proxy/internal/db"
	"9router/proxy/internal/handlers/chat"
	"9router/proxy/internal/proxy/oauth"
	"9router/proxy/internal/schedule"
)

// maintainReport summarizes what a one-shot maintenance pass found and did.
type maintainReport struct {
	OAuthDue     int
	AutoPingDue  int
	AutoPingErrs int
}

func maintainCommand() *cli.Command {
	return &cli.Command{
		Name:  "maintain",
		Usage: "Run one-shot OAuth token refresh and due quota auto-pings only when needed, then exit",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "report due OAuth refreshes and auto-pings without making network requests",
			},
			&cli.BoolFlag{
				Name:  "quiet",
				Usage: "suppress output when nothing is due",
			},
		},
		Action: func(cCtx *cli.Context) error {
			var repo *db.Repo
			dbApp := fx.New(
				app.ConfigModule,
				app.DatabaseModule,
				fx.NopLogger,
				fx.Populate(&repo),
			)
			startCtx, startCancel := context.WithTimeout(cCtx.Context, 15*time.Second)
			defer startCancel()
			if err := dbApp.Start(startCtx); err != nil {
				return fmt.Errorf("maintain: open db: %w", err)
			}
			defer func() {
				stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer stopCancel()
				_ = dbApp.Stop(stopCtx)
			}()

			ctx, cancel := context.WithTimeout(cCtx.Context, 2*time.Minute)
			defer cancel()

			report, err := runMaintain(ctx, repo, app.NewCLIParams(cCtx), cCtx.Bool("dry-run"), time.Now())
			if err != nil {
				return err
			}

			if cCtx.Bool("quiet") && report.OAuthDue == 0 && report.AutoPingDue == 0 {
				return nil
			}
			mode := "executed"
			if cCtx.Bool("dry-run") {
				mode = "dry-run"
			}
			fmt.Printf("maintain (%s): oauth_due=%d autoping_due=%d autoping_errors=%d\n",
				mode, report.OAuthDue, report.AutoPingDue, report.AutoPingErrs)
			return nil
		},
	}
}

func runMaintain(ctx context.Context, repo *db.Repo, cliParams app.CLIParams, dryRun bool, now time.Time) (maintainReport, error) {
	var report maintainReport

	conns, err := repo.GetProviderConnections("", true)
	if err != nil {
		return report, fmt.Errorf("maintain: load connections: %w", err)
	}
	dueOAuth := oauth.SelectConnectionsNeedingRefresh(conns, now)
	report.OAuthDue = len(dueOAuth)

	source := schedule.ConfigSourceFromSettings(func() (map[string]any, error) {
		return repo.GetSettingsRaw()
	})
	report.AutoPingDue = countDueAutoPings(source, now)

	// Fast exit: if nothing is due (e.g. static API keys only), do zero network work.
	if report.OAuthDue == 0 && report.AutoPingDue == 0 {
		return report, nil
	}
	if dryRun {
		return report, nil
	}

	if report.OAuthDue > 0 {
		oauth.RunRefreshOnce(ctx, repo)
	}
	if report.AutoPingDue > 0 {
		report.AutoPingErrs = runDueAutoPings(ctx, cliParams, source)
	}
	return report, nil
}

func countDueAutoPings(source schedule.ConfigSource, now time.Time) int {
	due := 0
	for _, provider := range schedule.Providers {
		cfg := source(provider)
		for _, sched := range cfg.Schedules {
			if sched != nil && sched.DueSlot(now) != nil {
				due++
			}
		}
	}
	return due
}

func runDueAutoPings(ctx context.Context, cliParams app.CLIParams, source schedule.ConfigSource) int {
	var chatH *chat.ChatHandler
	fxApp := fx.New(
		app.ConfigModule,
		app.DatabaseModule,
		app.HandlersModule,
		fx.Replace(cliParams),
		fx.NopLogger,
		fx.Populate(&chatH),
	)
	startCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := fxApp.Start(startCtx); err != nil {
		return 1
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		_ = fxApp.Stop(stopCtx)
	}()

	sched := schedule.NewScheduler(source, schedule.PingFuncFromChat(chatH))
	errs := sched.Tick(ctx)
	return len(errs)
}
