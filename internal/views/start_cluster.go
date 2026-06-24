package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/megalypse/go-svc-cluster/internal/components"
	"github.com/megalypse/go-svc-cluster/internal/domain/impl"
	"github.com/megalypse/go-svc-cluster/internal/domain/models"
	"github.com/megalypse/go-svc-cluster/internal/factory"
)

const (
	maxLogLines           = 500
	logPanelGap           = 4
	logPanelBorderWidth   = 2
	logPanelHorizontalPad = 2
)

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
		ctx:                 ctx,
		parentCtx:           parentCtx,
		cancel:              cancel,
		clusterId:           clusterId,
		nodes:               nodes,
		nodeStatus:          make([]*impl.ClusterInfo, len(nodes)),
		nodeLogs:            make([][]string, len(nodes)),
		nodeLogScrollBottom: make([]int, len(nodes)),
		cursor:              components.NewCursor(len(nodes) - 1),
		clusterChan:         clusterChan,
		spinner:             s,
	}
}

type StartClusterView struct {
	nodes               []*models.Node
	nodeStatus          []*impl.ClusterInfo
	nodeLogs            [][]string
	nodeLogScrollBottom []int
	cursor              *components.Cursor
	ctx                 context.Context
	parentCtx           context.Context
	cancel              context.CancelFunc
	clusterId           int
	clusterChan         <-chan *impl.ClusterInfo
	spinner             spinner.Model
	width               int
	height              int
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
		for i := range s.nodeLogs {
			s.clampLogScroll(i)
		}
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
		case "x":
			s.stopSelectedNode()
			return s, nil
		case "pgup", "ctrl+u":
			s.scrollSelectedLogs(s.logBodyHeight())
			return s, nil
		case "pgdown", "ctrl+d":
			s.scrollSelectedLogs(-s.logBodyHeight())
			return s, nil
		case "home":
			s.scrollSelectedLogs(maxLogLines)
			return s, nil
		case "end":
			s.scrollSelectedLogs(-maxLogLines)
			return s, nil
		}
	}

	return s, nil
}

var LoadingBlue = lipgloss.Color("#4287f5")
var SuccessGreen = lipgloss.Color("#16fa0a")
var FailureRed = lipgloss.Color("#ff4d4d")
var MutedGray = lipgloss.Color("#6c747d")

func (s *StartClusterView) appendLog(nodeId int, line string) {
	if nodeId < 0 || nodeId >= len(s.nodeLogs) {
		return
	}

	s.nodeLogs[nodeId] = append(s.nodeLogs[nodeId], line)
	if len(s.nodeLogs[nodeId]) > maxLogLines {
		s.nodeLogs[nodeId] = s.nodeLogs[nodeId][len(s.nodeLogs[nodeId])-maxLogLines:]
	}

	s.clampLogScroll(nodeId)
}

func (s *StartClusterView) scrollSelectedLogs(delta int) {
	selected := s.cursor.Cursor()
	if selected < 0 || selected >= len(s.nodeLogScrollBottom) {
		return
	}

	s.nodeLogScrollBottom[selected] += delta
	s.clampLogScroll(selected)
}

func (s *StartClusterView) stopSelectedNode() {
	selected := s.cursor.Cursor()
	if selected < 0 || selected >= len(s.nodes) {
		return
	}

	err := factory.GetServiceStartCluster().StopNode(s.clusterId, selected)
	if err != nil {
		s.appendLog(selected, fmt.Sprintf("stop failed: %v", err))
		return
	}

	s.appendLog(selected, "stop requested")
}

func (s *StartClusterView) clampLogScroll(nodeId int) {
	if nodeId < 0 || nodeId >= len(s.nodeLogScrollBottom) {
		return
	}

	maxScroll := len(s.wrappedLogLines(nodeId, s.logContentWidth())) - s.logBodyHeight()
	if maxScroll < 0 {
		maxScroll = 0
	}

	if s.nodeLogScrollBottom[nodeId] < 0 {
		s.nodeLogScrollBottom[nodeId] = 0
	}
	if s.nodeLogScrollBottom[nodeId] > maxScroll {
		s.nodeLogScrollBottom[nodeId] = maxScroll
	}
}

