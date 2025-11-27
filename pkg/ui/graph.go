package ui

import (
	"fmt"
	"sort"
	"strings"

	"beads_viewer/pkg/analysis"
	"beads_viewer/pkg/model"

	"github.com/charmbracelet/lipgloss"
)

// GraphModel represents the dependency graph view - ego-centric neighborhood display
type GraphModel struct {
	issues       []model.Issue
	issueMap     map[string]*model.Issue
	insights     *analysis.Insights
	selectedIdx  int
	scrollOffset int
	width        int
	height       int
	theme        Theme

	// Precomputed graph relationships
	blockers   map[string][]string // What each issue depends on (blocks this issue)
	dependents map[string][]string // What depends on each issue (this issue blocks)

	// Flat list for navigation
	sortedIDs []string
}

// NewGraphModel creates a new graph view from issues
func NewGraphModel(issues []model.Issue, insights *analysis.Insights, theme Theme) GraphModel {
	g := GraphModel{
		issues:   issues,
		insights: insights,
		theme:    theme,
	}
	g.rebuildGraph()
	return g
}

// SetIssues updates the graph data
func (g *GraphModel) SetIssues(issues []model.Issue, insights *analysis.Insights) {
	g.issues = issues
	g.insights = insights
	g.rebuildGraph()
}

func (g *GraphModel) rebuildGraph() {
	g.issueMap = make(map[string]*model.Issue)
	g.blockers = make(map[string][]string)
	g.dependents = make(map[string][]string)
	g.sortedIDs = nil

	for i := range g.issues {
		issue := &g.issues[i]
		g.issueMap[issue.ID] = issue
		g.sortedIDs = append(g.sortedIDs, issue.ID)
	}

	// Build relationships
	for _, issue := range g.issues {
		for _, dep := range issue.Dependencies {
			if dep.Type == model.DepBlocks || dep.Type == model.DepParentChild {
				// issue depends on dep.DependsOnID
				g.blockers[issue.ID] = append(g.blockers[issue.ID], dep.DependsOnID)
				// dep.DependsOnID blocks issue
				g.dependents[dep.DependsOnID] = append(g.dependents[dep.DependsOnID], issue.ID)
			}
		}
	}

	// Sort by impact score (from insights) if available, else by ID
	if g.insights != nil && g.insights.Stats != nil {
		sort.Slice(g.sortedIDs, func(i, j int) bool {
			scoreI := g.insights.Stats.CriticalPathScore[g.sortedIDs[i]]
			scoreJ := g.insights.Stats.CriticalPathScore[g.sortedIDs[j]]
			if scoreI != scoreJ {
				return scoreI > scoreJ // Higher impact first
			}
			return g.sortedIDs[i] < g.sortedIDs[j]
		})
	} else {
		sort.Strings(g.sortedIDs)
	}

	if g.selectedIdx >= len(g.sortedIDs) {
		g.selectedIdx = 0
	}
}

// Navigation
func (g *GraphModel) MoveUp() {
	if g.selectedIdx > 0 {
		g.selectedIdx--
		g.ensureVisible()
	}
}

func (g *GraphModel) MoveDown() {
	if g.selectedIdx < len(g.sortedIDs)-1 {
		g.selectedIdx++
		g.ensureVisible()
	}
}

func (g *GraphModel) MoveLeft()  { g.MoveUp() }
func (g *GraphModel) MoveRight() { g.MoveDown() }

func (g *GraphModel) PageUp() {
	g.selectedIdx -= 10
	if g.selectedIdx < 0 {
		g.selectedIdx = 0
	}
	g.ensureVisible()
}

func (g *GraphModel) PageDown() {
	g.selectedIdx += 10
	if g.selectedIdx >= len(g.sortedIDs) {
		g.selectedIdx = len(g.sortedIDs) - 1
	}
	g.ensureVisible()
}

func (g *GraphModel) ScrollLeft()  {}
func (g *GraphModel) ScrollRight() {}

