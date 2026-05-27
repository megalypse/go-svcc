package views

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/megalypse/go-svc-cluster/internal/components"
	"github.com/megalypse/go-svc-cluster/internal/domain/impl"
)

func NewViewSelectCluster(ctx context.Context) *SelectClusterView {
	var clusterNames []string
	clusters, _ := impl.GetClusters()

	for _, c := range clusters {
		clusterNames = append(clusterNames, c.Name)
	}

	return &SelectClusterView{
		ctx:         ctx,
		clusterList: components.NewListSelector(clusterNames),
	}
}

type SelectClusterView struct {
	ctx         context.Context
	clusterList *components.ListSelector
}

func (s *SelectClusterView) Init() tea.Cmd {
	return nil
}

func (s *SelectClusterView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	s.clusterList.Cursor.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			nextView := NewViewStartCluster(s.ctx, s.clusterList.Cursor.Cursor())
			return nextView, nextView.Init()
		case "ctrl+c", "esc":
			return s, tea.Quit
		}
	}

	return s, nil
}

func (s *SelectClusterView) View() string {
	return s.clusterList.Render()
}
