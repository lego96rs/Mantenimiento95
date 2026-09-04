package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"mantenimiento/internal/config"
	"mantenimiento/internal/db"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	handler, _ := newTestServerAndDB(t)
	return handler
}

func newTestServerAndDB(t *testing.T) (http.Handler, *db.DB) {
	t.Helper()

	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := database.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv, err := New(config.Config{Addr: ":0", Env: "dev"}, database, log)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	return srv.Handler(), database
}

func get(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestHealthz(t *testing.T) {
	rec := get(t, newTestServer(t), "/healthz")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("body = %s, want to contain ok", rec.Body.String())
	}
}

func TestHomeRenders(t *testing.T) {
	rec := get(t, newTestServer(t), "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Esqueleto inicial listo para extender") {
		t.Fatalf("home page body missing hero heading: %s", rec.Body.String())
	}
}

func TestUnknownRouteIs404(t *testing.T) {
	rec := get(t, newTestServer(t), "/no-existe")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestStaticServed(t *testing.T) {
	rec := get(t, newTestServer(t), "/static/css/app.css")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