func (g *GraphModel) ensureVisible() {
	// Will be used with scrollOffset if needed
}

func (g *GraphModel) SelectedIssue() *model.Issue {
	if len(g.sortedIDs) == 0 {
		return nil
	}
	id := g.sortedIDs[g.selectedIdx]
	return g.issueMap[id]
}

func (g *GraphModel) TotalCount() int {
	return len(g.sortedIDs)
}

// View renders the ego-centric graph view
func (g *GraphModel) View(width, height int) string {
	g.width = width
	g.height = height
	t := g.theme

	if len(g.sortedIDs) == 0 {
		return t.Renderer.NewStyle().
			Width(width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(t.Secondary).
			Render("No issues to display")
	}

	selectedID := g.sortedIDs[g.selectedIdx]
	selectedIssue := g.issueMap[selectedID]
	if selectedIssue == nil {
		return "Error: selected issue not found"
	}

	// Layout: Left panel (node list) | Right panel (neighborhood view)
	listWidth := 32
	if width < 100 {
		listWidth = 24
	}
	if width < 80 {
		// Narrow: just show neighborhood
		return g.renderNeighborhood(selectedID, selectedIssue, width, height, t)
	}

	detailWidth := width - listWidth - 3 // 3 for border/separator

	// Left: scrollable list of all nodes
	listView := g.renderNodeList(listWidth, height-2, t)

	// Right: neighborhood view of selected node
	neighborView := g.renderNeighborhood(selectedID, selectedIssue, detailWidth, height-2, t)

	// Combine with separator
	separator := t.Renderer.NewStyle().
		Foreground(t.Secondary).
		Render(strings.Repeat("│\n", height-2))

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, separator, neighborView)
}

// renderNodeList renders the left panel with all nodes
func (g *GraphModel) renderNodeList(width, height int, t Theme) string {
	var lines []string

	// Header
	headerStyle := t.Renderer.NewStyle().
		Bold(true).
		Foreground(t.Primary).
		Width(width)
	lines = append(lines, headerStyle.Render("📊 Nodes ("+fmt.Sprintf("%d", len(g.sortedIDs))+")"))
	lines = append(lines, strings.Repeat("─", width))

	// Calculate visible range
	visibleItems := height - 4
	if visibleItems < 1 {
		visibleItems = 1
	}

	startIdx := g.scrollOffset
	if g.selectedIdx < startIdx {
		startIdx = g.selectedIdx
	} else if g.selectedIdx >= startIdx+visibleItems {
		startIdx = g.selectedIdx - visibleItems + 1
	}
	g.scrollOffset = startIdx

	endIdx := startIdx + visibleItems
	if endIdx > len(g.sortedIDs) {
		endIdx = len(g.sortedIDs)
	}

	// Render visible items
	for i := startIdx; i < endIdx; i++ {
		id := g.sortedIDs[i]
		issue := g.issueMap[id]
		if issue == nil {
			continue
		}

		isSelected := i == g.selectedIdx

		// Status indicator
		statusIcon := getStatusIcon(issue.Status)

		// Truncate ID to fit
		maxIDLen := width - 4 // 2 for status, 2 for padding
		displayID := smartTruncateID(id, maxIDLen)

		line := fmt.Sprintf("%s %s", statusIcon, displayID)

		var style lipgloss.Style
		if isSelected {
			style = t.Renderer.NewStyle().
				Bold(true).
				Foreground(t.Primary).
				Background(t.Highlight).
				Width(width)
		} else {
			style = t.Renderer.NewStyle().
				Foreground(getStatusColor(issue.Status, t)).
				Width(width)
		}

		lines = append(lines, style.Render(line))
	}

	// Scroll indicator
	if len(g.sortedIDs) > visibleItems {
		scrollInfo := fmt.Sprintf("(%d-%d of %d)", startIdx+1, endIdx, len(g.sortedIDs))
		scrollStyle := t.Renderer.NewStyle().
			Foreground(t.Secondary).
			Italic(true).
			Width(width).
			Align(lipgloss.Center)
		lines = append(lines, scrollStyle.Render(scrollInfo))
	}

	return strings.Join(lines, "\n")
}

