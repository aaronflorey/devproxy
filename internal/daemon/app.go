package daemon

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/mochaka/devproxy/internal/admin"
	"github.com/mochaka/devproxy/internal/adminapi"
	"github.com/mochaka/devproxy/internal/certs"
	"github.com/mochaka/devproxy/internal/config"
)

type AppConfig struct {
	AdminSocketPath string
	HTTPAddress     string
	HTTPSAddress    string
	Config          config.Config
	DockerPing      func(context.Context) error
	EnsureMKCert    func(context.Context) error
	BuildNetworkRuntime func(context.Context) error
}

type App struct {
	cfg       AppConfig
	reconciler *Reconciler
	watcher   *Watcher
	network   *NetworkRuntime
	server    *adminapi.Server
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
	return nil
}

func (a *App) Run(ctx context.Context) error {
	if err := a.Start(ctx); err != nil {
		return err
	}
	<-ctx.Done()
	return a.Close(context.Background())
}

func (a *App) Refresh(context.Context) error {
	return a.reconciler.RebuildSnapshot(nil)
}

func (a *App) Close(ctx context.Context) error {
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
		DNS: admin.ListenerStatus{Enabled: runtimeHealth.DNS.Enabled, Bound: runtimeHealth.DNS.Bound, BindAddress: runtimeHealth.DNS.BindAddress, LastError: runtimeHealth.DNS.LastError},
		HTTP: admin.ListenerStatus{Enabled: runtimeHealth.HTTP.Enabled, Bound: runtimeHealth.HTTP.Bound, BindAddress: runtimeHealth.HTTP.BindAddress, LastError: runtimeHealth.HTTP.LastError},
		HTTPS: admin.ListenerStatus{Enabled: runtimeHealth.HTTPS.Enabled, Bound: runtimeHealth.HTTPS.Bound, BindAddress: runtimeHealth.HTTPS.BindAddress, LastError: runtimeHealth.HTTPS.LastError},
		Paused: runtimeHealth.Paused,
		CertificateReady: runtimeHealth.CertificateReady,
		ManagedSuffix: runtimeHealth.ManagedSuffix,
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

	return adminapi.StateSnapshot{Snapshot: snapshot, Status: status, Doctor: doctor, Logs: logs}
}

func DefaultDockerPing(context.Context) error { return nil }

func DefaultEnsureMKCert(context.Context) error {
	_, err := exec.LookPath("mkcert")
	if err != nil {
		return fmt.Errorf("mkcert not found: install mkcert before enabling HTTPS: %w", err)
	}
	return nil
}
