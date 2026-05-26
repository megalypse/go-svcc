package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Root struct {
	router tea.Model
}

func (r Root) Init() tea.Cmd {
	return nil
}

func (r Root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return r, nil
}

func (r Root) View() string {
	return lipgloss.NewStyle().Render("Service Cluster")
}
