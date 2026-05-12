package daemon

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/mochaka/devproxy/internal/admin"
	"github.com/mochaka/devproxy/internal/adminapi"
	"github.com/mochaka/devproxy/internal/certs"
	"github.com/mochaka/devproxy/internal/config"
	"github.com/mochaka/devproxy/internal/dashboard"
	"github.com/mochaka/devproxy/internal/install"
)

type AppConfig struct {
	AdminSocketPath     string
	DashboardAddress    string
	DNSAddress          string
	HTTPAddress         string
	HTTPSAddress        string
	Config              config.Config
	DockerPing          func(context.Context) error
	DockerScan          func(context.Context) ([]ContainerState, error)
	DockerEvents        DockerEventSource
	EnsureMKCert        func(context.Context) error
	BuildNetworkRuntime func(context.Context) error
}

type App struct {
	cfg        AppConfig
	reconciler *Reconciler
	watcher    *Watcher
	network    *NetworkRuntime
	server     *adminapi.Server
	dashboard  *dashboard.Server
	dashCancel context.CancelFunc
	issueMu    sync.Mutex
	issues     []admin.SessionIssue
}

func NewApp(cfg AppConfig) *App {
	r := NewReconciler(ReconcilerOptions{
		Suffix:          cfg.Config.DomainSuffix,
		RootServices:    cfg.Config.RootServices,
		IgnoredServices: cfg.Config.IgnoredServices,
		IgnoredPorts:    cfg.Config.IgnoredPorts,
		Overrides:       cfg.Config.Overrides,
	})
	_ = r.RebuildSnapshot(nil)
	if cfg.AdminSocketPath == "" {
		cfg.AdminSocketPath = "/tmp/devproxy/admin.sock"
	}
	if cfg.DashboardAddress == "" {
		cfg.DashboardAddress = dashboard.DefaultListenAddress
	}
	if cfg.HTTPAddress == "" {
		cfg.HTTPAddress = "127.0.0.1:80"
	}
	if cfg.HTTPSAddress == "" {
		cfg.HTTPSAddress = "127.0.0.1:443"
	}
	if cfg.Config.Serving.ManagedSuffix == "" {
		cfg.Config.Serving.ManagedSuffix = cfg.Config.DomainSuffix
	}
	return &App{cfg: cfg, reconciler: r, watcher: NewWatcher(r)}
}

func (a *App) Start(ctx context.Context) error {
	if ping := a.cfg.DockerPing; ping != nil {
		if err := ping(ctx); err != nil {
			return fmt.Errorf("docker reachability check failed: %w", err)
		}
	}

	if err := a.refreshFromDocker(ctx); err != nil {
		return err
	}

	if ensure := a.cfg.EnsureMKCert; ensure != nil {
		if err := ensure(ctx); err != nil {
			return fmt.Errorf("mkcert prerequisites failed: %w", err)
		}
	}

	if build := a.cfg.BuildNetworkRuntime; build != nil {
		if err := build(ctx); err != nil {
			return fmt.Errorf("listener bind validation failed: %w", err)
		}
	}

	network, err := NewNetworkRuntime(NetworkRuntimeConfig{
		ManagedSuffix:        a.cfg.Config.Serving.ManagedSuffix,
		Snapshot:             a.reconciler.Snapshot,
		RoutingPaused:        a.reconciler.IsRoutingPaused,
		StoredCertificates:   map[string]certs.StoredCertificate{},
		CertificateOutputDir: filepath.Dir(a.cfg.AdminSocketPath),
		DNSAddress:           a.cfg.DNSAddress,
		HTTPAddress:          a.cfg.HTTPAddress,
		HTTPSAddress:         a.cfg.HTTPSAddress,
	})
	if err != nil {
		return fmt.Errorf("build network runtime: %w", err)
	}
	if err := network.Start(); err != nil {
		return fmt.Errorf("listener bind startup failed: %w", err)
	}
	a.network = network

	server, err := adminapi.NewServer(adminapi.ServerConfig{
		SocketPath: a.cfg.AdminSocketPath,
		State:      a.stateSnapshot,
		Refresh: func(ctx context.Context, reason string) error {
			_ = reason
			return a.Refresh(ctx)
		},
		SetRoutingPaused:  a.setRoutingPaused,
		StartupStatus:     a.startupStatus,
		SetStartupEnabled: a.setStartupEnabled,
	})
	if err != nil {
		_ = a.network.Close()
		return err
	}
	if err := server.Start(); err != nil {
		_ = a.network.Close()
		return fmt.Errorf("start admin socket server: %w", err)
	}
	a.server = server

	dashboardClient := adminapi.NewClient(a.cfg.AdminSocketPath)
	dashboardServer := dashboard.NewServer(dashboard.Config{ListenAddress: a.cfg.DashboardAddress, Client: dashboardClient})
	dashCtx, dashCancel := context.WithCancel(context.Background())
	a.dashboard = dashboardServer
	a.dashCancel = dashCancel
	go func() {
		if err := dashboardServer.Run(dashCtx); err != nil {
			a.recordIssue("dashboard", "start", err.Error())
		}
	}()
	if a.cfg.DockerEvents != nil && a.cfg.DockerScan != nil {
		go a.watchDockerEvents(ctx)
	}
	return nil
}

