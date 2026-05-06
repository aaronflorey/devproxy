package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mochaka/devproxy/internal/admin"
	"github.com/mochaka/devproxy/internal/routing"
)

type StateSnapshot struct {
	Snapshot routing.Snapshot
	Status   admin.StatusView
	Doctor   admin.DoctorView
	Logs     []admin.LogEvent
	Issues   []admin.SessionIssue
}

type StateProvider func() StateSnapshot

type RefreshFunc func(context.Context, string) error
type SetRoutingPausedFunc func(context.Context, bool) error
type StartupStatusFunc func(context.Context) ([]admin.StartupRoleStatus, error)
type SetStartupEnabledFunc func(context.Context, string, bool) (admin.StartupRoleStatus, error)

type ServerConfig struct {
	SocketPath        string
	State             StateProvider
	Refresh           RefreshFunc
	SetRoutingPaused  SetRoutingPausedFunc
	StartupStatus     StartupStatusFunc
	SetStartupEnabled SetStartupEnabledFunc
}

type Server struct {
	socketPath        string
	state             StateProvider
	refresh           RefreshFunc
	setRoutingPaused  SetRoutingPausedFunc
	startupStatus     StartupStatusFunc
	setStartupEnabled SetStartupEnabledFunc
	httpServer        *http.Server
	listener          net.Listener
}

func NewServer(cfg ServerConfig) (*Server, error) {
	if cfg.SocketPath == "" {
		return nil, fmt.Errorf("admin socket path is required")
	}
	state := cfg.State
	if state == nil {
		state = func() StateSnapshot { return StateSnapshot{} }
	}
	refresh := cfg.Refresh
	if refresh == nil {
		refresh = func(context.Context, string) error { return nil }
	}
	setRoutingPaused := cfg.SetRoutingPaused
	if setRoutingPaused == nil {
		setRoutingPaused = func(context.Context, bool) error { return nil }
	}
	startupStatus := cfg.StartupStatus
	if startupStatus == nil {
		startupStatus = func(context.Context) ([]admin.StartupRoleStatus, error) {
			roles := state().Status.StartupRoles
			if len(roles) > 0 {
				return roles, nil
			}
			return []admin.StartupRoleStatus{
				{Role: "daemon", Domain: "system", Label: "com.devproxy.daemon", Installed: true, Running: true, Toggleable: false, StatusMessage: "Managed by system launchd"},
				{Role: "menubar", Domain: "gui", Label: "com.devproxy.menubar", Installed: false, Running: false, Toggleable: true, StatusMessage: "Not configured"},
			}, nil
		}
	}
	setStartupEnabled := cfg.SetStartupEnabled
	if setStartupEnabled == nil {
		setStartupEnabled = func(_ context.Context, role string, _ bool) (admin.StartupRoleStatus, error) {
			return admin.StartupRoleStatus{Role: role}, nil
		}
	}

	return &Server{socketPath: cfg.SocketPath, state: state, refresh: refresh, setRoutingPaused: setRoutingPaused, startupStatus: startupStatus, setStartupEnabled: setStartupEnabled}, nil
}

func (s *Server) SetRefreshFunc(refresh RefreshFunc) {
	if refresh == nil {
		return
	}
	s.refresh = refresh
}

func (s *Server) SetRoutingPauseResume(fn SetRoutingPausedFunc) {
	if fn == nil {
		return
	}
	s.setRoutingPaused = fn
}

func (s *Server) Start() error {
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0o755); err != nil {
		return fmt.Errorf("create admin socket parent: %w", err)
	}
	if err := removeStaleSocket(s.socketPath); err != nil {
		return err
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen admin socket %q: %w", s.socketPath, err)
	}
	if err := os.Chmod(s.socketPath, 0o600); err != nil {
		_ = listener.Close()
		return fmt.Errorf("set admin socket permissions: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/routes", s.handleRoutes)
	mux.HandleFunc("/doctor", s.handleDoctor)
	mux.HandleFunc("/logs", s.handleLogs)
	mux.HandleFunc("/issues", s.handleIssues)
	mux.HandleFunc("/refresh", s.handleRefresh)
	mux.HandleFunc("/routing/pause", s.handleRoutingPause)
	mux.HandleFunc("/routing/resume", s.handleRoutingResume)
	mux.HandleFunc("/startup", s.handleStartup)

	s.listener = listener
	s.httpServer = &http.Server{Handler: mux}
	go func() { _ = s.httpServer.Serve(listener) }()
	return nil
}

func (s *Server) Close(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	err := s.httpServer.Shutdown(ctx)
	_ = os.Remove(s.socketPath)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func removeStaleSocket(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat admin socket path %q: %w", path, err)
	}
	if info.Mode().Type() == os.ModeSocket || info.Mode().IsRegular() {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove stale admin socket %q: %w", path, err)
		}
		return nil
	}
	return fmt.Errorf("admin socket path %q already exists and is not removable type: %s", path, info.Mode())
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, StatusResponse{Status: s.state().Status})
}

func (s *Server) handleRoutes(w http.ResponseWriter, _ *http.Request) {
	state := s.state()
	httpsReady := state.Status.CertificateReady && state.Status.HTTPS.Bound
	writeJSON(w, http.StatusOK, RoutesResponse{Routes: admin.RoutesFromSnapshotWithRuntime(state.Snapshot, httpsReady)})
}

func (s *Server) handleDoctor(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, DoctorResponse{Doctor: s.state().Doctor})
}

func (s *Server) handleLogs(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, LogsResponse{Events: s.state().Logs})
}

func (s *Server) handleIssues(w http.ResponseWriter, _ *http.Request) {
	issues := admin.BuildSessionIssues(s.state().Issues)
	writeJSON(w, http.StatusOK, SessionIssuesResponse{Issues: issues})
}

func (s *Server) handleRefresh(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	var payload RefreshRequest
	_ = json.NewDecoder(req.Body).Decode(&payload)
	if err := s.refresh(req.Context(), payload.Reason); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, RefreshResponse{Accepted: true, Refreshed: false, At: time.Now().UTC(), Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, RefreshResponse{Accepted: true, Refreshed: true, At: time.Now().UTC()})
}

func (s *Server) handleRoutingPause(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	if err := s.setRoutingPaused(req.Context(), true); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, RoutingPauseResumeResponse{Paused: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, RoutingPauseResumeResponse{Paused: true})
}

func (s *Server) handleRoutingResume(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}
	if err := s.setRoutingPaused(req.Context(), false); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, RoutingPauseResumeResponse{Paused: true, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, RoutingPauseResumeResponse{Paused: false})
}

func (s *Server) handleStartup(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		roles, err := s.startupStatus(req.Context())
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: err.Error()})
			return
		}
		resp := StartupStatusResponse{Roles: make([]StartupRoleStatus, 0, len(roles))}
		for _, role := range roles {
			resp.Roles = append(resp.Roles, StartupRoleStatus{Role: role.Role, Domain: role.Domain, Label: role.Label, Installed: role.Installed, Running: role.Running, Toggleable: role.Toggleable, StatusMessage: role.StatusMessage})
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	if req.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}

	var payload StartupToggleRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid startup toggle request"})
		return
	}
	if payload.Role != "daemon" && payload.Role != "menubar" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "role must be daemon or menubar"})
		return
	}
	result, err := s.setStartupEnabled(req.Context(), payload.Role, payload.Enabled)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, StartupToggleResponse{Role: payload.Role, Enabled: payload.Enabled, AffectedRole: payload.Role, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, StartupToggleResponse{Role: result.Role, Enabled: payload.Enabled, AffectedRole: result.Role})
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
