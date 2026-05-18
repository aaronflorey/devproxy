package devproxy

import (
	"runtime"

	"github.com/mochaka/devproxy/internal/adminapi"
	"github.com/mochaka/devproxy/internal/menubar"
	"github.com/spf13/cobra"
)

func init() {
	registerCommandFactory(newMenubarCommand)
}

func newMenubarCommand() *cobra.Command {
	var socketPath string
	cmd := &cobra.Command{
		Use:   "menubar",
		Short: "Run the DevProxy macOS menu bar runtime",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			runtime.LockOSThread()
			client := adminapi.NewClient(socketPath)
			return menubar.Run(cmd.Context(), client, menubar.NewOpener())
		},
	}
	cmd.Flags().StringVar(&socketPath, "admin-socket", "/tmp/devproxy/admin.sock", "path to admin API unix socket")
	return cmd
}
