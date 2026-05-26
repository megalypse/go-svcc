package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/megalypse/go-svc-cluster/internal/components"
	"github.com/megalypse/go-svc-cluster/internal/domain/impl/cluster"
)

func NewViewSelectCluster() *SelectClusterView {
	var clusterNames []string
	clusters, _ := cluster.GetClusters()

	for _, c := range clusters {
		clusterNames = append(clusterNames, c.Name)
	}

	return &SelectClusterView{
		clusterList: components.NewListSelector(clusterNames),
	}
}

type SelectClusterView struct {
	clusterList *components.ListSelector
}

func (s *SelectClusterView) Init() tea.Cmd {
	return nil
}

func (s *SelectClusterView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	s.clusterList.Cursor.Update(msg)
	return s, nil
}

func (s *SelectClusterView) View() string {
	return s.clusterList.Render()
}
