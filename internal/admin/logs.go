package admin

import (
	"time"

	"github.com/mochaka/devproxy/internal/routing"
)

type LogEvent struct {
	Timestamp time.Time
	Type      string
	Message   string
	Hostname  string
	HandlingState string
	UpstreamScheme string
	UpstreamPort int
}

func BuildSessionEvents(snapshot routing.Snapshot) []LogEvent {
	result := []LogEvent{}
	now := time.Now().UTC()
	for _, w := range snapshot.Warnings {
		result = append(result, LogEvent{Timestamp: now, Type: "warning", Message: w.Message, Hostname: ""})
	}
	for _, c := range snapshot.Conflicts {
		result = append(result, LogEvent{Timestamp: now, Type: "conflict", Message: c.Reason, Hostname: c.Hostname, HandlingState: "conflict"})
	}
	for host, route := range snapshot.Routes {
		result = append(result, LogEvent{Timestamp: now, Type: "route", Message: "active route", Hostname: host, HandlingState: "proxy", UpstreamScheme: route.Upstream.Scheme, UpstreamPort: route.Upstream.Port})
	}
	return result
}
