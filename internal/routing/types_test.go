package routing

import (
	"reflect"
	"testing"
)

func TestRouteContractsExposeConflictAndWarningSurfaces(t *testing.T) {
	candidateType := reflect.TypeOf(Candidate{})
	for _, field := range []string{"ContainerID", "ContainerName", "Project", "Service", "Source", "PublishedPorts", "Labels", "Warnings"} {
		if _, ok := candidateType.FieldByName(field); !ok {
			t.Fatalf("candidate missing field %s", field)
		}
	}

	routeType := reflect.TypeOf(Route{})
	for _, field := range []string{"Hostname", "Domains", "Upstream", "Winner", "Losers", "Priority", "Provenance"} {
		if _, ok := routeType.FieldByName(field); !ok {
			t.Fatalf("route missing field %s", field)
		}
	}

	warningType := reflect.TypeOf(Warning{})
	for _, field := range []string{"Code", "Message", "Container", "Field", "Severity", "Source"} {
		if _, ok := warningType.FieldByName(field); !ok {
			t.Fatalf("warning missing field %s", field)
		}
	}

	conflictType := reflect.TypeOf(Conflict{})
	for _, field := range []string{"Hostname", "Winner", "Losers", "Reason", "PriorityTie"} {
		if _, ok := conflictType.FieldByName(field); !ok {
			t.Fatalf("conflict missing field %s", field)
		}
	}
}
