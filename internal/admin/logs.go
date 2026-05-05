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
}

func BuildSessionEvents(snapshot routing.Snapshot) []LogEvent {
	result := []LogEvent{}
	now := time.Now().UTC()
	for _, w := range snapshot.Warnings {
		result = append(result, LogEvent{Timestamp: now, Type: "warning", Message: w.Message, Hostname: ""})
	}
	for _, c := range snapshot.Conflicts {
		result = append(result, LogEvent{Timestamp: now, Type: "conflict", Message: c.Reason, Hostname: c.Hostname})
	}
	return result
}
