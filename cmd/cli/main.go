package cli

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/megalypse/go-svc-cluster/internal/views"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(&views.Root{})

		_, err := p.Run()
		return err
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
