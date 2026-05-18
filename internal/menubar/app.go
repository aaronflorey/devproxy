package menubar

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mochaka/devproxy/internal/admin"
	"github.com/mochaka/devproxy/internal/adminapi"
)

const (
	dashboardURL       = "http://127.0.0.1:45831/"
	logsURL            = "http://127.0.0.1:45831/logs"
	offlineCopy        = "DevProxy can’t reach the daemon right now. Ensure the daemon is running, then select Run Doctor for repair guidance."
	startupRoleMenubar = "menubar"
)

type adminClient interface {
	Status(context.Context) (admin.StatusView, error)
	Routes(context.Context) ([]admin.RouteView, error)
	Refresh(context.Context, string) (adminapi.RefreshResponse, error)
	PauseRouting(context.Context) (adminapi.RoutingPauseResumeResponse, error)
	ResumeRouting(context.Context) (adminapi.RoutingPauseResumeResponse, error)
	StartupStatus(context.Context) (adminapi.StartupStatusResponse, error)
	SetStartupEnabled(context.Context, adminapi.StartupToggleRequest) (adminapi.StartupToggleResponse, error)
	Doctor(context.Context) (admin.DoctorView, error)
}

type opener interface {
	OpenURL(context.Context, string) error
}

type routeMenuItem struct {
	Hostname string
	OpenURL  string
}

type repairActions struct {
	RunDoctor     bool
	OpenLogs      bool
	OpenDashboard bool
}

type menuState struct {
	HealthLine       string
	PauseLine        string
	ActiveRoutesLine string
	StartupLine      string
	ErrorLine        string
	RouteItems       []routeMenuItem
	RepairActions    repairActions
}

func buildMenuState(status admin.StatusView, routes []admin.RouteView, startup []adminapi.StartupRoleStatus) menuState {
	state := menuState{
		HealthLine:       "Daemon: healthy",
		PauseLine:        "Routing: active",
		ActiveRoutesLine: fmt.Sprintf("Active routes: %d", status.ActiveRoutes),
		RepairActions:    repairActions{RunDoctor: true, OpenLogs: true, OpenDashboard: true},
	}
	if status.Paused {
		state.PauseLine = "Routing: paused"
	}
	state.StartupLine = startupSummary(startup)
	state.RouteItems = make([]routeMenuItem, 0, len(routes))
	for _, r := range routes {
		state.RouteItems = append(state.RouteItems, routeMenuItem{Hostname: r.Hostname, OpenURL: r.OpenURL})
	}
	return state
}

func offlineMenuState(err error) menuState {
	errLine := offlineCopy
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		errLine = "Cannot connect: " + strings.TrimSpace(err.Error())
	}
	return menuState{
		HealthLine:       "Daemon: offline",
		PauseLine:        "Routing: unknown",
		ActiveRoutesLine: "Active routes: unavailable",
		ErrorLine:        errLine,
		RepairActions:    repairActions{RunDoctor: true, OpenLogs: true, OpenDashboard: true},
	}
}

func startupSummary(roles []adminapi.StartupRoleStatus) string {
	if len(roles) == 0 {
		return "Startup: unavailable"
	}
	var daemonStatus, menubarStatus string
	for _, role := range roles {
		switch role.Role {
		case "daemon":
			daemonStatus = role.StatusMessage
		case startupRoleMenubar:
			menubarStatus = role.StatusMessage
		}
	}
	if daemonStatus == "" {
		daemonStatus = "unknown"
	}
	if menubarStatus == "" {
		menubarStatus = "unknown"
	}
	return fmt.Sprintf("Startup — daemon: %s | menubar: %s", daemonStatus, menubarStatus)
}

type dispatcher struct {
	admin  adminClient
	opener opener
}

func newDispatcher(admin adminClient, opener opener) *dispatcher {
	return &dispatcher{admin: admin, opener: opener}
}

func (d *dispatcher) refresh(ctx context.Context) error {
	resp, err := d.admin.Refresh(ctx, "menubar refresh action")
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

func (d *dispatcher) togglePause(ctx context.Context, pause bool) error {
	if pause {
		resp, err := d.admin.PauseRouting(ctx)
		if err != nil {
			return err
		}
		if resp.Error != "" {
			return errors.New(resp.Error)
		}
		return nil
	}
	resp, err := d.admin.ResumeRouting(ctx)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

func (d *dispatcher) toggleStartup(ctx context.Context, enabled bool) error {
	resp, err := d.admin.SetStartupEnabled(ctx, adminapi.StartupToggleRequest{Role: startupRoleMenubar, Enabled: enabled})
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

func (d *dispatcher) openDashboard(ctx context.Context) error {
	return d.opener.OpenURL(ctx, dashboardURL)
}

func (d *dispatcher) openLogs(ctx context.Context) error {
	return d.opener.OpenURL(ctx, logsURL)
}

func (d *dispatcher) openRoute(ctx context.Context, routeURL string) error {
	return d.opener.OpenURL(ctx, routeURL)
}

func (d *dispatcher) runDoctor(ctx context.Context) error {
	_, err := d.admin.Doctor(ctx)
	return err
}
