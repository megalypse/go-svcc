package views

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/megalypse/go-svc-cluster/internal/domain/impl"
	"github.com/megalypse/go-svc-cluster/internal/domain/models"
	"github.com/megalypse/go-svc-cluster/internal/factory"
)

func NewViewStartCluster(ctx context.Context, clusterId int) *StartClusterView {
	ctx, cancel := context.WithCancel(ctx)
	clusters, _ := impl.GetClusters()
	cluster := clusters[clusterId]
	var nodes []*models.Node
	for _, node := range cluster.Nodes {
		nodes = append(nodes, node)
	}

	clusterChan := factory.GetServiceStartCluster().StartCluster(ctx, clusterId)
	s := spinner.New()
	s.Spinner = spinner.Dot

	return &StartClusterView{
		ctx:         ctx,
		cancel:      cancel,
		clusterId:   clusterId,
		nodes:       nodes,
		nodeStatus:  make([]*impl.ClusterInfo, len(nodes)),
		clusterChan: clusterChan,
		spinner:     s,
	}
}

type StartClusterView struct {
	nodes       []*models.Node
	nodeStatus  []*impl.ClusterInfo
	ctx         context.Context
	cancel      context.CancelFunc
	clusterId   int
	clusterChan <-chan *impl.ClusterInfo
	spinner     spinner.Model
}

type clusterInfoMsg struct {
	info *impl.ClusterInfo
}

type clusterDoneMsg struct{}

func waitClusterInfo(ctx context.Context, ch <-chan *impl.ClusterInfo) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return clusterDoneMsg{}

		case info, ok := <-ch:
			if !ok {
				return clusterDoneMsg{}
			}

			if info.Error != nil {
				return clusterDoneMsg{}
			}

			return clusterInfoMsg{info: info}
		}
	}
}

func (s *StartClusterView) Init() tea.Cmd {
	return tea.Batch(
		s.spinner.Tick,
		waitClusterInfo(s.ctx, s.clusterChan),
	)
}

func (s *StartClusterView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	case clusterInfoMsg:
		s.nodeStatus[msg.info.NodeId] = msg.info

		return s, waitClusterInfo(s.ctx, s.clusterChan)
	case clusterDoneMsg:
		s.cancel()
		return s, tea.Quit
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			s.cancel()
			return s, tea.Quit
		}
	}

	return s, nil
}

var LoadingBlue = lipgloss.Color("#4287f5")
var SuccessGreen = lipgloss.Color("#16fa0a")

func (s *StartClusterView) View() string {
	render := strings.Builder{}

	for i, node := range s.nodes {
		prefix := func() string {
			if s.nodeStatus[i] == nil || s.nodeStatus[i].Loading {
				return lipgloss.NewStyle().Foreground(LoadingBlue).Render(s.spinner.View())
			}

			if s.nodeStatus[i].Error != nil {
				return "X"
			}

			return lipgloss.NewStyle().Foreground(SuccessGreen).Render("✓")
		}()

		render.WriteString(prefix)
		render.WriteString(" ")
		render.WriteString(node.Name)
		render.WriteString("\n")
	}

	return render.String()
}
