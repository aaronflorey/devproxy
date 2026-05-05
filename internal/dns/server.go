package dns

import (
	"net"
	"strings"

	mdns "github.com/miekg/dns"
	"github.com/mochaka/devproxy/internal/routing"
)

type SnapshotReader func() routing.Snapshot

type Server struct {
	managedSuffix string
	readSnapshot  SnapshotReader
}

type HostnameLookup struct {
	Managed     bool
	ActiveRoute bool
	Route       routing.Route
}

func NewServer(managedSuffix string, readSnapshot SnapshotReader) *Server {
	return &Server{managedSuffix: normalizeSuffix(managedSuffix), readSnapshot: readSnapshot}
}

func (s *Server) BuildResponse(req *mdns.Msg) *mdns.Msg {
	resp := new(mdns.Msg)
	resp.SetReply(req)

	if req == nil || len(req.Question) == 0 {
		return resp
	}

	q := req.Question[0]
	if q.Qtype != mdns.TypeA {
		return resp
	}

	host := normalizeHost(q.Name)
	if !s.IsManagedHost(host) {
		return resp
	}

	resp.Authoritative = true
	resp.Answer = append(resp.Answer, &mdns.A{
		Hdr: mdns.RR_Header{Name: q.Name, Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: 30},
		A:   net.ParseIP("127.0.0.1").To4(),
	})

	return resp
}

func (s *Server) ServeDNS(w mdns.ResponseWriter, req *mdns.Msg) {
	_ = w.WriteMsg(s.BuildResponse(req))
}

func (s *Server) IsManagedHost(host string) bool {
	host = normalizeHost(host)
	if host == "" || s.managedSuffix == "" {
		return false
	}

	return host == s.managedSuffix || strings.HasSuffix(host, "."+s.managedSuffix)
}

func (s *Server) LookupHostname(host string) HostnameLookup {
	lookup := HostnameLookup{Managed: s.IsManagedHost(host)}
	if !lookup.Managed || s.readSnapshot == nil {
		return lookup
	}

	route, ok := s.readSnapshot().Routes[normalizeHost(host)]
	if !ok {
		return lookup
	}

	lookup.ActiveRoute = true
	lookup.Route = route
	return lookup
}

func normalizeSuffix(suffix string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(suffix)), ".")
}

func normalizeHost(host string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(host)), ".")
}
