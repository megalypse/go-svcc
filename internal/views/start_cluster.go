package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/megalypse/go-svc-cluster/internal/components"
	"github.com/megalypse/go-svc-cluster/internal/domain/impl"
	"github.com/megalypse/go-svc-cluster/internal/domain/models"
	"github.com/megalypse/go-svc-cluster/internal/factory"
)

const maxLogLines = 500

func NewViewStartCluster(ctx context.Context, clusterId int) *StartClusterView {
	parentCtx := ctx
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
		parentCtx:   parentCtx,
		cancel:      cancel,
		clusterId:   clusterId,
		nodes:       nodes,
		nodeStatus:  make([]*impl.ClusterInfo, len(nodes)),
		nodeLogs:    make([][]string, len(nodes)),
		cursor:      components.NewCursor(len(nodes) - 1),
		clusterChan: clusterChan,
		spinner:     s,
	}
}

type StartClusterView struct {
	nodes       []*models.Node
	nodeStatus  []*impl.ClusterInfo
	nodeLogs    [][]string
	cursor      *components.Cursor
	ctx         context.Context
	parentCtx   context.Context
	cancel      context.CancelFunc
	clusterId   int
	clusterChan <-chan *impl.ClusterInfo
	spinner     spinner.Model
	width       int
	height      int
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
		if msg.info.Log != "" {
			s.appendLog(msg.info.NodeId, msg.info.Log)
		} else {
			s.nodeStatus[msg.info.NodeId] = msg.info
			if msg.info.Error != nil {
				s.appendLog(msg.info.NodeId, fmt.Sprintf("error: %v", msg.info.Error))
			}
		}

		return s, waitClusterInfo(s.ctx, s.clusterChan)
	case clusterDoneMsg:
		s.cancel()
		return s, tea.Quit
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		return s, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			s.cancel()
			return s, tea.Quit
		case "b":
			s.cancel()
			nextView := NewViewSelectCluster(s.parentCtx)
			return nextView, nextView.Init()
		case "up", "k", "down", "j":
			s.cursor.Update(msg)
			return s, nil
		}
	}

	return s, nil
}

var LoadingBlue = lipgloss.Color("#4287f5")
var SuccessGreen = lipgloss.Color("#16fa0a")
var MutedGray = lipgloss.Color("#6c747d")

func (s *StartClusterView) appendLog(nodeId int, line string) {
	if nodeId < 0 || nodeId >= len(s.nodeLogs) {
		return
	}

	s.nodeLogs[nodeId] = append(s.nodeLogs[nodeId], line)
	if len(s.nodeLogs[nodeId]) > maxLogLines {
		s.nodeLogs[nodeId] = s.nodeLogs[nodeId][len(s.nodeLogs[nodeId])-maxLogLines:]
	}
}

func (s *StartClusterView) View() string {
	leftPanel := s.renderServices()
	rightPanel := s.renderLogs()

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (s *StartClusterView) renderServices() string {
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

		if i == s.cursor.Cursor() {
			render.WriteString("> ")
		} else {
			render.WriteString("  ")
		}

		render.WriteString(prefix)
		render.WriteString(" ")
		render.WriteString(node.Name)
		render.WriteString("\n")
	}

	return render.String()
}

func (s *StartClusterView) renderLogs() string {
	if len(s.nodes) == 0 {
		return ""
	}

	selected := s.cursor.Cursor()
	logs := s.nodeLogs[selected]
	panelHeight := s.height - 4
	if panelHeight < 8 {
		panelHeight = 16
	}

	visibleLines := logs
	if len(visibleLines) > panelHeight-2 {
		visibleLines = visibleLines[len(visibleLines)-(panelHeight-2):]
	}

	body := strings.Builder{}
	if len(visibleLines) == 0 {
		body.WriteString(lipgloss.NewStyle().Foreground(MutedGray).Render("Aguardando logs..."))
	} else {
		body.WriteString(strings.Join(visibleLines, "\n"))
	}

	panelWidth := s.width - 32
	if panelWidth < 40 {
		panelWidth = 64
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(MutedGray).
		Padding(0, 1).
		MarginLeft(4).
		Width(panelWidth).
		Height(panelHeight).
		Render(
			lipgloss.NewStyle().Bold(true).Render(s.nodes[selected].Name) +
				"\n" +
				body.String(),
		)
}
