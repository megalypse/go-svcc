package views

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/megalypse/go-svc-cluster/internal/components"
)

type Root struct {
	router    tea.Model
	killWatch chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
	width     int
	height    int
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
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		r.width = msg.Width
		r.height = msg.Height
		msg = r.childWindowSizeMsg()
		var cmd tea.Cmd
		r.router, cmd = r.router.Update(msg)
		return r, cmd
	}

	previousRouter := r.router
	var cmd tea.Cmd
	r.router, cmd = r.router.Update(msg)
	if previousRouter != r.router && r.width > 0 && r.height > 0 {
		var resizeCmd tea.Cmd
		r.router, resizeCmd = r.router.Update(r.childWindowSizeMsg())
		cmd = tea.Batch(cmd, resizeCmd)
	}

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

func (r *Root) childWindowSizeMsg() tea.WindowSizeMsg {
	height := r.height - 2
	if height < 1 {
		height = 1
	}

	return tea.WindowSizeMsg{
		Width:  r.width,
		Height: height,
	}
}

func (r *Root) View() string {
	return lipgloss.NewStyle().Foreground(SuccessGreen).Render("SVCC") + components.LineSkip + r.router.View()
}
