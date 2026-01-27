package ui

import "github.com/hy4ri/todoist-tui/internal/tui/state"

type Renderer struct {
	*state.State
}

func NewRenderer(s *state.State) *Renderer {
	return &Renderer{State: s}
}

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

func (r *Renderer) View() string {
	if r.Width == 0 {
		return "Loading..."
	}

	var content string
	switch r.CurrentView {
	case state.ViewHelp:
		content = r.HelpComp.View()
	case state.ViewTaskDetail:
		content = r.renderTaskDetail()
	case state.ViewTaskForm:
		content = r.renderTaskForm()
	case state.ViewSearch:
		content = r.renderSearch()
	case state.ViewCalendarDay:
		content = r.renderCalendarDay()
	case state.ViewSections:
		content = r.renderSections()
	default:
		content = r.renderMainView()
	}

	// Overlay project creation dialog if active
	if r.IsCreatingProject {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("📁 New Project") + "\n\n" +
			r.ProjectInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: create • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (r.Width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay active overlays...
	// (Skipping project/label overlays replication for brevity... waiting for replace_file_content to handle context correctly)
	// I should probably target specific blocks instead of replcaing huge chunks unless necessary.
	// But I will append section overlays at the END of overlays list.

	// Overlay project edit dialog if active
	if r.IsEditingProject {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("✏️ Edit Project") + "\n\n" +
			r.ProjectInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: save • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (r.Width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay project delete confirmation dialog if active
	if r.ConfirmDeleteProject && r.EditingProject != nil {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.ErrorColor).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.StatusBarError.Render("⚠️ Delete Project?") + "\n\n" +
			fmt.Sprintf("Are you sure you want to delete \"%s\"?\n", r.EditingProject.Name) +
			styles.HelpDesc.Render("This will delete all tasks in this project.") + "\n\n" +
			styles.HelpDesc.Render("y: confirm • n/Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (r.Width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay label creation dialog if active
	if r.IsCreatingLabel {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("🏷️ New Label") + "\n\n" +
			r.LabelInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: create • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (r.Width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay label edit dialog if active
	if r.IsEditingLabel {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("✏️ Edit Label") + "\n\n" +
			r.LabelInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: save • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (r.Width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay label delete confirmation dialog if active
	if r.ConfirmDeleteLabel && r.EditingLabel != nil {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.ErrorColor).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.StatusBarError.Render("⚠️ Delete Label?") + "\n\n" +
			fmt.Sprintf("Are you sure you want to delete \"%s\"?\n", r.EditingLabel.Name) +
			styles.HelpDesc.Render("y: confirm • n/Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (r.Width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay subtask creation dialog if active
	if r.IsCreatingSubtask {
		dialogWidth := 60
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("➕ Add Subtask") + "\n\n" +
			r.SubtaskInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: create • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (r.Width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay section creation dialog if active
	if r.IsCreatingSection {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("📂 New Section") + "\n\n" +
			r.SectionInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: create • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (r.Width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay section edit dialog if active
	if r.IsEditingSection {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("✏️ Edit Section") + "\n\n" +
			r.SectionInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: save • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (r.Width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay section delete confirmation dialog if active
	if r.ConfirmDeleteSection && r.EditingSection != nil {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.ErrorColor).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.StatusBarError.Render("⚠️ Delete Section?") + "\n\n" +
			fmt.Sprintf("Are you sure you want to delete \"%s\"?\n", r.EditingSection.Name) +
			styles.HelpDesc.Render("This will likely delete/move tasks inside.") + "\n\n" +
			styles.HelpDesc.Render("y: confirm • n/Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (r.Width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
		content = strings.Join(contentLines, "\n")
	}

	// Overlay move task dialog if active
	if r.IsMovingTask {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		var b strings.Builder
		b.WriteString(styles.Title.Render("➡️ Move Task to Section") + "\n\n")

		if len(r.Sections) == 0 {
			b.WriteString(styles.HelpDesc.Render("No sections in this project."))
		} else {
			for i, section := range r.Sections {
				cursor := "  "
				style := lipgloss.NewStyle()
				if i == r.MoveSectionCursor {
					cursor = "> "
					style = lipgloss.NewStyle().Foreground(styles.Highlight)
				}
				b.WriteString(cursor + style.Render(section.Name) + "\n")
			}
		}

		b.WriteString("\n" + styles.HelpDesc.Render("j/k: select • Enter: move • Esc: cancel"))

		dialog := dialogStyle.Render(b.String())

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (r.Width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay add comment dialog if active
	if r.IsAddingComment {
		dialogWidth := 60
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("💬 Add Comment") + "\n\n" +
			r.CommentInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: submit • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (r.Width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	return content
}

// renderMainView renders the main layout with tab bar and content.
func (r *Renderer) renderMainView() string {
	// Render tab bar
	tabBar := r.renderTabBar()

	// Calculate content height dynamically (total - tab bar - status bar)
	tabBarHeight := lipgloss.Height(tabBar)
	statusBarHeight := 2
	contentHeight := r.Height - tabBarHeight - statusBarHeight

	var mainContent string

	// If detail panel is shown, split the view
	if r.ShowDetailPanel && r.SelectedTask != nil {
		// Split layout
		detailWidth := r.Width / 2
		remainingWidth := r.Width - detailWidth - 3 // -3 for border/spacing

		if r.CurrentTab == state.TabProjects {
			// Three-pane layout: Sidebar | Tasks | Detail
			sidebarWidth := 30
			if remainingWidth < 70 {
				sidebarWidth = 20
			}
			if remainingWidth < 50 {
				sidebarWidth = 15
			}

			// We need 2 spaces for joins
			taskListWidth := remainingWidth - sidebarWidth - 2

			// Sizing validation
			if taskListWidth < 20 {
				taskListWidth = 20
			}

			// Render Sidebar
			r.SidebarComp.SetSize(sidebarWidth, contentHeight)
			r.SidebarComp.SetCursor(r.SidebarCursor)
			if r.FocusedPane == state.PaneSidebar {
				r.SidebarComp.Focus()
			} else {
				r.SidebarComp.Blur()
			}
			if r.CurrentProject != nil {
				r.SidebarComp.SetActiveProject(r.CurrentProject.ID)
			}
			sidebarPane := r.SidebarComp.View()

			// Render Task List
			taskListPane := r.renderProjectTaskList(taskListWidth, contentHeight)

			// Render Detail
			r.DetailComp.SetSize(detailWidth, contentHeight)
			r.DetailComp.SetTask(r.SelectedTask)
			r.DetailComp.SetComments(r.Comments)
			if r.CurrentView == state.ViewTaskDetail {
				r.DetailComp.Focus()
			} else {
				r.DetailComp.Blur()
			}
			rightPane := r.DetailComp.state.ViewPanel()

			// Enforce strict dimensions for top alignment and stable layout
			sidebarPane = lipgloss.Place(sidebarWidth, contentHeight, lipgloss.Left, lipgloss.Top, sidebarPane)
			taskListPane = lipgloss.Place(taskListWidth, contentHeight, lipgloss.Left, lipgloss.Top, taskListPane)
			rightPane = lipgloss.Place(detailWidth, contentHeight, lipgloss.Left, lipgloss.Top, rightPane)

			mainContent = lipgloss.JoinHorizontal(lipgloss.Top, sidebarPane, " ", taskListPane, " ", rightPane)
		} else {
			// Two-pane layout: Tasks | Detail
			// Need 1 space for join
			listWidth := remainingWidth - 1
			leftPane := r.renderTaskList(listWidth, contentHeight)

			// Render Detail
			r.DetailComp.SetSize(detailWidth, contentHeight)
			r.DetailComp.SetTask(r.SelectedTask)
			r.DetailComp.SetComments(r.Comments)
			if r.CurrentView == state.ViewTaskDetail {
				r.DetailComp.Focus()
			} else {
				r.DetailComp.Blur()
			}
			rightPane := r.DetailComp.state.ViewPanel()

			// Enforce strict dimensions
			leftPane = lipgloss.Place(listWidth, contentHeight, lipgloss.Left, lipgloss.Top, leftPane)
			rightPane = lipgloss.Place(detailWidth, contentHeight, lipgloss.Left, lipgloss.Top, rightPane)

			mainContent = lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)
		}

	} else {
		if r.CurrentTab == state.TabProjects {
			// Projects tab shows sidebar + content
			mainContent = r.renderProjectsTabContent(r.Width, contentHeight)
		} else {
			// Other tabs show content only (full width)
			mainContent = r.renderTaskList(r.Width-2, contentHeight)
		}
	}

	// Add status bar
	statusBar := r.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, mainContent, statusBar)
}

// tabInfo holds tab metadata for rendering and click handling.
type tabInfo struct {
	tab       Tab
	icon      string
	name      string
	shortName string
}

// getTabDefinitions returns the tab definitions.
func getTabDefinitions() []tabInfo {
	return []tabInfo{
		{state.TabToday, "[T]", "Today", "Tdy"},
		{state.TabUpcoming, "[U]", "Upcoming", "Up"},
		{state.TabLabels, "[L]", "Labels", "Lbl"},
		{state.TabCalendar, "[C]", "Calendar", "Cal"},
		{state.TabProjects, "[P]", "Projects", "Prj"},
	}
}

// renderTabBar renders the top tab bar.
func (r *Renderer) renderTabBar() string {
	tabs := getTabDefinitions()

	// Determine label style based on available width
	// Full: "T Today" (~9 chars rendered), Short: "T Tdy" (~7 chars), Minimal: "T" (~3 chars)
	// Each tab with padding(2+2) + separator(1) = +5 chars overhead
	// 5 tabs * 14 chars (full with padding) = ~70 chars minimum for full labels
	useShortLabels := r.Width < 80
	useMinimalLabels := r.Width < 50

	var tabStrs []string
	for _, t := range tabs {
		var label string
		if useMinimalLabels {
			label = t.icon
		} else if useShortLabels {
			label = fmt.Sprintf("%s %s", t.icon, t.shortName)
		} else {
			label = fmt.Sprintf("%s %s", t.icon, t.name)
		}

		if r.CurrentTab == t.tab {
			tabStrs = append(tabStrs, styles.state.TabActive.Render(label))
		} else {
			tabStrs = append(tabStrs, styles.Tab.Render(label))
		}
	}

	tabLine := strings.Join(tabStrs, " ")

	// Truncate if still too wide
	maxWidth := r.Width - 4 // Account for state.TabBar padding
	if lipgloss.Width(tabLine) > maxWidth && maxWidth > 0 {
		tabLine = lipgloss.NewStyle().MaxWidth(maxWidth).Render(tabLine)
	}

	return styles.state.TabBar.Width(r.Width).Render(tabLine)
}
