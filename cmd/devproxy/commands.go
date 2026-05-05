package devproxy

import "github.com/spf13/cobra"

type commandFactory func() *cobra.Command

func registerCommands(root *cobra.Command, factories ...commandFactory) {
	for _, factory := range factories {
		if factory == nil {
			continue
		}
		root.AddCommand(factory())
	}
}
