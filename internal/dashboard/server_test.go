package dashboard

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mochaka/devproxy/internal/admin"
	"github.com/mochaka/devproxy/internal/adminapi"
)

func TestDashboardRootRendersHealthRoutesConflictsAndSessionErrors(t *testing.T) {
	t.Parallel()

	client := &stubClient{
		status: admin.StatusView{SnapshotVersion: "snap-1", ActiveRoutes: 1, Conflicts: 1},
		routes: []admin.RouteView{{Hostname: "api.acme.test", OpenURL: "https://api.acme.test", UpstreamScheme: "https", UpstreamHost: "127.0.0.1", UpstreamPort: 8443}},
		logs: []admin.LogEvent{
		{Timestamp: time.Now().UTC(), Type: "conflict", Hostname: "api.acme.test", Message: "route conflict detected"},
		{Timestamp: time.Now().UTC(), Type: "error", Hostname: "", Message: "refresh failed"},
	},
	}

	srv := NewServer(Config{ListenAddress: "127.0.0.1:45831", Client: client})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /, got %d", w.Code)
	}
	body := w.Body.String()
	assertContains(t, body, "Refresh Routes")
	assertContains(t, body, "No Active Routes")
	assertContains(t, body, "DevProxy can’t reach the daemon right now. Ensure the daemon is running, then select Run Doctor for repair guidance.")
	assertContains(t, body, "api.acme.test")
	assertContains(t, body, "route conflict detected")
	assertContains(t, body, "refresh failed")
	assertContains(t, body, `href="https://api.acme.test"`)
}

func TestDashboardLogsRendersCurrentSessionData(t *testing.T) {
	t.Parallel()

	client := &stubClient{
		logs: []admin.LogEvent{
			{Timestamp: time.Now().UTC(), Type: "warning", Hostname: "api.acme.test", Message: "session warning"},
			{Timestamp: time.Now().UTC(), Type: "error", Hostname: "", Message: "doctor failed"},
		},
	}

	srv := NewServer(Config{ListenAddress: "127.0.0.1:45831", Client: client})
	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /logs, got %d", w.Code)
	}
	body := w.Body.String()
	assertContains(t, body, "Current Session Logs")
	assertContains(t, body, "Current Session Errors")
	assertContains(t, body, "session warning")
	assertContains(t, body, "doctor failed")
}

func TestDashboardRefreshPostsAndRedirectsWithFlashMessage(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		client := &stubClient{}
		srv := NewServer(Config{ListenAddress: "127.0.0.1:45831", Client: client})

		req := httptest.NewRequest(http.MethodPost, "/actions/refresh", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusSeeOther {
			t.Fatalf("expected 303 for successful refresh action, got %d", w.Code)
		}
		if client.refreshReason == "" {
			t.Fatalf("expected refresh to be invoked with a reason")
		}
		location := w.Header().Get("Location")
		u, err := url.Parse(location)
		if err != nil {
			t.Fatalf("parse redirect location: %v", err)
		}
		assertContains(t, u.RawQuery, "flash=Routes+refreshed")
	})

	t.Run("failure", func(t *testing.T) {
		client := &stubClient{refreshErr: errors.New("daemon unavailable")}
		srv := NewServer(Config{ListenAddress: "127.0.0.1:45831", Client: client})

		req := httptest.NewRequest(http.MethodPost, "/actions/refresh", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusSeeOther {
			t.Fatalf("expected 303 for failed refresh action, got %d", w.Code)
		}
		location := w.Header().Get("Location")
		u, err := url.Parse(location)
		if err != nil {
			t.Fatalf("parse redirect location: %v", err)
		}
		assertContains(t, u.RawQuery, "flash=Refresh+failed")
	})
}

func TestDashboardRefreshRequiresPost(t *testing.T) {
	t.Parallel()

	client := &stubClient{}
	srv := NewServer(Config{ListenAddress: "127.0.0.1:45831", Client: client})
	req := httptest.NewRequest(http.MethodGet, "/actions/refresh", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET /actions/refresh, got %d", w.Code)
	}
}

type stubClient struct {
	status       admin.StatusView
	routes       []admin.RouteView
	logs         []admin.LogEvent
	doctor       admin.DoctorView
	refreshErr   error
	refreshReason string
}

func (s *stubClient) Status(context.Context) (admin.StatusView, error) { return s.status, nil }
func (s *stubClient) Routes(context.Context) ([]admin.RouteView, error) { return s.routes, nil }
func (s *stubClient) Logs(context.Context) ([]admin.LogEvent, error) { return s.logs, nil }
func (s *stubClient) Doctor(context.Context) (admin.DoctorView, error) { return s.doctor, nil }
func (s *stubClient) Refresh(_ context.Context, reason string) (adminapi.RefreshResponse, error) {
	s.refreshReason = reason
	if s.refreshErr != nil {
		return adminapi.RefreshResponse{}, s.refreshErr
	}
	return adminapi.RefreshResponse{Accepted: true, Refreshed: true, At: time.Now().UTC()}, nil
}
func assertContains(t *testing.T, body, expected string) {
	t.Helper()
	if !strings.Contains(body, expected) {
		t.Fatalf("expected body to contain %q\nbody: %s", expected, body)
	}
}
