package devproxy

import (
	"fmt"

	"github.com/mochaka/devproxy/internal/adminapi"
	"github.com/spf13/cobra"
)

func init() {
	registerCommandFactory(newRefreshCommand)
}

func newRefreshCommand() *cobra.Command {
	var socketPath string
	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Trigger a full daemon container rescan",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			client := adminapi.NewClient(socketPath)
			resp, err := client.Refresh(cmd.Context(), "operator refresh command")
			if err != nil {
				return err
			}
			if resp.Error != "" {
				return fmt.Errorf("refresh failed: %s", resp.Error)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "refresh accepted=%t refreshed=%t at=%s\n", resp.Accepted, resp.Refreshed, resp.At.Format("2006-01-02T15:04:05Z07:00"))
			return err
		},
	}
	cmd.Flags().StringVar(&socketPath, "admin-socket", "/tmp/devproxy/admin.sock", "path to admin API unix socket")
	return cmd
}
