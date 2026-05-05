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
}

type StateProvider func() StateSnapshot

type RefreshFunc func(context.Context, string) error

type ServerConfig struct {
	SocketPath string
	State      StateProvider
	Refresh    RefreshFunc
}

type Server struct {
	socketPath string
	state      StateProvider
	refresh    RefreshFunc
	httpServer *http.Server
	listener   net.Listener
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
	return &Server{socketPath: cfg.SocketPath, state: state, refresh: refresh}, nil
}

func (s *Server) SetRefreshFunc(refresh RefreshFunc) {
	if refresh == nil {
		return
	}
	s.refresh = refresh
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
	mux.HandleFunc("/refresh", s.handleRefresh)

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
	writeJSON(w, http.StatusOK, RoutesResponse{Routes: admin.RoutesFromSnapshot(s.state().Snapshot)})
}

func (s *Server) handleDoctor(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, DoctorResponse{Doctor: s.state().Doctor})
}

func (s *Server) handleLogs(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, LogsResponse{Events: s.state().Logs})
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

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
