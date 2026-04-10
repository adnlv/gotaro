package web

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
	Tasks         []TaskRow
	CompletedView bool
	ArchivedView  bool
	Query         ListQueryView
	SortField     string
	SortDir       string
	HasPrev       bool
	HasNext       bool
	PrevLink      string
	NextLink      string
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
	Tags          []TagRow
}

type TagRow struct {
	Name  string
	Color string
}

type TaskFormView struct {
	ID            uint64
	Title         string
	Description   string
	Archived      bool
	Status        string
	StatusLabel   string
	StatusBG      string
	StatusFG      string
	Priority      string
	PriorityLabel string
	PriorityBG    string
	PriorityFG    string
	DueDate       string
	Tags          string
	IsEdit        bool
}
