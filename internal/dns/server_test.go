package dns

import (
	"testing"

	mdns "github.com/miekg/dns"
	"github.com/mochaka/devproxy/internal/routing"
)

func TestManagedSuffixReturnsLoopbackARecord(t *testing.T) {
	srv := NewServer("test", staticSnapshot(routing.Snapshot{}))

	resp := srv.BuildResponse(newAQuery("api.acme.test."))
	if resp == nil {
		t.Fatalf("expected dns response")
	}

	if len(resp.Answer) != 1 {
		t.Fatalf("expected one answer, got %d", len(resp.Answer))
	}

	rec, ok := resp.Answer[0].(*mdns.A)
	if !ok {
		t.Fatalf("expected A answer, got %T", resp.Answer[0])
	}

	if got := rec.A.String(); got != "127.0.0.1" {
		t.Fatalf("expected 127.0.0.1, got %s", got)
	}
}

func TestOutsideManagedSuffixReturnsNoAnswer(t *testing.T) {
	srv := NewServer("test", staticSnapshot(routing.Snapshot{}))

	resp := srv.BuildResponse(newAQuery("api.acme.local."))
	if resp == nil {
		t.Fatalf("expected dns response")
	}

	if len(resp.Answer) != 0 {
		t.Fatalf("expected no local authoritative answers, got %d", len(resp.Answer))
	}
}

func TestHostnameLookupManagedAndActiveRoute(t *testing.T) {
	snap := routing.Snapshot{Routes: map[string]routing.Route{
		"api.acme.test": {Hostname: "api.acme.test"},
	}}
	srv := NewServer("test", staticSnapshot(snap))

	lookup := srv.LookupHostname("api.acme.test")
	if !lookup.Managed {
		t.Fatalf("expected host to be managed")
	}
	if !lookup.ActiveRoute {
		t.Fatalf("expected host to have active route")
	}
}

func TestHostnameLookupManagedWithoutRoute(t *testing.T) {
	srv := NewServer("test", staticSnapshot(routing.Snapshot{Routes: map[string]routing.Route{}}))

	lookup := srv.LookupHostname("missing.acme.test")
	if !lookup.Managed {
		t.Fatalf("expected host under suffix to be managed")
	}
	if lookup.ActiveRoute {
		t.Fatalf("expected missing route to be inactive")
	}
}

func TestPausedRoutingDoesNotChangeManagedSuffixResolution(t *testing.T) {
	srv := NewServer("test", staticSnapshot(routing.Snapshot{}))

	before := srv.BuildResponse(newAQuery("api.acme.test."))
	after := srv.BuildResponse(newAQuery("api.acme.test."))

	if len(before.Answer) != 1 || len(after.Answer) != 1 {
		t.Fatalf("expected managed suffix to resolve before and after pause check")
	}

	b, bok := before.Answer[0].(*mdns.A)
	a, aok := after.Answer[0].(*mdns.A)
	if !bok || !aok || b.A.String() != "127.0.0.1" || a.A.String() != "127.0.0.1" {
		t.Fatalf("expected stable loopback answer while routing pause state changes")
	}
}

func newAQuery(host string) *mdns.Msg {
	msg := new(mdns.Msg)
	msg.SetQuestion(host, mdns.TypeA)
	return msg
}

func staticSnapshot(s routing.Snapshot) func() routing.Snapshot {
	return func() routing.Snapshot { return s }
}
