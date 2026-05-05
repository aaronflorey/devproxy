package devproxy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/mochaka/devproxy/internal/config"
	"github.com/mochaka/devproxy/internal/daemon"
	"github.com/spf13/cobra"
)

func newDaemonCommand() *cobra.Command {
	var socketPath string
	var httpAddress string
	var httpsAddress string

	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run devproxy foreground daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			cfg := loadedCfg
			if cfg.DomainSuffix == "" {
				cfg = config.DefaultConfig()
			}

			app := daemon.NewApp(daemon.AppConfig{
				AdminSocketPath: socketPath,
				HTTPAddress:     httpAddress,
				HTTPSAddress:    httpsAddress,
				Config:          cfg,
				DockerPing:      dockerPing,
				EnsureMKCert:    daemon.DefaultEnsureMKCert,
			})
			defer func() { _ = app.Close(context.Background()) }()

			if err := app.Run(ctx); err != nil && err != context.Canceled {
				return fmt.Errorf("daemon startup failed: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&socketPath, "admin-socket", "/tmp/devproxy/admin.sock", "path to admin API unix socket")
	cmd.Flags().StringVar(&httpAddress, "http-address", "127.0.0.1:80", "HTTP listener bind address")
	cmd.Flags().StringVar(&httpsAddress, "https-address", "127.0.0.1:443", "HTTPS listener bind address")
	return cmd
}

func dockerPing(context.Context) error {
	cmd := exec.Command("docker", "info")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker info failed: %w: %s", err, string(output))
	}
	return nil
}
