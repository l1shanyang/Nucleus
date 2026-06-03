package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	chimw "github.com/go-chi/chi/v5/middleware"

	"nucleus/internal/http/middleware"
)

func TestRequestLogWritesRequestIDHeaderAndLogsRequest(t *testing.T) {
	var buf bytes.Buffer
	restoreDefaultLogger(t, &buf)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	})
	h := chimw.RequestID(chimw.RealIP(middleware.RequestLog(next)))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes?limit=1", http.NoBody)
	req.Header.Set("X-Request-ID", "test-request-id")
	req.Header.Set("X-Real-IP", "203.0.113.10")
	req.Header.Set("User-Agent", "nucleus-test")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if got := w.Header().Get("X-Request-ID"); got != "test-request-id" {
		t.Fatalf("X-Request-ID = %q, want %q", got, "test-request-id")
	}

	log := decodeLogRecord(t, &buf)
	assertLogValue(t, log, "msg", "http request")
	assertLogValue(t, log, "request_id", "test-request-id")
	assertLogValue(t, log, "method", http.MethodPost)
	assertLogValue(t, log, "path", "/api/v1/notes")
	assertLogValue(t, log, "status", float64(http.StatusCreated))
	assertLogValue(t, log, "remote_ip", "203.0.113.10")
	assertLogValue(t, log, "user_agent", "nucleus-test")
	assertLogValue(t, log, "bytes", float64(len("ok")))
	if _, ok := log["latency"]; !ok {
		t.Fatal("latency field missing")
	}
}

func TestRequestLogDefaultsStatusToOK(t *testing.T) {
	var buf bytes.Buffer
	restoreDefaultLogger(t, &buf)

	h := chimw.RequestID(middleware.RequestLog(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))
	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if got := w.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("X-Request-ID is empty")
	}

	log := decodeLogRecord(t, &buf)
	assertLogValue(t, log, "status", float64(http.StatusOK))
	assertLogValue(t, log, "bytes", float64(0))
}

func restoreDefaultLogger(t *testing.T, buf *bytes.Buffer) {
	t.Helper()

	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	t.Cleanup(func() {
		slog.SetDefault(original)
	})
}

func decodeLogRecord(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to decode log: %v; log=%s", err, buf.String())
	}
	return record
}

func assertLogValue(t *testing.T, record map[string]any, key string, want any) {
	t.Helper()

	if got := record[key]; got != want {
		t.Fatalf("%s = %#v, want %#v", key, got, want)
	}
}
