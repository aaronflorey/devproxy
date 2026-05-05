package devproxy

import (
	"context"
	"fmt"

	"github.com/mochaka/devproxy/internal/admin"
	"github.com/mochaka/devproxy/internal/adminapi"
	"github.com/mochaka/devproxy/internal/doctor"
	"github.com/spf13/cobra"
)

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
			checker := doctor.NewChecker(doctor.Dependencies{
				CheckAdminSocket: func(ctx context.Context) error {
					client := adminapi.NewClient(socketPath)
					_, err := client.Status(ctx)
					return err
				},
				ReadNetworkHealth: func(ctx context.Context) (admin.NetworkRuntimeHealth, error) {
					client := adminapi.NewClient(socketPath)
					status, err := client.Status(ctx)
					if err != nil {
						return admin.NetworkRuntimeHealth{}, err
					}
					return admin.NetworkRuntimeHealth{DNS: admin.ListenerStatus{Enabled: true, Bound: status.DNS.Healthy, BindAddress: "127.0.0.1:53535"}, HTTP: status.HTTP, HTTPS: status.HTTPS, Paused: status.Paused, CertificateReady: status.CertificateReady, ManagedSuffix: status.DNS.ManagedSuffix}, nil
				},
			})
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
			return nil
		},
	}
	cmd.Flags().StringVar(&socketPath, "admin-socket", "/tmp/devproxy/admin.sock", "path to admin API unix socket")
	cmd.Flags().StringVar(&exampleHost, "example-host", "", "example managed hostname used for DNS resolution check")
	return cmd
}
