//go:build darwin

package menubar

import "testing"

func TestRuntimeRouteSlotAssignmentsExposeProjectedRoutes(t *testing.T) {
	routes := []routeMenuItem{
		{Hostname: "api.acme.test", OpenURL: "https://api.acme.test"},
		{Hostname: "acme.test", OpenURL: "http://acme.test"},
	}

	assignments := computeRouteSlotAssignments(0, routes)
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(assignments))
	}
	if !assignments[0].visible || assignments[0].host != "api.acme.test" || assignments[0].openURL != "https://api.acme.test" {
		t.Fatalf("first assignment mismatch: %+v", assignments[0])
	}
	if !assignments[1].visible || assignments[1].host != "acme.test" || assignments[1].openURL != "http://acme.test" {
		t.Fatalf("second assignment mismatch: %+v", assignments[1])
	}
}

func TestRuntimeRouteSlotAssignmentsHideStaleSlotsOnShrink(t *testing.T) {
	routes := []routeMenuItem{{Hostname: "api.acme.test", OpenURL: "https://api.acme.test"}}

	assignments := computeRouteSlotAssignments(3, routes)
	if len(assignments) != 3 {
		t.Fatalf("expected 3 assignments, got %d", len(assignments))
	}
	if !assignments[0].visible {
		t.Fatalf("expected first slot to remain visible")
	}
	for i := 1; i < len(assignments); i++ {
		if assignments[i].visible {
			t.Fatalf("expected stale slot %d to be hidden", i)
		}
		if assignments[i].openURL != "" {
			t.Fatalf("expected stale slot %d openURL cleared, got %q", i, assignments[i].openURL)
		}
	}
}
