package devproxy

import (
	"fmt"
	"strings"

	"github.com/mochaka/devproxy/internal/adminapi"
	"github.com/spf13/cobra"
)

func init() {
	registerCommandFactory(newRoutesCommand)
}

func newRoutesCommand() *cobra.Command {
	var socketPath string
	cmd := &cobra.Command{
		Use:   "routes",
		Short: "List active route mappings and conflict losers",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			client := adminapi.NewClient(socketPath)
			routes, err := client.Routes(cmd.Context())
			if err != nil {
				return err
			}
			for _, route := range routes {
				losers := "-"
				if len(route.Losers) > 0 {
					losers = strings.Join(route.Losers, ",")
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s -> %s://%s:%d winner=%s conflict=%t losers=%s\n",
					route.Hostname,
					route.UpstreamScheme,
					route.UpstreamHost,
					route.UpstreamPort,
					route.Winner,
					route.Conflict,
					losers,
				)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&socketPath, "admin-socket", "/tmp/devproxy/admin.sock", "path to admin API unix socket")
	return cmd
}
