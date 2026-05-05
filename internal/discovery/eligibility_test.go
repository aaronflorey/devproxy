package discovery

import "testing"

func TestEligibility(t *testing.T) {
	cfg := EligibilityConfig{IgnoredServices: []string{"redis"}, IgnoredPorts: []int{6379}}

	blocked := CandidateInput{Service: "redis", Running: true, PublishedTCPPorts: []int{8080}}
	if IsEligible(blocked, cfg) {
		t.Fatalf("expected ignored service to be ineligible")
	}

	override := CandidateInput{Service: "redis", Running: true, EnabledLabel: boolPtr(true), PublishedTCPPorts: []int{8080}}
	if !IsEligible(override, cfg) {
		t.Fatalf("expected devproxy.enable=true override")
	}

	noPorts := CandidateInput{Service: "web", Running: true, EnabledLabel: boolPtr(true), PublishedTCPPorts: nil}
	if IsEligible(noPorts, cfg) {
		t.Fatalf("expected missing published port to remain ineligible")
	}
}

func boolPtr(v bool) *bool { return &v }
