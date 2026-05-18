package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/mochaka/devproxy/internal/admin"
	"github.com/mochaka/devproxy/internal/adminapi"
)

const (
	DefaultListenAddress = "127.0.0.1:45831"
	errDaemonUnreachable = "DevProxy can’t reach the daemon right now. Ensure the daemon is running, then select Run Doctor for repair guidance."
)

type adminClient interface {
	Status(context.Context) (admin.StatusView, error)
	Routes(context.Context) ([]admin.RouteView, error)
	Logs(context.Context) ([]admin.LogEvent, error)
	Doctor(context.Context) (admin.DoctorView, error)
	Refresh(context.Context, string) (adminapi.RefreshResponse, error)
}

type Config struct {
	ListenAddress string
	Client        adminClient
}

type Server struct {
	listenAddress string
	client        adminClient
	templates     *templateSet
	mux           *http.ServeMux
}

type templateSet struct {
	root *templateExecutor
}

type templateExecutor struct{}

func NewServer(cfg Config) *Server {
	listen := cfg.ListenAddress
	if listen == "" {
		listen = DefaultListenAddress
	}
	if cfg.Client == nil {
		panic("dashboard admin client is required")
	}

	tmpl, err := parseTemplates()
	if err != nil {
		panic(err)
	}
	staticAssets, err := staticFS()
	if err != nil {
		panic(err)
	}

	s := &Server{listenAddress: listen, client: cfg.Client, templates: &templateSet{root: &templateExecutor{}}, mux: http.NewServeMux()}
	s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticAssets))))
	s.mux.HandleFunc("/", s.handleDashboard(tmpl))
	s.mux.HandleFunc("/routes", s.handleRoutesPage(tmpl))
	s.mux.HandleFunc("/logs", s.handleLogs(tmpl))
	s.mux.HandleFunc("/doctor", s.handleDoctorPage(tmpl))
	s.mux.HandleFunc("/status", s.handleStatusPage(tmpl))
	s.mux.HandleFunc("/actions/refresh", s.handleRefresh)

	// JSON polling endpoints for live updates
	s.mux.HandleFunc("/api/dashboard", s.handleAPIDashboard)
	s.mux.HandleFunc("/api/routes", s.handleAPIRoutes)
	s.mux.HandleFunc("/api/logs", s.handleAPILogs)
	s.mux.HandleFunc("/api/doctor", s.handleAPIDoctor)
	s.mux.HandleFunc("/api/status", s.handleAPIStatus)
	return s
}

func ValidateListenAddress(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid listen address %q: %w", address, err)
	}
	if host != "127.0.0.1" && host != "localhost" {
		return fmt.Errorf("dashboard listen host must be localhost or 127.0.0.1")
	}
	return nil
}

func (s *Server) ListenAddress() string { return s.listenAddress }
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) Run(ctx context.Context) error {
	if err := ValidateListenAddress(s.listenAddress); err != nil {
		return err
	}
	server := &http.Server{Addr: s.listenAddress, Handler: s.mux}
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// --- Page data structs ---

type pageBase struct {
	ActivePage  string
	DaemonError string
	Flash       string
}

type dashboardPageData struct {
	pageBase
	Status          admin.StatusView
	Routes          []admin.RouteView
	RecentConflicts []admin.LogEvent
	RecentErrors    []admin.LogEvent
	NoActiveRoutes  bool
}

type routesPageData struct {
	pageBase
	Routes         []admin.RouteView
	NoActiveRoutes bool
}

type logsPageData struct {
	pageBase
	Logs             []admin.LogEvent
	Errors           []admin.LogEvent
	ApprovedErrorMsg string
}

type doctorPageData struct {
	pageBase
	Doctor admin.DoctorView
}

type statusPageData struct {
	pageBase
	Status admin.StatusView
}

// --- Template rendering ---

func (s *Server) handleDashboard(tmpl interface {
	ExecuteTemplate(io.Writer, string, any) error
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := dashboardPageData{pageBase: pageBase{ActivePage: "dashboard", Flash: r.URL.Query().Get("flash")}}
		status, err := s.client.Status(r.Context())
		if err != nil {
			data.DaemonError = errDaemonUnreachable
			_ = tmpl.ExecuteTemplate(w, "dashboard.html.tmpl", data)
			return
		}
		data.Status = status
		routes, routesErr := s.client.Routes(r.Context())
		if routesErr == nil {
			sort.Slice(routes, func(i, j int) bool { return routes[i].Hostname < routes[j].Hostname })
			data.Routes = routes
			data.NoActiveRoutes = len(routes) == 0
		}
		logs, logsErr := s.client.Logs(r.Context())
		if logsErr == nil {
			for _, entry := range logs {
				switch strings.ToLower(entry.Type) {
				case "conflict":
					data.RecentConflicts = append(data.RecentConflicts, entry)
				case "error", "warning":
					data.RecentErrors = append(data.RecentErrors, entry)
				}
			}
		}
		_ = tmpl.ExecuteTemplate(w, "dashboard.html.tmpl", data)
	}
}

func (s *Server) handleRoutesPage(tmpl interface {
	ExecuteTemplate(io.Writer, string, any) error
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/routes" {
			http.NotFound(w, r)
			return
		}
		data := routesPageData{pageBase: pageBase{ActivePage: "routes"}}
		routes, err := s.client.Routes(r.Context())
		if err != nil {
			data.DaemonError = errDaemonUnreachable
			_ = tmpl.ExecuteTemplate(w, "routes.html.tmpl", data)
			return
		}
		sort.Slice(routes, func(i, j int) bool { return routes[i].Hostname < routes[j].Hostname })
		data.Routes = routes
		data.NoActiveRoutes = len(routes) == 0
		_ = tmpl.ExecuteTemplate(w, "routes.html.tmpl", data)
	}
}

