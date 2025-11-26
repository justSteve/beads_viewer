package ui_test

import (
	"os"
	"testing"
	"time"

	"beads_viewer/pkg/model"
	"beads_viewer/pkg/ui"
	"github.com/charmbracelet/lipgloss"
)

func createTime(hoursAgo int) time.Time {
	return time.Now().Add(time.Duration(-hoursAgo) * time.Hour)
}

// We need a whitebox test for this specific verification or assume blackbox behavior.
// Blackbox: SetIssues -> Move to col -> Check SelectedIssue
func TestBoardModelBlackbox(t *testing.T) {
	issues := []model.Issue{
		{ID: "1", Status: model.StatusOpen, Priority: 1, CreatedAt: createTime(0)},
	}
	
	theme := ui.DefaultTheme(lipgloss.NewRenderer(os.Stdout))
	b := ui.NewBoardModel(issues, theme)
	
	// Focus Open col (0)
	sel := b.SelectedIssue()
	if sel == nil || sel.ID != "1" {
		t.Errorf("Expected ID 1 selected in Open col")
	}

	// Update issues
	newIssues := []model.Issue{
		{ID: "2", Status: model.StatusOpen, Priority: 1, CreatedAt: createTime(0)},
	}
	b.SetIssues(newIssues)

	sel = b.SelectedIssue()
	if sel == nil || sel.ID != "2" {
		t.Errorf("Expected ID 2 selected after update, got %v", sel)
	}

	// Filter to empty
	b.SetIssues([]model.Issue{})
	sel = b.SelectedIssue()
	if sel != nil {
		t.Errorf("Expected nil selection for empty board")
	}
}
