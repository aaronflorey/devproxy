package admin

import (
	"testing"
	"time"

	"github.com/mochaka/devproxy/internal/routing"
)

func TestStatusIncludesIndependentNetworkAndCertificateHealth(t *testing.T) {
	now := time.Now().UTC()
	snapshot := routing.Snapshot{Version: "v1", CreatedAt: now, Routes: map[string]routing.Route{"api.acme.test": {Hostname: "api.acme.test"}}}
	watcher := WatcherHealth{Connected: true}

	status := BuildStatus(snapshot, watcher, now, NetworkRuntimeStatus{
		DNS:              DNSStatus{Healthy: true, ManagedSuffix: "test"},
		HTTP:             ListenerStatus{Enabled: true, Bound: true, BindAddress: "127.0.0.1:80"},
		HTTPS:            ListenerStatus{Enabled: true, Bound: false, BindAddress: "127.0.0.1:443", LastError: "bind: permission denied"},
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

func TestRoutesProjectionIncludesDeterministicOpenURLAndFallbackReason_D04(t *testing.T) {
	snapshot := routing.Snapshot{
		Routes: map[string]routing.Route{
			"api.acme.test": {
				Hostname: "api.acme.test",
				Winner:   routing.Candidate{ContainerName: "acme-api-1"},
				Upstream: routing.Upstream{Scheme: "http", Host: "127.0.0.1", Port: 8080},
			},
		},
	}

	routes := RoutesFromSnapshot(snapshot)
	if len(routes) != 1 {
		t.Fatalf("expected one route view, got %+v", routes)
	}
	if routes[0].OpenURL == "" {
		t.Fatalf("expected deterministic open URL for route, got %+v", routes[0])
	}
	if routes[0].PreferredScheme == "" {
		t.Fatalf("expected preferred scheme metadata, got %+v", routes[0])
	}
	if routes[0].PreferredScheme == "http" && routes[0].FallbackReason == "" {
		t.Fatalf("expected non-empty fallback reason when preferring http, got %+v", routes[0])
	}
}

func TestSessionIssueProjectionIsBoundedNewestFirst_D05(t *testing.T) {
	issues := make([]SessionIssue, 0, 30)
	now := time.Now().UTC()
	for i := 0; i < 30; i++ {
		issues = append(issues, SessionIssue{
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Role:      "daemon",
			Action:    "refresh",
			Message:   "failure",
		})
	}

	view := BuildSessionIssues(issues)
	if len(view) != 25 {
		t.Fatalf("expected bounded session issue list of 25 entries, got %d", len(view))
	}
	if !view[0].Timestamp.After(view[len(view)-1].Timestamp) {
		t.Fatalf("expected newest-first ordering in session issues, got first=%s last=%s", view[0].Timestamp, view[len(view)-1].Timestamp)
	}
}
