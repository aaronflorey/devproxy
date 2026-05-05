package routing

import "testing"

func TestConflictResolution(t *testing.T) {
	routes := []Route{
		{Hostname: "api.acme.test", Priority: 10, Winner: Candidate{ContainerName: "z-api"}},
		{Hostname: "api.acme.test", Priority: 20, Winner: Candidate{ContainerName: "a-api"}},
		{Hostname: "api.acme.test", Priority: 20, Winner: Candidate{ContainerName: "b-api"}},
	}

	active, conflicts := ResolveConflicts(routes)
	if len(active) != 1 || active[0].Winner.ContainerName != "a-api" {
		t.Fatalf("expected highest priority + tie-break winner, got %#v", active)
	}
	if len(conflicts) != 1 || len(conflicts[0].Losers) != 2 {
		t.Fatalf("expected loser retention in conflict read model")
	}
}
