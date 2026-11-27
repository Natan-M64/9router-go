package db

import (
	"context"
	json "encoding/json/v2"
	"testing"
)

func TestUpdateProviderConnectionProjectIDPreservesOtherData(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepo(database)
	priority := 1
	const initial = `{"accessToken":"new-access","refreshToken":"refresh","other":"keep"}`
	if err := repo.CreateProviderConnectionFull(context.Background(), "ag-1", "antigravity", "oauth", "account", &priority, initial); err != nil {
		t.Fatalf("create provider connection: %v", err)
	}

	if err := repo.UpdateProviderConnectionProjectID(context.Background(), "ag-1", "project-1"); err != nil {
		t.Fatalf("UpdateProviderConnectionProjectID: %v", err)
	}
	got, err := repo.GetProviderConnectionByID("ag-1")
	if err != nil {
		t.Fatalf("GetProviderConnectionByID: %v", err)
	}
	var data map[string]string
	if err := json.Unmarshal([]byte(got.Data), &data); err != nil {
		t.Fatalf("decode connection data: %v", err)
	}
	for field, want := range map[string]string{
		"accessToken":  "new-access",
		"refreshToken": "refresh",
		"other":        "keep",
		"projectId":    "project-1",
	} {
		if data[field] != want {
			t.Errorf("data[%q] = %q, want %q", field, data[field], want)
		}
	}
}
