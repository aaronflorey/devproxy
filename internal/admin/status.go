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
}

type DNSStatus struct {
	Healthy       bool
	ManagedSuffix string
}

func BuildStatus(snapshot routing.Snapshot, watcher daemon.WatcherHealth, lastSync time.Time, dnsHealthy bool, managedSuffix string) StatusView {
	return StatusView{SnapshotVersion: snapshot.Version, ActiveRoutes: len(snapshot.Routes), Conflicts: len(snapshot.Conflicts), Warnings: len(snapshot.Warnings), LastSync: lastSync, Watcher: watcher, DNS: DNSStatus{Healthy: dnsHealthy, ManagedSuffix: managedSuffix}}
}
