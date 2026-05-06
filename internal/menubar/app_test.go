package menubar

import (
	"context"
	"errors"
	"testing"

	"github.com/mochaka/devproxy/internal/admin"
	"github.com/mochaka/devproxy/internal/adminapi"
)

func TestMenubarBuildStateFromAdminData(t *testing.T) {
	status := admin.StatusView{ActiveRoutes: 2, Paused: true}
	routes := []admin.RouteView{
		{Hostname: "api.acme.test", OpenURL: "https://api.acme.test"},
		{Hostname: "acme.test", OpenURL: "http://acme.test"},
	}

	state := buildMenuState(status, routes, nil)

	if state.HealthLine != "Daemon: healthy" {
		t.Fatalf("expected healthy status line, got %q", state.HealthLine)
	}
	if state.PauseLine != "Routing: paused" {
		t.Fatalf("expected paused routing line, got %q", state.PauseLine)
	}
	if state.ActiveRoutesLine != "Active routes: 2" {
		t.Fatalf("expected active route count line, got %q", state.ActiveRoutesLine)
	}
	if len(state.RouteItems) != 2 {
		t.Fatalf("expected 2 route items, got %d", len(state.RouteItems))
	}
	if state.RouteItems[0].OpenURL != "https://api.acme.test" {
		t.Fatalf("expected first route OpenURL pass-through, got %q", state.RouteItems[0].OpenURL)
	}
	if state.RouteItems[1].OpenURL != "http://acme.test" {
		t.Fatalf("expected second route OpenURL pass-through, got %q", state.RouteItems[1].OpenURL)
	}
}

func TestMenubarActionsDispatchAndErrorVisibility(t *testing.T) {
	admin := &stubAdminClient{}
	opener := &stubOpener{}
	d := newDispatcher(admin, opener)

	if err := d.refresh(context.Background()); err != nil {
		t.Fatalf("refresh should not fail: %v", err)
	}
	if admin.refreshReason == "" {
		t.Fatalf("expected refresh to call admin client")
	}

	if err := d.togglePause(context.Background(), true); err != nil {
		t.Fatalf("pause should not fail: %v", err)
	}
	if !admin.pauseCalled {
		t.Fatalf("expected pause action to call PauseRouting")
	}

	if err := d.togglePause(context.Background(), false); err != nil {
		t.Fatalf("resume should not fail: %v", err)
	}
	if !admin.resumeCalled {
		t.Fatalf("expected resume action to call ResumeRouting")
	}

	if err := d.toggleStartup(context.Background(), true); err != nil {
		t.Fatalf("toggle startup should not fail: %v", err)
	}
	if admin.startupToggle.Role != "menubar" {
		t.Fatalf("expected startup toggle role menubar, got %q", admin.startupToggle.Role)
	}

	admin.refreshErr = errors.New("refresh unavailable")
	if err := d.refresh(context.Background()); err == nil || err.Error() != "refresh unavailable" {
		t.Fatalf("expected refresh error to surface exactly, got %v", err)
	}
}

func TestMenubarOpenActionsUseFixedAndProjectedURLs(t *testing.T) {
	opener := &stubOpener{}
	d := newDispatcher(&stubAdminClient{}, opener)

	if err := d.openDashboard(context.Background()); err != nil {
		t.Fatalf("open dashboard should not fail: %v", err)
	}
	if got, want := opener.opened[0], "http://127.0.0.1:45831/"; got != want {
		t.Fatalf("dashboard URL mismatch: got %q want %q", got, want)
	}

	if err := d.openLogs(context.Background()); err != nil {
		t.Fatalf("open logs should not fail: %v", err)
	}
	if got, want := opener.opened[1], "http://127.0.0.1:45831/logs"; got != want {
		t.Fatalf("logs URL mismatch: got %q want %q", got, want)
	}

	if err := d.openRoute(context.Background(), "https://api.acme.test"); err != nil {
		t.Fatalf("open route should not fail: %v", err)
	}
	if got, want := opener.opened[2], "https://api.acme.test"; got != want {
		t.Fatalf("expected OpenURL pass-through for route action, got %q want %q", got, want)
	}
}

func TestMenubarOfflineStateUsesApprovedCopyAndKeepsRepairActions(t *testing.T) {
	state := offlineMenuState(errors.New("admin offline"))

	if got := state.HealthLine; got != "Daemon: offline" {
		t.Fatalf("expected offline health line, got %q", got)
	}
	if got := state.ErrorLine; got != "DevProxy can’t reach the daemon right now. Ensure the daemon is running, then select Run Doctor for repair guidance." {
		t.Fatalf("unexpected offline copy: %q", got)
	}
	if !state.RepairActions.RunDoctor {
		t.Fatalf("expected run doctor repair action to remain available")
	}
	if !state.RepairActions.OpenLogs {
		t.Fatalf("expected open logs repair action to remain available")
	}
	if !state.RepairActions.OpenDashboard {
		t.Fatalf("expected open dashboard repair action to remain available")
	}
}

type stubAdminClient struct {
	refreshReason string
	refreshErr    error
	pauseCalled   bool
	resumeCalled  bool
	startupToggle adminapi.StartupToggleRequest
	startupResp   adminapi.StartupToggleResponse
	startupErr    error
}

func (s *stubAdminClient) Status(context.Context) (admin.StatusView, error) { return admin.StatusView{}, nil }
func (s *stubAdminClient) Routes(context.Context) ([]admin.RouteView, error) { return nil, nil }
func (s *stubAdminClient) Refresh(_ context.Context, reason string) (adminapi.RefreshResponse, error) {
	s.refreshReason = reason
	if s.refreshErr != nil {
		return adminapi.RefreshResponse{}, s.refreshErr
	}
	return adminapi.RefreshResponse{Accepted: true, Refreshed: true}, nil
}
func (s *stubAdminClient) PauseRouting(context.Context) (adminapi.RoutingPauseResumeResponse, error) {
	s.pauseCalled = true
	return adminapi.RoutingPauseResumeResponse{Paused: true}, nil
}
func (s *stubAdminClient) ResumeRouting(context.Context) (adminapi.RoutingPauseResumeResponse, error) {
	s.resumeCalled = true
	return adminapi.RoutingPauseResumeResponse{Paused: false}, nil
}
func (s *stubAdminClient) StartupStatus(context.Context) (adminapi.StartupStatusResponse, error) {
	return adminapi.StartupStatusResponse{Roles: []adminapi.StartupRoleStatus{{Role: "daemon"}, {Role: "menubar", Toggleable: true}}}, nil
}
func (s *stubAdminClient) SetStartupEnabled(_ context.Context, req adminapi.StartupToggleRequest) (adminapi.StartupToggleResponse, error) {
	s.startupToggle = req
	if s.startupErr != nil {
		return adminapi.StartupToggleResponse{}, s.startupErr
	}
	if s.startupResp.Role == "" {
		return adminapi.StartupToggleResponse{Role: req.Role, Enabled: req.Enabled, AffectedRole: req.Role}, nil
	}
	return s.startupResp, nil
}
func (s *stubAdminClient) Doctor(context.Context) (admin.DoctorView, error) { return admin.DoctorView{}, nil }

type stubOpener struct{ opened []string }

func (s *stubOpener) OpenURL(_ context.Context, target string) error {
	s.opened = append(s.opened, target)
	return nil
}
