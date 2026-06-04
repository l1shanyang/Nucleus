package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nucleus/internal/http/handler"
	"nucleus/internal/version"
)

func TestVersionHandler_Show(t *testing.T) {
	h := handler.NewVersionHandler()
	req := httptest.NewRequest(http.MethodGet, "/version", http.NoBody)
	w := httptest.NewRecorder()

	h.Show(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp version.Info
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Version == "" {
		t.Fatal("version is empty")
	}
	if resp.Commit == "" {
		t.Fatal("commit is empty")
	}
	if resp.BuildTime == "" {
		t.Fatal("build_time is empty")
	}
	if !strings.HasPrefix(resp.GoVersion, "go") {
		t.Fatalf("go_version = %q, want go*", resp.GoVersion)
	}
}
