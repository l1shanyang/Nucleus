package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	chimw "github.com/go-chi/chi/v5/middleware"

	"nucleus/internal/apperror"
	"nucleus/internal/http/handler"
)

func TestWrapHandler_LogsHandlerErrorWithRequestContext(t *testing.T) {
	var buf bytes.Buffer
	restoreHandlerLogger(t, &buf)

	dbErr := errors.New(`ERROR: relation "workspaces" does not exist`)
	h := chimw.RequestID(handler.WrapHandler(func(http.ResponseWriter, *http.Request) error {
		return apperror.Internal("failed to create workspace", dbErr)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", http.NoBody)
	req.Header.Set("X-Request-ID", "test-request-id")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp handler.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error.Message == dbErr.Error() {
		t.Fatalf("response leaked database error: %q", resp.Error.Message)
	}

	log := decodeHandlerLogRecord(t, &buf)
	assertHandlerLogValue(t, log, "level", "ERROR")
	assertHandlerLogValue(t, log, "msg", "http handler error")
	assertHandlerLogValue(t, log, "request_id", "test-request-id")
	assertHandlerLogValue(t, log, "method", http.MethodPost)
	assertHandlerLogValue(t, log, "path", "/api/v1/workspaces")
	assertHandlerLogValue(t, log, "status", float64(http.StatusInternalServerError))
	assertHandlerLogValue(t, log, "code", "INTERNAL")

	gotError, ok := log["error"].(string)
	if !ok {
		t.Fatalf("error log field is %T, want string", log["error"])
	}
	if gotError != `failed to create workspace: ERROR: relation "workspaces" does not exist` {
		t.Fatalf("error = %q, want wrapped database error", gotError)
	}
}

func restoreHandlerLogger(t *testing.T, buf *bytes.Buffer) {
	t.Helper()

	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	t.Cleanup(func() {
		slog.SetDefault(original)
	})
}

func decodeHandlerLogRecord(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to decode log: %v; log=%s", err, buf.String())
	}
	return record
}

func assertHandlerLogValue(t *testing.T, record map[string]any, key string, want any) {
	t.Helper()

	if got := record[key]; got != want {
		t.Fatalf("%s = %#v, want %#v", key, got, want)
	}
}
