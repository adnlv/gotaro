package web

import (
	"html/template"

	"github.com/adnlv/gotaro/internal/domain"
)

// PageData is passed to HTML templates (no password material).
type PageData struct {
	Title  string
	User   *PageUser
	CSRF   string
	Flash  string
	Error  string
	Data   any
	Active string
}

type PageUser struct {
	ID    uint64
	Email string
}

type TaskListView struct {
	Tasks          []TaskRow
	CompletedView  bool
	ArchivedView   bool
	Stats          domain.TaskStats
	ActivityChart  template.HTML
	FiltersActive  bool
	ListBasePath   string
	ExportURL      string
	Query          ListQueryView
	SortField      string
	SortDir        string
	HasPrev        bool
	HasNext        bool
	PrevLink       string
	NextLink       string
	ProjectOptions []string
	TagOptions     []string
}

type ListQueryView struct {
	Status   string
	Priority string
	// Accent colors for active filters (empty when "Any").
	StatusAccentBG   string
	PriorityAccentBG string
	Tag              string
	DueFrom          string
	DueTo            string
	Search           string
	Project          string
}

type TaskRow struct {
	ID            uint64
	Title         string
	Description   string
	Status        string
	StatusLabel   string
	StatusBG      string
	StatusFG      string
	Priority      string
	PriorityLabel string
	PriorityBG    string
	PriorityFG    string
	CreatedAt     string
	DueDate       string
	Overdue       bool
	Project       string
	Tags          []TagRow
}

type TagRow struct {
	Name  string
	Color string
}

type TaskFormView struct {
	ID             uint64
	Title          string
	Description    string
	Archived       bool
	Status         string
	StatusLabel    string
	StatusBG       string
	StatusFG       string
	Priority       string
	PriorityLabel  string
	PriorityBG     string
	PriorityFG     string
	DueDate        string
	Project        string
	Tags           string
	ProjectOptions []string
	TagOptions     []string
	IsEdit         bool
}
