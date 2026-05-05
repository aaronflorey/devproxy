package registry

import (
	"testing"

	"github.com/mochaka/devproxy/internal/routing"
)

func TestRegistrySnapshot(t *testing.T) {
	builder := NewBuilder()
	routes := []routing.Route{{Hostname: "api.acme.test", Winner: routing.Candidate{ContainerName: "api"}}}
	conflicts := []routing.Conflict{{Hostname: "api.acme.test", Reason: "priority"}}
	warnings := []routing.Warning{{Code: "x"}}

	snap := builder.Build(routes, conflicts, warnings)
	if snap.Version == "" || snap.CreatedAt.IsZero() {
		t.Fatalf("expected immutable snapshot metadata")
	}
	if len(snap.Routes) != 1 || len(snap.Conflicts) != 1 || len(snap.Warnings) != 1 {
		t.Fatalf("expected routes conflicts warnings to be published")
	}
}
