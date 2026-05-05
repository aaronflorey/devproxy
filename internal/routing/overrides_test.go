package routing

import "testing"

func TestOverridePrecedence(t *testing.T) {
	config := RoutePreferences{Domain: "api.acme.test", Port: intPtr(8080), Priority: intPtr(10)}
	labels := RoutePreferences{Port: intPtr(3000)}

	merged, warnings := MergeOverrides(config, labels)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if merged.Domain != "api.acme.test" || merged.Port == nil || *merged.Port != 3000 || merged.Priority == nil || *merged.Priority != 10 {
		t.Fatalf("unexpected merged result: %#v", merged)
	}
}

func intPtr(v int) *int { return &v }
