package discovery

import "testing"

func TestPortSelection(t *testing.T) {
	ports := []PublishedPort{{HostPort: 3000, Protocol: "tcp"}, {HostPort: 443, Protocol: "tcp"}, {HostPort: 8025, Protocol: "tcp"}}

	selected, source, ok := SelectPublishedPort(ports, RouteOptions{}, 0)
	if !ok || selected.HostPort != 443 || source != "preference" {
		t.Fatalf("expected preferred 443, got %#v source=%s ok=%t", selected, source, ok)
	}

	selected, source, ok = SelectPublishedPort(ports, RouteOptions{LabelPort: intPtr(3000)}, 0)
	if !ok || selected.HostPort != 3000 || source != "label" {
		t.Fatalf("expected label selected port 3000, got %#v source=%s ok=%t", selected, source, ok)
	}
}

func intPtr(v int) *int { return &v }
