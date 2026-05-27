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
	ctx       context.Context
	cancel    context.CancelFunc
}

func (r *Root) Init() tea.Cmd {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	r.ctx = ctx
	r.cancel = stop
	r.killWatch = make(chan struct{}, 1)
	r.router = NewViewSelectCluster(ctx)

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
		r.cancel()
		return r, tea.Quit
	default:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c":
				r.cancel()
				return r, tea.Quit
			}
		}
	}

	return r, cmd
}

func (r *Root) View() string {
	return r.router.View()
}