func (a *App) Run(ctx context.Context) error {
	if err := a.Start(ctx); err != nil {
		return err
	}
	<-ctx.Done()
	return a.Close(context.Background())
}

func (a *App) Refresh(ctx context.Context) error {
	return a.refreshFromDocker(ctx)
}

func (a *App) setRoutingPaused(_ context.Context, paused bool) error {
	a.reconciler.SetRoutingPaused(paused)
	return nil
}

func (a *App) startupStatus(context.Context) ([]admin.StartupRoleStatus, error) {
	statuses := install.StartupStatuses(install.DefaultPaths())
	out := make([]admin.StartupRoleStatus, 0, len(statuses))
	for _, item := range statuses {
		out = append(out, admin.StartupRoleStatus{
			Role:          item.Role,
			Domain:        item.Domain,
			Label:         item.Label,
			Installed:     item.Installed,
			Running:       item.Running,
			Toggleable:    item.Toggleable,
			StatusMessage: item.StatusMessage,
		})
	}
	return out, nil
}

func (a *App) setStartupEnabled(ctx context.Context, role string, enabled bool) (admin.StartupRoleStatus, error) {
	if role != "daemon" && role != "menubar" {
		return admin.StartupRoleStatus{}, fmt.Errorf("unsupported startup role %q", role)
	}
	if role == "daemon" {
		msg := "daemon startup is managed by system launchd and cannot be toggled from UI"
		a.recordIssue("daemon", "startup-toggle", msg)
		return admin.StartupRoleStatus{Role: "daemon", Domain: "system", Label: "com.devproxy.daemon", Installed: true, Running: true, Toggleable: false, StatusMessage: msg}, errors.New(msg)
	}

	if err := install.SetMenubarStartupEnabled(ctx, install.DefaultPaths(), enabled); err != nil {
		a.recordIssue(role, "startup-toggle", err.Error())
		return admin.StartupRoleStatus{}, err
	}

	for _, st := range install.StartupStatuses(install.DefaultPaths()) {
		if st.Role == role {
			return admin.StartupRoleStatus{Role: st.Role, Domain: st.Domain, Label: st.Label, Installed: st.Installed, Running: st.Running, Toggleable: st.Toggleable, StatusMessage: st.StatusMessage}, nil
		}
	}
	return admin.StartupRoleStatus{Role: role}, nil
}

func (a *App) recordIssue(role, action, message string) {
	a.issueMu.Lock()
	defer a.issueMu.Unlock()
	a.issues = append(a.issues, admin.SessionIssue{Timestamp: time.Now().UTC(), Role: role, Action: action, Message: message})
	if len(a.issues) > admin.SessionIssueLimit {
		a.issues = a.issues[len(a.issues)-admin.SessionIssueLimit:]
	}
}

func (a *App) Close(ctx context.Context) error {
	if a.dashCancel != nil {
		a.dashCancel()
	}
	if a.server != nil {
		_ = a.server.Close(ctx)
	}
	if a.network != nil {
		return a.network.Close()
	}
	return nil
}

