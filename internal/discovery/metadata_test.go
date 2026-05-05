package discovery

import "testing"

func TestComposeMetadataPrefersLabelsAndFallsBackToName(t *testing.T) {
	container := Container{
		ID:   "abc",
		Name: "acme-api-1",
		Labels: map[string]string{
			"com.docker.compose.project": "acme",
			"com.docker.compose.service": "api",
		},
	}

	c := BuildCandidateBase(container)
	if c.Project != "acme" || c.Service != "api" || c.Source != "compose-labels" {
		t.Fatalf("expected compose metadata, got %#v", c)
	}

	fallback := BuildCandidateBase(Container{ID: "def", Name: "shop-web-1"})
	if fallback.Project != "shop" || fallback.Service != "web" || fallback.Source != "container-name" {
		t.Fatalf("expected fallback metadata, got %#v", fallback)
	}
}

func TestMalformedLabelFieldsWarnAndContinue(t *testing.T) {
	container := Container{ID: "x", Name: "acme-api-1", Labels: map[string]string{"devproxy.priority": "bad"}}
	warnings := []WarningRecord{}
	_, warnings = ApplyLabelFields(RouteOptions{}, container, warnings)
	if len(warnings) == 0 {
		t.Fatalf("expected warning for malformed label")
	}
}
