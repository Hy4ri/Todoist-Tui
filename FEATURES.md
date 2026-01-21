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
- [ ] Move task to project
- [ ] Subtasks (parent tasks)
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
| [ ] UpdateLabel | Edit label |
| [ ] DeleteLabel | Delete label |

### Label Features in TUI

- [x] View labels list
- [x] View tasks by label
- [x] Create new label
- [ ] Edit label
- [ ] Delete label
- [ ] Label colors

---

## Sections

| API Function | TUI Implementation |
|--------------|-------------------|
| [x] GetSections | View sections in project |
| [ ] GetSection | - |
| [ ] CreateSection | Create new section |
| [ ] UpdateSection | Edit section |
| [ ] DeleteSection | Delete section |

### Section Features in TUI

- [x] View sections grouped
- [ ] Create section
- [ ] Edit section
- [ ] Delete section
- [ ] Move tasks between sections

---

## Comments

| API Function | TUI Implementation |
|--------------|-------------------|
| [x] GetComments | View task comments |
| [ ] GetComment | - |
| [ ] CreateComment | Add comment |
| [ ] UpdateComment | Edit comment |
| [ ] DeleteComment | Delete comment |

### Comment Features in TUI

- [x] View task comments
- [ ] Add comment
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

---

## Summary

| Category | Implemented | Total | Coverage |
|----------|-------------|-------|----------|
| Tasks | 7 | 7 | 100% |
| Projects | 3 | 6 | 50% |
| Labels | 2 | 5 | 40% |
| Sections | 1 | 5 | 20% |
| Comments | 1 | 5 | 20% |
| **Overall** | **14** | **28** | **50%** |
