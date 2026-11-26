package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"9router/proxy/internal/app"
	"9router/proxy/internal/db"
)

func openTestRepo(t *testing.T) *db.Repo {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")
	conn, err := db.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := db.EnsureCoreSchema(conn); err != nil {
		t.Fatalf("EnsureCoreSchema: %v", err)
	}
	return db.NewRepo(conn)
}

func TestRunMaintain_StaticAPIKeysOnlyDoesNothing(t *testing.T) {
	repo := openTestRepo(t)

	if err := repo.CreateProviderConnectionFull(context.Background(), "conn-apikey-1", "openai", "apikey", "test-apikey", nil, `{"apiKey":"sk-test-123"}`); err != nil {
		t.Fatalf("CreateProviderConnectionFull: %v", err)
	}

	report, err := runMaintain(context.Background(), repo, app.DefaultCLIParams(), false, time.Now())
	if err != nil {
		t.Fatalf("runMaintain: %v", err)
	}
	if report.OAuthDue != 0 || report.AutoPingDue != 0 || report.AutoPingErrs != 0 {
		t.Fatalf("expected zero due work for static API key connection, got %+v", report)
	}
}

func TestRunMaintain_DryRunDetectsExpiringOAuth(t *testing.T) {
	repo := openTestRepo(t)

	now := time.Now().UTC()
	expSoon := now.Add(10 * time.Minute).Format(time.RFC3339)
	data := `{"accessToken":"at-1","refreshToken":"rt-1","expiresAt":"` + expSoon + `"}`

	if err := repo.CreateProviderConnectionFull(context.Background(), "conn-oauth-1", "codex", "oauth", "test-oauth", nil, data); err != nil {
		t.Fatalf("CreateProviderConnectionFull: %v", err)
	}

	report, err := runMaintain(context.Background(), repo, app.DefaultCLIParams(), true, now)
	if err != nil {
		t.Fatalf("runMaintain: %v", err)
	}
	if report.OAuthDue != 1 {
		t.Fatalf("expected OAuthDue=1, got %+v", report)
	}
	if report.AutoPingDue != 0 {
		t.Fatalf("expected AutoPingDue=0, got %+v", report)
	}
}