func (s *StartClusterView) View() string {
	leftPanel := s.renderServices()
	rightPanel := s.renderLogs(lipgloss.Width(leftPanel))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (s *StartClusterView) renderServices() string {
	render := strings.Builder{}
	panelHeight := s.servicePanelHeight()
	start, end := s.visibleServiceRange(panelHeight)

	for i := start; i < end; i++ {
		if i > start {
			render.WriteString("\n")
		}
		render.WriteString(s.renderServiceLine(i))
	}

	return lipgloss.NewStyle().
		Width(s.servicePanelWidth()).
		Height(panelHeight).
		Render(render.String())
}

func (s *StartClusterView) renderServiceLine(nodeId int) string {
	node := s.nodes[nodeId]
	prefix := func() string {
		if s.nodeStatus[nodeId] == nil || s.nodeStatus[nodeId].Loading {
			return lipgloss.NewStyle().Foreground(LoadingBlue).Render(s.spinner.View())
		}

		if s.nodeStatus[nodeId].Error != nil {
			return lipgloss.NewStyle().Foreground(FailureRed).Render("X")
		}

		return lipgloss.NewStyle().Foreground(SuccessGreen).Render("✓")
	}()

	cursor := "  "
	if nodeId == s.cursor.Cursor() {
		cursor = "> "
	}

	return cursor + prefix + " " + node.Name
}

func (s *StartClusterView) renderLogs(leftPanelWidth int) string {
	if len(s.nodes) == 0 {
		return ""
	}

	selected := s.cursor.Cursor()
	panelHeight := s.logPanelHeight()
	bodyHeight := s.logBodyHeight()
	panelWidth := s.logPanelWidth(leftPanelWidth)
	contentWidth := panelWidth - logPanelHorizontalPad
	if contentWidth < 1 {
		contentWidth = 1
	}
	visibleLines := s.visibleLogLines(selected, bodyHeight, contentWidth)

	body := strings.Builder{}
	if len(visibleLines) == 0 {
		body.WriteString(lipgloss.NewStyle().Foreground(MutedGray).Render("Aguardando logs..."))
	} else {
		body.WriteString(strings.Join(visibleLines, "\n"))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(MutedGray).
		Padding(0, 1).
		MarginLeft(logPanelGap).
		Width(panelWidth).
		Height(panelHeight).
		Render(
			lipgloss.NewStyle().Bold(true).Render(s.nodes[selected].Name) +
				"\n" +
				body.String(),
		)
}

func (s *StartClusterView) visibleLogLines(nodeId int, bodyHeight int, contentWidth int) []string {
	if nodeId < 0 || nodeId >= len(s.nodeLogs) || bodyHeight <= 0 {
		return nil
	}

	logs := s.wrappedLogLines(nodeId, contentWidth)
	if len(logs) == 0 {
		return nil
	}

	scrollBottom := s.nodeLogScrollBottom[nodeId]
	end := len(logs) - scrollBottom
	if end < 0 {
		end = 0
	}
	start := end - bodyHeight
	if start < 0 {
		start = 0
	}

	return logs[start:end]
}

func (s *StartClusterView) logPanelHeight() int {
	if s.height < 1 {
		return len(s.nodes)
	}

	return s.height
}

func (s *StartClusterView) logBodyHeight() int {
	bodyHeight := s.logPanelHeight() - 2
	if bodyHeight < 1 {
		return 1
	}

	return bodyHeight
}

func (s *StartClusterView) logPanelWidth(leftPanelWidth int) int {
	panelWidth := s.width - leftPanelWidth - logPanelGap - logPanelBorderWidth
	if s.width == 0 {
		return 64
	}
	if panelWidth < 1 {
		return 1
	}

	return panelWidth
}

func (s *StartClusterView) logContentWidth() int {
	contentWidth := s.logPanelWidth(s.servicePanelWidth()) - logPanelHorizontalPad
	if contentWidth < 1 {
		return 1
	}

	return contentWidth
}

func (s *StartClusterView) servicePanelHeight() int {
	return s.logPanelHeight()
}

func (s *StartClusterView) servicePanelWidth() int {
	width := 1
	for i := range s.nodes {
		lineWidth := lipgloss.Width(s.renderServiceLine(i))
		if lineWidth > width {
			width = lineWidth
		}
	}

	return width
}

func (s *StartClusterView) visibleServiceRange(height int) (int, int) {
	if len(s.nodes) == 0 || height <= 0 {
		return 0, 0
	}
	if height >= len(s.nodes) {
		return 0, len(s.nodes)
	}

	selected := s.cursor.Cursor()
	start := selected - height/2
	if start < 0 {
		start = 0
	}

	end := start + height
	if end > len(s.nodes) {
		end = len(s.nodes)
		start = end - height
	}

	return start, end
}

func (s *StartClusterView) wrappedLogLines(nodeId int, width int) []string {
	if nodeId < 0 || nodeId >= len(s.nodeLogs) {
		return nil
	}
	if width < 1 {
		width = 1
	}

	var lines []string
	for _, line := range s.nodeLogs[nodeId] {
		wrapped := strings.Split(ansi.Wrap(line, width, ""), "\n")
		lines = append(lines, wrapped...)
	}

	return lines
}
