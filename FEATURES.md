# Todoist TUI - API Feature Coverage

This document tracks which Todoist API features are available and which are implemented in the TUI.

## Tasks

| API Function | TUI Implementation |
|--------------|-------------------|
| [x] GetTasks | View tasks in Today, Upcoming, Project views |
| [x] GetTask | View task details |
| [x] CreateTask | Add new task (a key) |
| [x] UpdateTask | Edit task (e key), priority, due date |
| [x] CloseTask | Complete task (x key) |
| [x] ReopenTask | Uncomplete task (x key) |
| [x] DeleteTask | Delete task (dd keys) |

### Task Features in TUI

- [x] Filter by today
- [x] Filter by project
- [x] Filter by label
- [x] Filter by date (calendar)
- [x] Set priority (1-4 keys)
- [x] Set due today (< key)
- [x] Set due tomorrow (> key)
- [x] Search tasks (/ key)
- [x] View task description
- [x] Edit task content
- [x] Edit task description
- [ ] Edit task labels
- [ ] Move task to project (Cross-project move not implemented)
- [x] Move task to section
- [x] Subtasks (Inline creation)
- [ ] Task duration

---

## Projects

| API Function | TUI Implementation |
|--------------|-------------------|
| [x] GetProjects | View projects in sidebar |
| [x] GetProject | Select project to view tasks |
| [x] CreateProject | Create new project (n key) |
| [x] UpdateProject | Edit project name (e key) |
| [x] DeleteProject | Delete project (dd keys) |
| [ ] GetProjectCollaborators | View collaborators |

### Project Features in TUI

- [x] View project list
- [x] View project tasks
- [x] Create new project
- [x] Favorite projects shown separately
- [x] Nested/child projects (indented)
- [x] Edit project (e key)
- [x] Delete project (dd keys with confirmation)
- [ ] Project colors
- [ ] Project view style (list/board)

---

## Labels

| API Function | TUI Implementation |
|--------------|-------------------|
| [x] GetLabels | View labels list |
| [ ] GetLabel | - |
| [x] CreateLabel | Create new label (n key) |
| [x] UpdateLabel | Edit label (e key in list) |
| [x] DeleteLabel | Delete label (dd keys in list) |

### Label Features in TUI

- [x] View labels list
- [x] View tasks by label
- [x] Create new label
- [x] Edit label
- [x] Delete label
- [ ] Label colors

---

## Sections

| API Function | TUI Implementation |
|--------------|-------------------|
| [x] GetSections | View sections in project |
| [x] GetSection | (Internal) |
| [x] CreateSection | Create new section (S -> a) |
| [x] UpdateSection | Edit section (S -> e) |
| [x] DeleteSection | Delete section (S -> dd) |

### Section Features in TUI

- [x] View sections grouped
- [x] Create section
- [x] Edit section
- [x] Delete section
- [x] Move tasks between sections (m key)

---

## Comments

| API Function | TUI Implementation |
|--------------|-------------------|
| [x] GetComments | View task comments |
| [x] GetComment | (Internal) |
| [x] CreateComment | Add comment (c key) |
| [ ] UpdateComment | Edit comment |
| [ ] DeleteComment | Delete comment |

### Comment Features in TUI

- [x] View task comments
- [x] Add comment
- [ ] Edit comment
- [ ] Delete comment
- [ ] File attachments

---

## TUI-Only Features

These features are implemented in the TUI but not directly API-related:

- [x] Vim-style navigation (j/k/h/l)
- [x] Tab navigation (1-5 keys)
- [x] Calendar view (compact)
- [x] Calendar view (expanded/grid)
- [x] Day detail view from calendar
- [x] Help view (? key)
- [x] Status bar with hints
- [x] Loading spinner
- [x] Error display
- [x] Configurable vim mode
- [x] Calendar default view preference saved
- [x] Task priority colors
- [x] Overdue task highlighting
- [x] Today task highlighting
- [x] Inline subtask creation (s key)
- [x] Undo capability (u key)
- [x] Split View (Task Detail Panel)
- [x] Section ID management in tasks
- [x] Comment overlay dialog

---

## Summary

| Category | Implemented | Total | Coverage |
|----------|-------------|-------|----------|
| Tasks | 7 | 7 | 100% |
| Projects | 5 | 6 | 83% |
| Labels | 4 | 5 | 80% |
| Sections | 4 | 5 | 80% |
| Comments | 2 | 5 | 40% |
| **Overall** | **22** | **28** | **78%** |
