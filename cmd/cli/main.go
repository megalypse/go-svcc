package cli

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/megalypse/go-svc-cluster/internal/views"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "",
	Short: "Interactive RQM workflow runner",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(views.Root{})

		_, err := p.Run()
		return err
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
