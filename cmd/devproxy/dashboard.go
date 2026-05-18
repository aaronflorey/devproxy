package devproxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/mochaka/devproxy/internal/adminapi"
	"github.com/mochaka/devproxy/internal/dashboard"
	"github.com/spf13/cobra"
)

var dashboardProbe = dashboardResponding
var openDashboardURL = openURL

func init() {
	registerCommandFactory(newDashboardCommand)
}

func newDashboardCommand() *cobra.Command {
	var socketPath string
	var listenAddress string
	var openBrowser bool
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Run the local DevProxy dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			if err := dashboard.ValidateListenAddress(listenAddress); err != nil {
				return err
			}
			shouldOpen := openBrowser || stdoutIsTTY()
			client := adminapi.NewClient(socketPath)
			srv := dashboard.NewServer(dashboard.Config{ListenAddress: listenAddress, Client: client})

			targetURL := "http://" + listenAddress
			if responding, err := dashboardProbe(cmd.Context(), targetURL); err != nil {
				return err
			} else if responding {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "dashboard already running on %s\n", targetURL)
				if shouldOpen {
					_ = openDashboardURL(targetURL)
				}
				return nil
			}
			if shouldOpen {
				go func() {
					time.Sleep(200 * time.Millisecond)
					_ = openDashboardURL(targetURL)
				}()
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "dashboard listening on %s\n", targetURL)
			if err := srv.Run(cmd.Context()); err != nil {
				if reused, reuseErr := reuseRunningDashboard(cmd.Context(), targetURL, err, cmd.OutOrStdout(), shouldOpen); reuseErr != nil {
					return reuseErr
				} else if reused {
					return nil
				}
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&socketPath, "admin-socket", "/tmp/devproxy/admin.sock", "path to admin API unix socket")
	cmd.Flags().StringVar(&listenAddress, "listen", dashboard.DefaultListenAddress, "dashboard listen address (localhost only)")
	cmd.Flags().BoolVar(&openBrowser, "open", false, "open dashboard in browser after startup")
	return cmd
}

func reuseRunningDashboard(ctx context.Context, targetURL string, runErr error, output io.Writer, openBrowser bool) (bool, error) {
	if !isAddressInUse(runErr) {
		return false, nil
	}
	responding, probeErr := dashboardProbe(ctx, targetURL)
	if probeErr != nil {
		return false, probeErr
	}
	if !responding {
		return false, runErr
	}
	_, _ = fmt.Fprintf(output, "dashboard already running on %s\n", targetURL)
	if openBrowser {
		_ = openDashboardURL(targetURL)
	}
	return true, nil
}

func isAddressInUse(err error) bool {
	if err == nil {
		return false
	}
	var opErr *net.OpError
	msg := strings.ToLower(err.Error())
	return (errors.As(err, &opErr) && strings.Contains(msg, "address already in use")) || strings.Contains(msg, "address already in use")
}

func dashboardResponding(ctx context.Context, targetURL string) (bool, error) {
	probeCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return false, err
	}
	return strings.Contains(string(body), "DevProxy Dashboard"), nil
}

func stdoutIsTTY() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func openURL(target string) error {
	if runtime.GOOS == "darwin" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return exec.CommandContext(ctx, "open", target).Run()
	}
	return nil
}
