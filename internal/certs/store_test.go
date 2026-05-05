package certs

import (
	"testing"

	"github.com/mochaka/devproxy/internal/routing"
)

func TestCertificateInventoryReusesWildcardCoverageForProject(t *testing.T) {
	snap := routing.Snapshot{Routes: map[string]routing.Route{
		"acme.test": {
			Hostname:        "acme.test",
			ServedHostnames: []string{"acme.test", "api.acme.test", "mailpit.acme.test"},
		},
	}}

	existing := map[string]StoredCertificate{
		"acme.test": {
			ProjectRoot: "acme.test",
			SANs:        []string{"acme.test", "*.acme.test"},
		},
	}

	plan := BuildCertificateInventory(snap, "test", existing)
	if len(plan) != 1 {
		t.Fatalf("expected one project inventory unit, got %d", len(plan))
	}
	if !plan[0].ReuseExisting {
		t.Fatalf("expected wildcard-covered hostnames to reuse existing certificate")
	}
}

func TestCertificateInventoryIssuesForNewProjectRoot(t *testing.T) {
	snap := routing.Snapshot{Routes: map[string]routing.Route{
		"api.billing.test": {
			Hostname:        "api.billing.test",
			ServedHostnames: []string{"billing.test", "api.billing.test"},
		},
	}}

	plan := BuildCertificateInventory(snap, "test", nil)
	if len(plan) != 1 {
		t.Fatalf("expected one project inventory unit, got %d", len(plan))
	}
	if plan[0].ReuseExisting {
		t.Fatalf("expected new project root to require issuance")
	}
	if got, want := plan[0].ProjectRoot, "billing.test"; got != want {
		t.Fatalf("expected project root %q, got %q", want, got)
	}
}

func TestCertificateInventoryRequestsReissueForDeeperHostnamesOutsideWildcard(t *testing.T) {
	snap := routing.Snapshot{Routes: map[string]routing.Route{
		"deep": {
			Hostname:        "metrics.api.acme.test",
			ServedHostnames: []string{"acme.test", "api.acme.test", "metrics.api.acme.test"},
		},
	}}

	existing := map[string]StoredCertificate{
		"acme.test": {
			ProjectRoot: "acme.test",
			SANs:        []string{"acme.test", "*.acme.test"},
		},
	}

	plan := BuildCertificateInventory(snap, "test", existing)
	if len(plan) != 1 {
		t.Fatalf("expected one project inventory unit, got %d", len(plan))
	}
	if plan[0].ReuseExisting {
		t.Fatalf("expected deeper hostname shape to require reissue or added coverage")
	}
	if len(plan[0].RequiredSANs) < 3 {
		t.Fatalf("expected SAN list to include deeper hostnames, got %v", plan[0].RequiredSANs)
	}
}