// renderNeighborhood renders the ego-centric view of selected node
func (g *GraphModel) renderNeighborhood(id string, issue *model.Issue, width, height int, t Theme) string {
	var sections []string

	// Header with selected node info
	headerStyle := t.Renderer.NewStyle().
		Bold(true).
		Foreground(t.Primary)

	statusIcon := getStatusIcon(issue.Status)
	prioIcon := getPriorityIcon(issue.Priority)
	typeIcon := getTypeIcon(issue.IssueType)

	header := headerStyle.Render(fmt.Sprintf("%s %s %s %s", statusIcon, prioIcon, typeIcon, id))
	sections = append(sections, header)

	// Title
	if issue.Title != "" {
		titleStyle := t.Renderer.NewStyle().
			Foreground(t.Base.GetForeground()).
			Width(width - 2)
		title := truncateRunesHelper(issue.Title, width-4, "…")
		sections = append(sections, titleStyle.Render("   "+title))
	}

	sections = append(sections, "")

	// Stats line
	blockerCount := len(g.blockers[id])
	dependentCount := len(g.dependents[id])

	statsStyle := t.Renderer.NewStyle().Foreground(t.Secondary)
	stats := fmt.Sprintf("⬆️ Blocked by: %d    ⬇️ Blocks: %d", blockerCount, dependentCount)
	sections = append(sections, statsStyle.Render(stats))
	sections = append(sections, "")

	// BLOCKERS section (what this issue depends on)
	if blockerCount > 0 {
		sections = append(sections, renderSectionHeader("⬆️ BLOCKED BY (must complete first)", t))
		for i, blockerID := range g.blockers[id] {
			if i >= 8 { // Limit to 8 items
				remaining := blockerCount - 8
				sections = append(sections, t.Renderer.NewStyle().
					Foreground(t.Secondary).
					Italic(true).
					Render(fmt.Sprintf("   ... and %d more", remaining)))
				break
			}
			sections = append(sections, g.renderRelatedNode(blockerID, width, t, "   "))
		}
		sections = append(sections, "")
	}

	// DEPENDENTS section (what depends on this issue)
	if dependentCount > 0 {
		sections = append(sections, renderSectionHeader("⬇️ BLOCKS (waiting on this)", t))
		for i, depID := range g.dependents[id] {
			if i >= 8 { // Limit to 8 items
				remaining := dependentCount - 8
				sections = append(sections, t.Renderer.NewStyle().
					Foreground(t.Secondary).
					Italic(true).
					Render(fmt.Sprintf("   ... and %d more", remaining)))
				break
			}
			sections = append(sections, g.renderRelatedNode(depID, width, t, "   "))
		}
		sections = append(sections, "")
	}

	// Insights section (if available)
	if g.insights != nil && g.insights.Stats != nil {
		sections = append(sections, renderSectionHeader("📈 IMPACT METRICS", t))

		metricsStyle := t.Renderer.NewStyle().Foreground(t.Secondary)

		if score, ok := g.insights.Stats.PageRank[id]; ok && score > 0 {
			sections = append(sections, metricsStyle.Render(fmt.Sprintf("   PageRank: %.4f", score)))
		}
		if score, ok := g.insights.Stats.CriticalPathScore[id]; ok && score > 0 {
			sections = append(sections, metricsStyle.Render(fmt.Sprintf("   Critical Path: %.2f", score)))
		}
		if score, ok := g.insights.Stats.Betweenness[id]; ok && score > 0 {
			sections = append(sections, metricsStyle.Render(fmt.Sprintf("   Betweenness: %.4f", score)))
		}
	}

	// Navigation hint
	sections = append(sections, "")
	navStyle := t.Renderer.NewStyle().
		Foreground(t.Secondary).
		Italic(true)
	sections = append(sections, navStyle.Render("j/k: navigate • enter: view details • g: back to list"))

	return strings.Join(sections, "\n")
}

