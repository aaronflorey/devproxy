package admin

import (
	"github.com/mochaka/devproxy/internal/routing"
	"time"
)

type StatusView struct {
	SnapshotVersion  string
	ActiveRoutes     int
	Conflicts        int
	Warnings         int
	ConflictDetails  []routing.Conflict
	WarningDetails   []routing.Warning
	LastSync         time.Time
	Watcher          WatcherHealth
	DNS              DNSStatus
	HTTP             ListenerStatus
	HTTPS            ListenerStatus
	Paused           bool
	CertificateReady bool
	StartupRoles     []StartupRoleStatus
}

type StartupRoleStatus struct {
	Role          string
	Domain        string
	Label         string
	Installed     bool
	Running       bool
	Toggleable    bool
	StatusMessage string
}

type WatcherHealth struct {
	Connected         bool
	LastDisconnect    time.Time
	LastReconnectSync time.Time
}

type DNSStatus struct {
	Healthy       bool
	ManagedSuffix string
}

type ListenerStatus struct {
	Enabled     bool
	Bound       bool
	BindAddress string
	LastError   string
}

type NetworkRuntimeStatus struct {
	DNS              DNSStatus
	HTTP             ListenerStatus
	HTTPS            ListenerStatus
	Paused           bool
	CertificateReady bool
}

func BuildStatus(snapshot routing.Snapshot, watcher WatcherHealth, lastSync time.Time, runtime NetworkRuntimeStatus) StatusView {
	return StatusView{
		SnapshotVersion:  snapshot.Version,
		ActiveRoutes:     len(snapshot.Routes),
		Conflicts:        len(snapshot.Conflicts),
		Warnings:         len(snapshot.Warnings),
		ConflictDetails:  append([]routing.Conflict(nil), snapshot.Conflicts...),
		WarningDetails:   append([]routing.Warning(nil), snapshot.Warnings...),
		LastSync:         lastSync,
		Watcher:          watcher,
		DNS:              runtime.DNS,
		HTTP:             runtime.HTTP,
		HTTPS:            runtime.HTTPS,
		Paused:           runtime.Paused,
		CertificateReady: runtime.CertificateReady,
	}
}

type NetworkRuntimeHealth struct {
	DNS              ListenerStatus
	HTTP             ListenerStatus
	HTTPS            ListenerStatus
	Paused           bool
	CertificateReady bool
	ManagedSuffix    string
}

func NetworkRuntimeStatusFromHealth(health NetworkRuntimeHealth) NetworkRuntimeStatus {
	return NetworkRuntimeStatus{
		DNS:              DNSStatus{Healthy: health.DNS.Bound, ManagedSuffix: health.ManagedSuffix},
		HTTP:             ListenerStatus{Enabled: health.HTTP.Enabled, Bound: health.HTTP.Bound, BindAddress: health.HTTP.BindAddress, LastError: health.HTTP.LastError},
		HTTPS:            ListenerStatus{Enabled: health.HTTPS.Enabled, Bound: health.HTTPS.Bound, BindAddress: health.HTTPS.BindAddress, LastError: health.HTTPS.LastError},
		Paused:           health.Paused,
		CertificateReady: health.CertificateReady,
	}
}
