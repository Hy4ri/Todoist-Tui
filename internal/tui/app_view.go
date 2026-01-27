package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	var content string
	switch a.currentView {
	case ViewHelp:
		content = a.helpComp.View()
	case ViewTaskDetail:
		content = a.renderTaskDetail()
	case ViewTaskForm:
		content = a.renderTaskForm()
	case ViewSearch:
		content = a.renderSearch()
	case ViewCalendarDay:
		content = a.renderCalendarDay()
	case ViewSections:
		content = a.renderSections()
	default:
		content = a.renderMainView()
	}

	// Overlay project creation dialog if active
	if a.isCreatingProject {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("📁 New Project") + "\n\n" +
			a.projectInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: create • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
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
	if a.isEditingProject {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("✏️ Edit Project") + "\n\n" +
			a.projectInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: save • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
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
	if a.confirmDeleteProject && a.editingProject != nil {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.ErrorColor).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.StatusBarError.Render("⚠️ Delete Project?") + "\n\n" +
			fmt.Sprintf("Are you sure you want to delete \"%s\"?\n", a.editingProject.Name) +
			styles.HelpDesc.Render("This will delete all tasks in this project.") + "\n\n" +
			styles.HelpDesc.Render("y: confirm • n/Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
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
	if a.isCreatingLabel {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("🏷️ New Label") + "\n\n" +
			a.labelInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: create • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
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
	if a.isEditingLabel {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("✏️ Edit Label") + "\n\n" +
			a.labelInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: save • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
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
	if a.confirmDeleteLabel && a.editingLabel != nil {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.ErrorColor).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.StatusBarError.Render("⚠️ Delete Label?") + "\n\n" +
			fmt.Sprintf("Are you sure you want to delete \"%s\"?\n", a.editingLabel.Name) +
			styles.HelpDesc.Render("y: confirm • n/Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
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
	if a.isCreatingSubtask {
		dialogWidth := 60
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("➕ Add Subtask") + "\n\n" +
			a.subtaskInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: create • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
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
	if a.isCreatingSection {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("📂 New Section") + "\n\n" +
			a.sectionInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: create • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
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
	if a.isEditingSection {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("✏️ Edit Section") + "\n\n" +
			a.sectionInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: save • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
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
	if a.confirmDeleteSection && a.editingSection != nil {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.ErrorColor).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.StatusBarError.Render("⚠️ Delete Section?") + "\n\n" +
			fmt.Sprintf("Are you sure you want to delete \"%s\"?\n", a.editingSection.Name) +
			styles.HelpDesc.Render("This will likely delete/move tasks inside.") + "\n\n" +
			styles.HelpDesc.Render("y: confirm • n/Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
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
	if a.isMovingTask {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		var b strings.Builder
		b.WriteString(styles.Title.Render("➡️ Move Task to Section") + "\n\n")

		if len(a.sections) == 0 {
			b.WriteString(styles.HelpDesc.Render("No sections in this project."))
		} else {
			for i, section := range a.sections {
				cursor := "  "
				style := lipgloss.NewStyle()
				if i == a.moveSectionCursor {
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
		leftPad := (a.width - dialogWidth - 4) / 2
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
	if a.isAddingComment {
		dialogWidth := 60
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("💬 Add Comment") + "\n\n" +
			a.commentInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: submit • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
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
func (a *App) renderMainView() string {
	// Render tab bar
	tabBar := a.renderTabBar()

	// Calculate content height dynamically (total - tab bar - status bar)
	tabBarHeight := lipgloss.Height(tabBar)
	statusBarHeight := 2
	contentHeight := a.height - tabBarHeight - statusBarHeight

	var mainContent string

	// If detail panel is shown, split the view
	if a.showDetailPanel && a.selectedTask != nil {
		// Split layout
		detailWidth := a.width / 2
		remainingWidth := a.width - detailWidth - 3 // -3 for border/spacing

		if a.currentTab == TabProjects {
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
			a.sidebarComp.SetSize(sidebarWidth, contentHeight)
			a.sidebarComp.SetCursor(a.sidebarCursor)
			if a.focusedPane == PaneSidebar {
				a.sidebarComp.Focus()
			} else {
				a.sidebarComp.Blur()
			}
			if a.currentProject != nil {
				a.sidebarComp.SetActiveProject(a.currentProject.ID)
			}
			sidebarPane := a.sidebarComp.View()

			// Render Task List
			taskListPane := a.renderProjectTaskList(taskListWidth, contentHeight)

			// Render Detail
			a.detailComp.SetSize(detailWidth, contentHeight)
			a.detailComp.SetTask(a.selectedTask)
			a.detailComp.SetComments(a.comments)
			if a.currentView == ViewTaskDetail {
				a.detailComp.Focus()
			} else {
				a.detailComp.Blur()
			}
			rightPane := a.detailComp.ViewPanel()

			// Enforce strict dimensions for top alignment and stable layout
			sidebarPane = lipgloss.Place(sidebarWidth, contentHeight, lipgloss.Left, lipgloss.Top, sidebarPane)
			taskListPane = lipgloss.Place(taskListWidth, contentHeight, lipgloss.Left, lipgloss.Top, taskListPane)
			rightPane = lipgloss.Place(detailWidth, contentHeight, lipgloss.Left, lipgloss.Top, rightPane)

			mainContent = lipgloss.JoinHorizontal(lipgloss.Top, sidebarPane, " ", taskListPane, " ", rightPane)
		} else {
			// Two-pane layout: Tasks | Detail
			// Need 1 space for join
			listWidth := remainingWidth - 1
			leftPane := a.renderTaskList(listWidth, contentHeight)

			// Render Detail
			a.detailComp.SetSize(detailWidth, contentHeight)
			a.detailComp.SetTask(a.selectedTask)
			a.detailComp.SetComments(a.comments)
			if a.currentView == ViewTaskDetail {
				a.detailComp.Focus()
			} else {
				a.detailComp.Blur()
			}
			rightPane := a.detailComp.ViewPanel()

			// Enforce strict dimensions
			leftPane = lipgloss.Place(listWidth, contentHeight, lipgloss.Left, lipgloss.Top, leftPane)
			rightPane = lipgloss.Place(detailWidth, contentHeight, lipgloss.Left, lipgloss.Top, rightPane)

			mainContent = lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)
		}

	} else {
		if a.currentTab == TabProjects {
			// Projects tab shows sidebar + content
			mainContent = a.renderProjectsTabContent(a.width, contentHeight)
		} else {
			// Other tabs show content only (full width)
			mainContent = a.renderTaskList(a.width-2, contentHeight)
		}
	}

	// Add status bar
	statusBar := a.renderStatusBar()

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
		{TabToday, "[T]", "Today", "Tdy"},
		{TabUpcoming, "[U]", "Upcoming", "Up"},
		{TabLabels, "[L]", "Labels", "Lbl"},
		{TabCalendar, "[C]", "Calendar", "Cal"},
		{TabProjects, "[P]", "Projects", "Prj"},
	}
}

// renderTabBar renders the top tab bar.
func (a *App) renderTabBar() string {
	tabs := getTabDefinitions()

	// Determine label style based on available width
	// Full: "T Today" (~9 chars rendered), Short: "T Tdy" (~7 chars), Minimal: "T" (~3 chars)
	// Each tab with padding(2+2) + separator(1) = +5 chars overhead
	// 5 tabs * 14 chars (full with padding) = ~70 chars minimum for full labels
	useShortLabels := a.width < 80
	useMinimalLabels := a.width < 50

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

		if a.currentTab == t.tab {
			tabStrs = append(tabStrs, styles.TabActive.Render(label))
		} else {
			tabStrs = append(tabStrs, styles.Tab.Render(label))
		}
	}

	tabLine := strings.Join(tabStrs, " ")

	// Truncate if still too wide
	maxWidth := a.width - 4 // Account for TabBar padding
	if lipgloss.Width(tabLine) > maxWidth && maxWidth > 0 {
		tabLine = lipgloss.NewStyle().MaxWidth(maxWidth).Render(tabLine)
	}

	return styles.TabBar.Width(a.width).Render(tabLine)
}