func (g *GraphModel) renderRelatedNode(id string, width int, t Theme, prefix string) string {
	issue := g.issueMap[id]
	if issue == nil {
		return t.Renderer.NewStyle().
			Foreground(t.Secondary).
			Italic(true).
			Render(prefix + id + " (not in current filter)")
	}

	statusIcon := getStatusIcon(issue.Status)
	statusColor := getStatusColor(issue.Status, t)

	// Format: prefix + status + truncated ID + title snippet
	maxIDLen := 20
	displayID := smartTruncateID(id, maxIDLen)

	titleSnippet := ""
	remainingWidth := width - len(prefix) - 3 - len(displayID) - 3
	if remainingWidth > 10 && issue.Title != "" {
		titleSnippet = " " + truncateRunesHelper(issue.Title, remainingWidth, "…")
	}

	line := fmt.Sprintf("%s%s %s%s", prefix, statusIcon, displayID, titleSnippet)

	return t.Renderer.NewStyle().
		Foreground(statusColor).
		Render(line)
}

func renderSectionHeader(title string, t Theme) string {
	return t.Renderer.NewStyle().
		Bold(true).
		Foreground(t.Feature).
		Render(title)
}

// Helper functions

func getStatusIcon(status model.Status) string {
	switch status {
	case model.StatusOpen:
		return "🔵"
	case model.StatusInProgress:
		return "🟡"
	case model.StatusBlocked:
		return "🔴"
	case model.StatusClosed:
		return "✅"
	default:
		return "⚪"
	}
}

func getStatusColor(status model.Status, t Theme) lipgloss.AdaptiveColor {
	switch status {
	case model.StatusOpen:
		return t.Open
	case model.StatusInProgress:
		return t.InProgress
	case model.StatusBlocked:
		return t.Blocked
	case model.StatusClosed:
		return t.Closed
	default:
		return t.Secondary
	}
}

func getPriorityIcon(priority int) string {
	switch priority {
	case 1:
		return "🔥"
	case 2:
		return "⚡"
	case 3:
		return "📌"
	case 4:
		return "📋"
	default:
		return "  "
	}
}

func getTypeIcon(itype model.IssueType) string {
	switch itype {
	case model.TypeBug:
		return "🐛"
	case model.TypeFeature:
		return "✨"
	case model.TypeTask:
		return "📝"
	case model.TypeEpic:
		return "🎯"
	case model.TypeChore:
		return "🔧"
	default:
		return "📄"
	}
}

// smartTruncateID creates a smart short ID from a long ID
func smartTruncateID(id string, maxLen int) string {
	if len(id) <= maxLen {
		return id
	}

	// Try to create an abbreviated form for underscore-separated IDs
	parts := strings.Split(id, "_")
	if len(parts) > 2 {
		// Take first letter of each part except last, keep more of last part
		var abbrev strings.Builder
		for i, part := range parts {
			if i == len(parts)-1 {
				// Last part: keep more of it
				remaining := maxLen - abbrev.Len()
				if remaining > 0 {
					if len(part) <= remaining {
						abbrev.WriteString(part)
					} else {
						abbrev.WriteString(part[:remaining-1])
						abbrev.WriteRune('…')
					}
				}
			} else {
				// Non-last parts: just first char + underscore
				if len(part) > 0 {
					abbrev.WriteRune(rune(part[0]))
					abbrev.WriteRune('_')
				}
			}
		}
		result := abbrev.String()
		if len(result) <= maxLen {
			return result
		}
	}

	// Fall back to simple truncation
	runes := []rune(id)
	if len(runes) > maxLen-1 {
		return string(runes[:maxLen-1]) + "…"
	}
	return id
}
