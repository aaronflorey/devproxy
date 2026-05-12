package devproxy

import (
	"context"
	"fmt"
	"strings"

	"github.com/mochaka/devproxy/internal/admin"
	"github.com/mochaka/devproxy/internal/doctor"
	"github.com/spf13/cobra"
)

var buildDoctorChecker = func(socketPath string) *doctor.Checker {
	client := newAdminClient(socketPath)
	return doctor.NewChecker(doctor.Dependencies{
		CheckAdminSocket: func(ctx context.Context) error {
			_, err := client.Status(ctx)
			return err
		},
		ReadNetworkHealth: func(ctx context.Context) (admin.NetworkRuntimeHealth, error) {
			status, err := client.Status(ctx)
			if err != nil {
				return admin.NetworkRuntimeHealth{}, err
			}
			return admin.NetworkRuntimeHealth{DNS: admin.ListenerStatus{Enabled: true, Bound: status.DNS.Healthy, BindAddress: "127.0.0.1:53535"}, HTTP: status.HTTP, HTTPS: status.HTTPS, Paused: status.Paused, CertificateReady: status.CertificateReady, ManagedSuffix: status.DNS.ManagedSuffix}, nil
		},
	})
}

func init() {
	registerCommandFactory(newDoctorCommand)
}

func newDoctorCommand() *cobra.Command {
	var socketPath string
	var exampleHost string
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run diagnostic checks for devproxy runtime",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			if exampleHost == "" {
				exampleHost = "example." + loadedCfg.DomainSuffix
			}
			client := newAdminClient(socketPath)
			checker := buildDoctorChecker(socketPath)
			report := checker.Run(cmd.Context(), exampleHost)
			for _, check := range report.Checks {
				status := "ok"
				if !check.OK {
					status = "fail"
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", check.Name, status, check.Message); err != nil {
					return err
				}
			}
			view, err := client.Doctor(cmd.Context())
			if err != nil {
				return err
			}
			if err := writeDoctorDetails(cmd, view); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&socketPath, "admin-socket", "/tmp/devproxy/admin.sock", "path to admin API unix socket")
	cmd.Flags().StringVar(&exampleHost, "example-host", "", "example managed hostname used for DNS resolution check")
	return cmd
}

func writeDoctorDetails(cmd *cobra.Command, view admin.DoctorView) error {
	if len(view.Conflicts) > 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "snapshot_conflicts:"); err != nil {
			return err
		}
		for _, conflict := range view.Conflicts {
			losers := make([]string, 0, len(conflict.Losers))
			for _, loser := range conflict.Losers {
				losers = append(losers, loser.ContainerName)
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s winner=%s losers=%s reason=%s\n", conflict.Hostname, conflict.Winner.ContainerName, strings.Join(losers, ","), conflict.Reason); err != nil {
				return err
			}
		}
	}
	if len(view.Warnings) > 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "snapshot_warnings:"); err != nil {
			return err
		}
		for _, warning := range view.Warnings {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s code=%s field=%s container=%s source=%s\n", warning.Message, warning.Code, warning.Field, warning.Container, warning.Source); err != nil {
				return err
			}
		}
	}
	return nil
}
