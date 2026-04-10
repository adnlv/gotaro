package web

// PageData is passed to HTML templates (no password material).
type PageData struct {
	Title   string
	User    *PageUser
	CSRF    string
	Flash   string
	Error   string
	Data    any
	Active  string
}

type PageUser struct {
	ID    uint64
	Email string
}

type TaskListView struct {
	Tasks           []TaskRow
	CompletedView   bool
	Query           ListQueryView
	SortField       string
	SortDir         string
	HasPrev         bool
	HasNext         bool
	PrevLink        string
	NextLink        string
}

type ListQueryView struct {
	Status   string
	Priority string
	Tag      string
	DueFrom  string
	DueTo    string
	Search   string
}

type TaskRow struct {
	ID          uint64
	Title       string
	Description string
	Status      string
	Priority    string
	CreatedAt   string
	DueDate     string
	Tags        []TagRow
}

type TagRow struct {
	Name  string
	Color string
}

type TaskFormView struct {
	ID          uint64
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     string
	Tags        string
	IsEdit      bool
}
