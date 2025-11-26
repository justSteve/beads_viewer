package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tier represents the width tier of the display
type Tier int

const (
	TierCompact Tier = iota
	TierNormal
	TierWide
	TierUltraWide
)

type IssueDelegate struct {
	Tier  Tier
	Theme Theme
}

func (d IssueDelegate) Height() int {
	return 1
}

func (d IssueDelegate) Spacing() int {
	return 0
}

func (d IssueDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d IssueDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(IssueItem)
	if !ok {
		return
	}
	
	t := d.Theme
	
	// Styles
	var baseStyle lipgloss.Style
	if index == m.Index() {
		baseStyle = t.Selected
	} else {
		baseStyle = t.Base.Copy().PaddingLeft(1).PaddingRight(1).
			Border(lipgloss.HiddenBorder(), false, false, false, true)
	}

	// ID
	id := t.Renderer.NewStyle().Width(8).Foreground(t.Secondary).Bold(true).Render(i.Issue.ID)
	
	// Type
	icon, color := t.GetTypeIcon(string(i.Issue.IssueType))
	typeIcon := t.Renderer.NewStyle().Width(2).Align(lipgloss.Center).Foreground(color).Render(icon)
	
	// Priority
	prio := t.Renderer.NewStyle().Width(3).Align(lipgloss.Center).Render(GetPriorityIcon(i.Issue.Priority))
	
	// Status
	statusColor := t.GetStatusColor(string(i.Issue.Status))
	status := t.Renderer.NewStyle().Width(12).Align(lipgloss.Center).Bold(true).Foreground(statusColor).Render(strings.ToUpper(string(i.Issue.Status)))

	// Optional Columns
	age := ""
	comments := ""
	updated := ""
	assignee := ""
	
	extraWidth := 0

	// Assignee
	if d.Tier >= TierNormal {
		s := t.Renderer.NewStyle().Width(12).Foreground(t.Secondary).Align(lipgloss.Right)
		if i.Issue.Assignee != "" {
			assignee = s.Render("@" + i.Issue.Assignee)
		} else {
			assignee = s.Render("")
		}
		extraWidth += 12
	}

	// Age & Comments
	if d.Tier >= TierWide {
		ageStr := FormatTimeRel(i.Issue.CreatedAt)
		age = t.Renderer.NewStyle().Width(8).Foreground(t.Secondary).Align(lipgloss.Right).Render(ageStr)
		
		commentCount := len(i.Issue.Comments)
		s := t.Renderer.NewStyle().Width(4).Foreground(t.Subtext).Align(lipgloss.Right)
		if commentCount > 0 {
			comments = s.Render(fmt.Sprintf("💬%d", commentCount))
		} else {
			comments = s.Render("")
		}
		extraWidth += 12
	}

	// Updated
	if d.Tier >= TierUltraWide {
		updatedStr := FormatTimeRel(i.Issue.UpdatedAt)
		updated = t.Renderer.NewStyle().Width(10).Foreground(t.Secondary).Align(lipgloss.Right).Render(updatedStr)
		
		normImpact := i.Impact / 10.0
		if normImpact > 1.0 { normImpact = 1.0 }
		
		impactStr := RenderSparkline(normImpact, 4)
		impactStyle := t.Renderer.NewStyle().Foreground(GetHeatmapColor(normImpact)) // TODO: update GetHeatmapColor to use Theme?
		// For now keep global helper for sparkline colors or move to Theme.
		// Actually `GetHeatmapColor` uses globals `GradientHigh` etc.
		// I should update `visuals.go` to use Theme too?
		// Let's leave visuals global for now or fix later.
		
		impactRender := impactStyle.Render(impactStr)
		if i.Impact > 0 {
			impactRender = fmt.Sprintf("%s %.0f", impactRender, i.Impact)
		}
		
		updated = lipgloss.JoinHorizontal(lipgloss.Left, updated, t.Renderer.NewStyle().Width(8).Align(lipgloss.Right).Render(impactRender))
		extraWidth += 18
	}

	// Title
	gaps := 4 
	if d.Tier >= TierNormal { gaps += 1 }
	if d.Tier >= TierWide { gaps += 2 }
	if d.Tier >= TierUltraWide { gaps += 1 }

	fixedWidth := 8 + 2 + 3 + 12 + extraWidth + gaps
	availableWidth := m.Width() - fixedWidth - 4
	if availableWidth < 10 { availableWidth = 10 }

	titleStyle := t.Renderer.NewStyle().Foreground(t.Base.GetForeground()).Width(availableWidth).MaxWidth(availableWidth)
	if index == m.Index() {
		titleStyle = titleStyle.Foreground(t.Primary).Bold(true)
	}
	title := titleStyle.Render(i.Issue.Title)

	// Compose
	parts := []string{id, typeIcon, prio, status, title}
	if d.Tier >= TierWide { parts = append(parts, comments, age) }
	if d.Tier >= TierNormal { parts = append(parts, assignee) }
	if d.Tier >= TierUltraWide { parts = append(parts, updated) }

	row := lipgloss.JoinHorizontal(lipgloss.Left, parts...)
	fmt.Fprint(w, baseStyle.Render(row))
}
