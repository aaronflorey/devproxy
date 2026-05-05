package admin

import (
	"testing"
	"time"

	"github.com/mochaka/devproxy/internal/daemon"
	"github.com/mochaka/devproxy/internal/routing"
)

func TestStatusIncludesIndependentNetworkAndCertificateHealth(t *testing.T) {
	now := time.Now().UTC()
	snapshot := routing.Snapshot{Version: "v1", CreatedAt: now, Routes: map[string]routing.Route{"api.acme.test": {Hostname: "api.acme.test"}}}
	watcher := daemon.WatcherHealth{Connected: true}

	status := BuildStatus(snapshot, watcher, now, NetworkRuntimeStatus{
		DNS: DNSStatus{Healthy: true, ManagedSuffix: "test"},
		HTTP: ListenerStatus{Enabled: true, Bound: true, BindAddress: "127.0.0.1:80"},
		HTTPS: ListenerStatus{Enabled: true, Bound: false, BindAddress: "127.0.0.1:443", LastError: "bind: permission denied"},
		Paused:           true,
		CertificateReady: false,
	})

	if !status.DNS.Healthy {
		t.Fatalf("expected DNS health true")
	}
	if !status.HTTP.Enabled || !status.HTTP.Bound {
		t.Fatalf("expected HTTP listener health fields populated")
	}
	if !status.HTTPS.Enabled || status.HTTPS.Bound {
		t.Fatalf("expected HTTPS health fields populated with independent bind result")
	}
	if !status.Paused {
		t.Fatalf("expected paused state to be represented independently")
	}
	if status.CertificateReady {
		t.Fatalf("expected certificate readiness to be represented independently")
	}
}