func (s *Server) handleLogs(tmpl interface {
	ExecuteTemplate(io.Writer, string, any) error
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/logs" {
			http.NotFound(w, r)
			return
		}
		data := logsPageData{pageBase: pageBase{ActivePage: "logs"}, ApprovedErrorMsg: errDaemonUnreachable}
		logs, err := s.client.Logs(r.Context())
		if err == nil {
			data.Logs = logs
			for _, entry := range logs {
				typ := strings.ToLower(entry.Type)
				if typ == "error" || typ == "warning" {
					data.Errors = append(data.Errors, entry)
				}
			}
		}
		_ = tmpl.ExecuteTemplate(w, "logs.html.tmpl", data)
	}
}

func (s *Server) handleDoctorPage(tmpl interface {
	ExecuteTemplate(io.Writer, string, any) error
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/doctor" {
			http.NotFound(w, r)
			return
		}
		data := doctorPageData{pageBase: pageBase{ActivePage: "doctor"}}
		doctor, err := s.client.Doctor(r.Context())
		if err != nil {
			data.DaemonError = errDaemonUnreachable
			_ = tmpl.ExecuteTemplate(w, "doctor.html.tmpl", data)
			return
		}
		data.Doctor = doctor
		_ = tmpl.ExecuteTemplate(w, "doctor.html.tmpl", data)
	}
}

func (s *Server) handleStatusPage(tmpl interface {
	ExecuteTemplate(io.Writer, string, any) error
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			http.NotFound(w, r)
			return
		}
		data := statusPageData{pageBase: pageBase{ActivePage: "status"}}
		status, err := s.client.Status(r.Context())
		if err != nil {
			data.DaemonError = errDaemonUnreachable
			_ = tmpl.ExecuteTemplate(w, "status.html.tmpl", data)
			return
		}
		data.Status = status
		_ = tmpl.ExecuteTemplate(w, "status.html.tmpl", data)
	}
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	flash := "Routes refreshed"
	if _, err := s.client.Refresh(r.Context(), "dashboard refresh action"); err != nil {
		flash = "Refresh failed"
	}
	http.Redirect(w, r, "/?flash="+url.QueryEscape(flash), http.StatusSeeOther)
}

// --- JSON polling handlers ---

type apiResponse struct {
	DaemonError string `json:"daemon_error,omitempty"`
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) handleAPIDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiResponse{DaemonError: "method not allowed"})
		return
	}
	data := struct {
		apiResponse
		Status    admin.StatusView  `json:"status,omitempty"`
		Routes    []admin.RouteView `json:"routes,omitempty"`
		Conflicts []admin.LogEvent  `json:"conflicts,omitempty"`
		Errors    []admin.LogEvent  `json:"errors,omitempty"`
	}{}
	status, err := s.client.Status(r.Context())
	if err != nil {
		data.DaemonError = errDaemonUnreachable
		writeJSON(w, http.StatusOK, data)
		return
	}
	data.Status = status
	routes, routesErr := s.client.Routes(r.Context())
	if routesErr == nil {
		sort.Slice(routes, func(i, j int) bool { return routes[i].Hostname < routes[j].Hostname })
		data.Routes = routes
	}
	logs, logsErr := s.client.Logs(r.Context())
	if logsErr == nil {
		for _, entry := range logs {
			switch strings.ToLower(entry.Type) {
			case "conflict":
				data.Conflicts = append(data.Conflicts, entry)
			case "error", "warning":
				data.Errors = append(data.Errors, entry)
			}
		}
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleAPIRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiResponse{DaemonError: "method not allowed"})
		return
	}
	data := struct {
		apiResponse
		Routes []admin.RouteView `json:"routes,omitempty"`
	}{}
	routes, err := s.client.Routes(r.Context())
	if err != nil {
		data.DaemonError = errDaemonUnreachable
		writeJSON(w, http.StatusOK, data)
		return
	}
	sort.Slice(routes, func(i, j int) bool { return routes[i].Hostname < routes[j].Hostname })
	data.Routes = routes
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleAPILogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiResponse{DaemonError: "method not allowed"})
		return
	}
	data := struct {
		apiResponse
		Logs   []admin.LogEvent `json:"logs,omitempty"`
		Errors []admin.LogEvent `json:"errors,omitempty"`
	}{}
	logs, err := s.client.Logs(r.Context())
	if err != nil {
		data.DaemonError = errDaemonUnreachable
		writeJSON(w, http.StatusOK, data)
		return
	}
	data.Logs = logs
	for _, entry := range logs {
		typ := strings.ToLower(entry.Type)
		if typ == "error" || typ == "warning" {
			data.Errors = append(data.Errors, entry)
		}
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleAPIDoctor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiResponse{DaemonError: "method not allowed"})
		return
	}
	data := struct {
		apiResponse
		Doctor admin.DoctorView `json:"doctor,omitempty"`
	}{}
	doctor, err := s.client.Doctor(r.Context())
	if err != nil {
		data.DaemonError = errDaemonUnreachable
		writeJSON(w, http.StatusOK, data)
		return
	}
	data.Doctor = doctor
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiResponse{DaemonError: "method not allowed"})
		return
	}
	data := struct {
		apiResponse
		Status admin.StatusView `json:"status,omitempty"`
	}{}
	status, err := s.client.Status(r.Context())
	if err != nil {
		data.DaemonError = errDaemonUnreachable
		writeJSON(w, http.StatusOK, data)
		return
	}
	data.Status = status
	writeJSON(w, http.StatusOK, data)
}
