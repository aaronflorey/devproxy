package devproxy

import (
	"fmt"
	"strings"

	"github.com/mochaka/devproxy/internal/admin"
	"github.com/mochaka/devproxy/internal/adminapi"
	"github.com/spf13/cobra"
)

var newAdminClient = adminapi.NewClient

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
			client := newAdminClient(socketPath)
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
			if err != nil {
				return err
			}
			if err := writeStatusDetails(cmd, status); err != nil {
				return err
			}
			return err
		},
	}
	cmd.Flags().StringVar(&socketPath, "admin-socket", "/tmp/devproxy/admin.sock", "path to admin API unix socket")
	return cmd
}

func writeStatusDetails(cmd *cobra.Command, status admin.StatusView) error {
	if len(status.ConflictDetails) > 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "conflicts:"); err != nil {
			return err
		}
		for _, conflict := range status.ConflictDetails {
			losers := make([]string, 0, len(conflict.Losers))
			for _, loser := range conflict.Losers {
				losers = append(losers, loser.ContainerName)
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s winner=%s losers=%s reason=%s\n", conflict.Hostname, conflict.Winner.ContainerName, strings.Join(losers, ","), conflict.Reason); err != nil {
				return err
			}
		}
	}
	if len(status.WarningDetails) > 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "warnings:"); err != nil {
			return err
		}
		for _, warning := range status.WarningDetails {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s code=%s field=%s container=%s source=%s\n", warning.Message, warning.Code, warning.Field, warning.Container, warning.Source); err != nil {
				return err
			}
		}
	}
	return nil
}