func (a *App) stateSnapshot() adminapi.StateSnapshot {
	snapshot := a.reconciler.Snapshot()
	runtimeHealth := NetworkRuntimeHealth{ManagedSuffix: a.cfg.Config.Serving.ManagedSuffix}
	if a.network != nil {
		runtimeHealth = a.network.Health()
	}

	watcher := a.watcher.Health()
	status := admin.BuildStatus(snapshot, admin.WatcherHealth{Connected: watcher.Connected, LastDisconnect: watcher.LastDisconnect, LastReconnectSync: watcher.LastReconnectSync}, a.reconciler.LastSync(), admin.NetworkRuntimeStatusFromHealth(admin.NetworkRuntimeHealth{
		DNS:              admin.ListenerStatus{Enabled: runtimeHealth.DNS.Enabled, Bound: runtimeHealth.DNS.Bound, BindAddress: runtimeHealth.DNS.BindAddress, LastError: runtimeHealth.DNS.LastError},
		HTTP:             admin.ListenerStatus{Enabled: runtimeHealth.HTTP.Enabled, Bound: runtimeHealth.HTTP.Bound, BindAddress: runtimeHealth.HTTP.BindAddress, LastError: runtimeHealth.HTTP.LastError},
		HTTPS:            admin.ListenerStatus{Enabled: runtimeHealth.HTTPS.Enabled, Bound: runtimeHealth.HTTPS.Bound, BindAddress: runtimeHealth.HTTPS.BindAddress, LastError: runtimeHealth.HTTPS.LastError},
		Paused:           runtimeHealth.Paused,
		CertificateReady: runtimeHealth.CertificateReady,
		ManagedSuffix:    runtimeHealth.ManagedSuffix,
	}))
	doctor := admin.BuildDoctor(snapshot, admin.NetworkDoctorStatus{
		DNSHealthy:       runtimeHealth.DNS.Bound,
		HTTPBound:        runtimeHealth.HTTP.Bound,
		HTTPSBound:       runtimeHealth.HTTPS.Bound,
		Paused:           runtimeHealth.Paused,
		CertificateReady: runtimeHealth.CertificateReady,
		ManagedSuffix:    runtimeHealth.ManagedSuffix,
	})
	logs := admin.BuildSessionEvents(snapshot)
	startupRoles, _ := a.startupStatus(context.Background())
	status.StartupRoles = startupRoles

	a.issueMu.Lock()
	issues := append([]admin.SessionIssue(nil), a.issues...)
	a.issueMu.Unlock()

	return adminapi.StateSnapshot{Snapshot: snapshot, Status: status, Doctor: doctor, Logs: logs, Issues: issues}
}

func (a *App) refreshFromDocker(ctx context.Context) error {
	if a.cfg.DockerScan == nil {
		return a.watcher.OnReconnect(nil)
	}

	containers, err := a.cfg.DockerScan(ctx)
	if err != nil {
		a.watcher.OnDisconnect()
		a.recordIssue("docker", "scan", err.Error())
		return fmt.Errorf("docker container sync failed: %w", err)
	}

	return a.watcher.OnReconnect(containers)
}

func (a *App) watchDockerEvents(ctx context.Context) {
	for {
		stream, err := a.cfg.DockerEvents(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			a.recordWatcherIssue("events-connect", err)
			if !waitForWatcherRetry(ctx) {
				return
			}
			continue
		}

		if !a.watcher.Health().Connected {
			if err := a.resyncWatcher(ctx); err != nil {
				_ = stream.Close()
				if !waitForWatcherRetry(ctx) {
					return
				}
				continue
			}
		}

		if err := a.consumeDockerEvents(ctx, stream); err != nil {
			_ = stream.Close()
			if ctx.Err() != nil {
				return
			}
			a.recordWatcherIssue("events-stream", err)
			if !waitForWatcherRetry(ctx) {
				return
			}
			continue
		}
		return
	}
}

func (a *App) consumeDockerEvents(ctx context.Context, stream *EventStream) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case err, ok := <-stream.Errors:
			if !ok || err == nil {
				return errors.New("docker event stream closed")
			}
			return err
		case event, ok := <-stream.Events:
			if !ok {
				return errors.New("docker event stream closed")
			}
			a.handleDockerEvent(ctx, event)
		}
	}
}

func (a *App) handleDockerEvent(ctx context.Context, event DockerEvent) {
	if !isSupportedDockerEvent(event.Action) {
		return
	}
	containers, err := a.cfg.DockerScan(ctx)
	if err != nil {
		a.recordWatcherIssue("event-sync", fmt.Errorf("sync %s event: %w", event.Action, err))
		return
	}
	if !a.watcher.Health().Connected {
		if err := a.watcher.OnReconnect(containers); err != nil {
			a.recordWatcherIssue("event-resync", err)
		}
		return
	}
	if err := a.watcher.HandleEvent(event.Action, containers); err != nil {
		a.recordIssue("docker", "event-ignore", err.Error())
	}
}

func (a *App) resyncWatcher(ctx context.Context) error {
	containers, err := a.cfg.DockerScan(ctx)
	if err != nil {
		a.recordWatcherIssue("reconnect-sync", err)
		return err
	}
	if err := a.watcher.OnReconnect(containers); err != nil {
		a.recordWatcherIssue("reconnect-sync", err)
		return err
	}
	return nil
}

func (a *App) recordWatcherIssue(action string, err error) {
	a.watcher.OnDisconnect()
	a.recordIssue("docker", action, err.Error())
}

func waitForWatcherRetry(ctx context.Context) bool {
	timer := time.NewTimer(200 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
