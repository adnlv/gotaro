package web

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/adnlv/gotaro/internal/app"
)

func TestHandler_rootRedirectsToTasks(t *testing.T) {
	t.Parallel()
	srv, err := NewServer(slog.Default(), &app.AuthService{}, &app.TaskService{}, false)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status: %d", rr.Code)
	}
	if g, w := rr.Header().Get("Location"), "/tasks"; g != w {
		t.Fatalf("Location: got %q want %q", g, w)
	}
}

func TestHandler_tasksRequiresLogin(t *testing.T) {
	t.Parallel()
	srv, err := NewServer(slog.Default(), &app.AuthService{}, &app.TaskService{}, false)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/tasks", nil))
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status: %d", rr.Code)
	}
	if g, w := rr.Header().Get("Location"), "/login"; g != w {
		t.Fatalf("Location: got %q want %q", g, w)
	}
}

func TestTaskExportPath_preservesFiltersAndView(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/tasks/completed?status=todo&q=hello&offset=25", nil)
	got := taskExportPath(r, true, false)
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != "/tasks/export.csv" {
		t.Fatalf("path: %q", u.Path)
	}
	q := u.Query()
	if q.Get("view") != "completed" || q.Get("q") != "hello" || q.Get("status") != "todo" {
		t.Fatalf("query: %v", q)
	}
	if q.Get("offset") != "" {
		t.Fatalf("expected offset stripped, got %q", q.Get("offset"))
	}
}
