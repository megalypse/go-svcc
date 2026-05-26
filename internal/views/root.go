package views

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

type Root struct {
	router    tea.Model
	killWatch chan struct{}
}

func (r *Root) Init() tea.Cmd {
	r.router = NewViewSelectCluster()

	ctx, _ := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	go func() {
		<-ctx.Done()
		r.killWatch <- struct{}{}
	}()

	return nil
}

func (r *Root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	r.router, cmd = r.router.Update(msg)

	select {
	case <-r.killWatch:
		return r, tea.Quit
	default:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c":
				return r, tea.Quit
			}
		}
	}

	return r, cmd
}

func (r *Root) View() string {
	return r.router.View()
}
