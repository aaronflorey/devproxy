package admin

import (
	"time"

	"github.com/mochaka/devproxy/internal/daemon"
	"github.com/mochaka/devproxy/internal/routing"
)

type StatusView struct {
	SnapshotVersion string
	ActiveRoutes    int
	Conflicts       int
	Warnings        int
	LastSync        time.Time
	Watcher         daemon.WatcherHealth
	DNS             DNSStatus
	HTTP            ListenerStatus
	HTTPS           ListenerStatus
	Paused          bool
	CertificateReady bool
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

func BuildStatus(snapshot routing.Snapshot, watcher daemon.WatcherHealth, lastSync time.Time, runtime NetworkRuntimeStatus) StatusView {
	return StatusView{
		SnapshotVersion:  snapshot.Version,
		ActiveRoutes:     len(snapshot.Routes),
		Conflicts:        len(snapshot.Conflicts),
		Warnings:         len(snapshot.Warnings),
		LastSync:         lastSync,
		Watcher:          watcher,
		DNS:              runtime.DNS,
		HTTP:             runtime.HTTP,
		HTTPS:            runtime.HTTPS,
		Paused:           runtime.Paused,
		CertificateReady: runtime.CertificateReady,
	}
}

func NetworkRuntimeStatusFromDaemon(health daemon.NetworkRuntimeHealth) NetworkRuntimeStatus {
	return NetworkRuntimeStatus{
		DNS: DNSStatus{Healthy: health.DNS.Bound, ManagedSuffix: health.ManagedSuffix},
		HTTP: ListenerStatus{Enabled: health.HTTP.Enabled, Bound: health.HTTP.Bound, BindAddress: health.HTTP.BindAddress, LastError: health.HTTP.LastError},
		HTTPS: ListenerStatus{Enabled: health.HTTPS.Enabled, Bound: health.HTTPS.Bound, BindAddress: health.HTTPS.BindAddress, LastError: health.HTTPS.LastError},
		Paused: health.Paused,
		CertificateReady: health.CertificateReady,
	}
}
