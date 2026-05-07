package devproxy

import (
	"fmt"

	"github.com/mochaka/devproxy/internal/adminapi"
	"github.com/spf13/cobra"
)

func init() {
	registerCommandFactory(newLogsCommand)
}

func newLogsCommand() *cobra.Command {
	var socketPath string
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Print current daemon-session log events",
		Long:  "Print current daemon-session log events from the local admin API. Persisted history is not available in v1.",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			client := adminapi.NewClient(socketPath)
			events, err := client.Logs(cmd.Context())
			if err != nil {
				return err
			}
			issues, err := client.Issues(cmd.Context())
			if err != nil {
				return err
			}
			for _, event := range events {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s [%s] host=%s state=%s upstream=%s:%d %s\n",
					event.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
					event.Type,
					event.Hostname,
					event.HandlingState,
					event.UpstreamScheme,
					event.UpstreamPort,
					event.Message,
				)
				if err != nil {
					return err
				}
			}
			for _, issue := range issues {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s [issue] role=%s action=%s %s\n",
					issue.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
					issue.Role,
					issue.Action,
					issue.Message,
				)
				if err != nil {
					return err
				}
			}
			if len(events) == 0 && len(issues) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "no current daemon-session events recorded")
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&socketPath, "admin-socket", "/tmp/devproxy/admin.sock", "path to admin API unix socket")
	return cmd
}
