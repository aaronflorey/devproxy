package devproxy

import (
	"fmt"

	"github.com/mochaka/devproxy/internal/adminapi"
	"github.com/spf13/cobra"
)

func init() {
	registerCommandFactory(newStatusCommand)
}

func newStatusCommand() *cobra.Command {
	var socketPath string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show daemon health and route status",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			client := adminapi.NewClient(socketPath)
			status, err := client.Status(cmd.Context())
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "snapshot=%s active_routes=%d conflicts=%d warnings=%d paused=%t cert_ready=%t dns_healthy=%t managed_suffix=%s http_bound=%t https_bound=%t\n",
				status.SnapshotVersion,
				status.ActiveRoutes,
				status.Conflicts,
				status.Warnings,
				status.Paused,
				status.CertificateReady,
				status.DNS.Healthy,
				status.DNS.ManagedSuffix,
				status.HTTP.Bound,
				status.HTTPS.Bound,
			)
			return err
		},
	}
	cmd.Flags().StringVar(&socketPath, "admin-socket", "/tmp/devproxy/admin.sock", "path to admin API unix socket")
	return cmd
}
