package db

import (
	"encoding/json"
	"testing"
)

// The ban disables the connection, keeps the fixed reason where the provider
// card reads errors, and leaves other connections alone.
func TestDisableAntigravityAccountBan_DisablesAndKeepsTheReason(t *testing.T) {
	repo, cleanup := setupHealthConnTestDB(t)
	defer cleanup()
	db := repo.RawDB()
	for _, id := range []string{"ag-banned", "ag-healthy"} {
		if _, err := db.Exec(`INSERT INTO providerConnections (id, provider, authType, name, isActive, data, createdAt, updatedAt)
			VALUES (?, 'antigravity', 'oauth', ?, 1, '{}', datetime('now'), datetime('now'))`, id, id); err != nil {
			t.Fatalf("insert %s: %v", id, err)
		}
	}

	const reason = "Desativada automaticamente: teste"
	if err := repo.DisableAntigravityAccountBan("ag-banned", reason); err != nil {
		t.Fatalf("DisableAntigravityAccountBan: %v", err)
	}

	var active int
	if err := db.QueryRow(`SELECT isActive FROM providerConnections WHERE id = 'ag-banned'`).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active != 0 {
		t.Errorf("isActive = %d after the ban, want 0", active)
	}

	var data string
	if err := db.QueryRow(`SELECT data FROM providerConnections WHERE id = 'ag-banned'`).Scan(&data); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Status                string `json:"status"`
		AccountDisabledReason string `json:"accountDisabledReason"`
		LastError             struct {
			Message    string `json:"message"`
			AccountBan bool   `json:"accountBan"`
		} `json:"lastError"`
	}
	if err := json.Unmarshal([]byte(data), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" {
		t.Errorf("status = %q, want error", got.Status)
	}
	if got.AccountDisabledReason != reason || got.LastError.Message != reason {
		t.Errorf("reason not kept: accountDisabledReason=%q lastError.message=%q", got.AccountDisabledReason, got.LastError.Message)
	}
	if !got.LastError.AccountBan {
		t.Error("lastError.accountBan = false, want true")
	}

	var healthy int
	if err := db.QueryRow(`SELECT isActive FROM providerConnections WHERE id = 'ag-healthy'`).Scan(&healthy); err != nil {
		t.Fatal(err)
	}
	if healthy != 1 {
		t.Errorf("an unrelated connection was switched off: isActive = %d", healthy)
	}
}

func TestDisableAntigravityAccountBan_EmptyIDIsANoOp(t *testing.T) {
	repo, cleanup := setupHealthConnTestDB(t)
	defer cleanup()
	if err := repo.DisableAntigravityAccountBan("", "x"); err != nil {
		t.Errorf("empty id returned %v, want nil", err)
	}
}
