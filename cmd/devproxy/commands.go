package devproxy

import "github.com/spf13/cobra"

type commandFactory func() *cobra.Command

var registeredCommandFactories []commandFactory

func registerCommandFactory(factory commandFactory) {
	if factory == nil {
		return
	}
	registeredCommandFactories = append(registeredCommandFactories, factory)
}

func registerCommands(root *cobra.Command, factories ...commandFactory) {
	allFactories := append([]commandFactory{}, factories...)
	allFactories = append(allFactories, registeredCommandFactories...)
	for _, factory := range allFactories {
		if factory == nil {
			continue
		}
		root.AddCommand(factory())
	}
}
