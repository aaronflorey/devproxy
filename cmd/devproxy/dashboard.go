package devproxy

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/mochaka/devproxy/internal/adminapi"
	"github.com/mochaka/devproxy/internal/dashboard"
	"github.com/spf13/cobra"
)

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
			client := adminapi.NewClient(socketPath)
			srv := dashboard.NewServer(dashboard.Config{ListenAddress: listenAddress, Client: client})

			if openBrowser {
				go func() {
					time.Sleep(200 * time.Millisecond)
					_ = openURL("http://" + listenAddress)
				}()
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "dashboard listening on http://%s\n", listenAddress)
			return srv.Run(cmd.Context())
		},
	}
	cmd.Flags().StringVar(&socketPath, "admin-socket", "/tmp/devproxy/admin.sock", "path to admin API unix socket")
	cmd.Flags().StringVar(&listenAddress, "listen", dashboard.DefaultListenAddress, "dashboard listen address (localhost only)")
	cmd.Flags().BoolVar(&openBrowser, "open", false, "open dashboard in browser after startup")
	return cmd
}

func openURL(target string) error {
	if runtime.GOOS == "darwin" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return exec.CommandContext(ctx, "open", target).Run()
	}
	return nil
}
