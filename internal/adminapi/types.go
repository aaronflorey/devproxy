package adminapi

import (
	"time"

	"github.com/mochaka/devproxy/internal/admin"
)

type StatusResponse struct {
	Status admin.StatusView `json:"status"`
}

type RoutesResponse struct {
	Routes []admin.RouteView `json:"routes"`
}

type DoctorResponse struct {
	Doctor admin.DoctorView `json:"doctor"`
}

type LogsResponse struct {
	Events []admin.LogEvent `json:"events"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type RefreshRequest struct {
	Reason string `json:"reason,omitempty"`
}

type RefreshResponse struct {
	Accepted  bool      `json:"accepted"`
	Refreshed bool      `json:"refreshed"`
	At        time.Time `json:"at"`
	Error     string    `json:"error,omitempty"`
}

type RoutingPauseResumeResponse struct {
	Paused bool   `json:"paused"`
	Error  string `json:"error,omitempty"`
}

type StartupRoleStatus struct {
	Role          string `json:"role"`
	Domain        string `json:"domain"`
	Label         string `json:"label"`
	Installed     bool   `json:"installed"`
	Running       bool   `json:"running"`
	Toggleable    bool   `json:"toggleable"`
	StatusMessage string `json:"status_message"`
}

type StartupStatusResponse struct {
	Roles []StartupRoleStatus `json:"roles"`
}

type StartupToggleRequest struct {
	Role    string `json:"role"`
	Enabled bool   `json:"enabled"`
}

type StartupToggleResponse struct {
	Role         string `json:"role"`
	Enabled      bool   `json:"enabled"`
	AffectedRole string `json:"affected_role"`
	Error        string `json:"error,omitempty"`
}

type SessionIssuesResponse struct {
	Issues []admin.SessionIssue `json:"issues"`
}

type CommandFactory func() NamedCommand

type NamedCommand interface {
	Name() string
}

// RegisterCoreCommands centralizes root command registration so future plans
// can add new subcommands without modifying cmd/devproxy/root.go directly.
func RegisterCoreCommands(register func(CommandFactory)) {
	if register == nil {
		return
	}
	register(NewPrintConfigCommand)
	register(NewDaemonCommand)
}

func NewPrintConfigCommand() NamedCommand {
	return namedCommand{name: "print-config"}
}

func NewDaemonCommand() NamedCommand {
	return namedCommand{name: "daemon"}
}

type namedCommand struct {
	name string
}

func (c namedCommand) Name() string {
	return c.name
}
