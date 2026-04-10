package web

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/adnlv/gotaro/internal/app"
	"github.com/adnlv/gotaro/internal/domain"
)

func parseDatePtr(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func optionalStatus(s string) (*domain.Status, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	st, ok := domain.StatusFromString(s)
	if !ok {
		return nil, fmt.Errorf("invalid status")
	}
	return &st, nil
}

func optionalPriority(s string) (*domain.Priority, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	pr, ok := domain.PriorityFromString(s)
	if !ok {
		return nil, fmt.Errorf("invalid priority")
	}
	return &pr, nil
}

func descriptionPtr(raw string) *string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return &raw
}

func taskWriteFromForm(r *http.Request) (app.TaskWrite, error) {
	if err := r.ParseForm(); err != nil {
		return app.TaskWrite{}, err
	}
	st, ok := domain.StatusFromString(r.FormValue("status"))
	if !ok {
		return app.TaskWrite{}, fmt.Errorf("%w: invalid status", domain.ErrInvalidInput)
	}
	pr, ok := domain.PriorityFromString(r.FormValue("priority"))
	if !ok {
		return app.TaskWrite{}, fmt.Errorf("%w: invalid priority", domain.ErrInvalidInput)
	}
	due, err := parseDatePtr(r.FormValue("due_date"))
	if err != nil {
		return app.TaskWrite{}, fmt.Errorf("%w: invalid due date", domain.ErrInvalidInput)
	}
	desc := descriptionPtr(r.FormValue("description"))
	tags, err := domain.ParseTagList(r.FormValue("tags"))
	if err != nil {
		return app.TaskWrite{}, err
	}
	return app.TaskWrite{
		Title:       r.FormValue("title"),
		Description: desc,
		Status:      st,
		Priority:    pr,
		DueDate:     due,
		TagNames:    tags,
	}, nil
}

func intQuery(r *http.Request, key string, def int) int {
	s := strings.TrimSpace(r.URL.Query().Get(key))
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// listPath is "open", "completed", or "archived".
func paginationLinks(r *http.Request, listPath string, limit, offset int, hasMore bool) (prev, next string, hasPrev, hasNext bool) {
	var base string
	switch listPath {
	case "completed":
		base = "/tasks/completed"
	case "archived":
		base = "/tasks/archived"
	default:
		base = "/tasks"
	}
	q := r.URL.Query()
	sort := q.Get("sort")
	if sort == "" {
		sort = "created_at"
	}
	dir := q.Get("dir")
	if dir == "" {
		dir = "desc"
	}

	build := func(off int) string {
		v := url.Values{}
		for k, vals := range q {
			if k == "page" {
				continue
			}
			for _, val := range vals {
				v.Add(k, val)
			}
		}
		v.Set("sort", sort)
		v.Set("dir", dir)
		if off > 0 {
			v.Set("offset", strconv.Itoa(off))
		}
		qs := v.Encode()
		if qs == "" {
			return base
		}
		return base + "?" + qs
	}

	if offset > 0 {
		prevOff := offset - limit
		if prevOff < 0 {
			prevOff = 0
		}
		prev = build(prevOff)
		hasPrev = true
	}
	if hasMore {
		next = build(offset + limit)
		hasNext = true
	}
	return prev, next, hasPrev, hasNext
}

func flashMessage(code string) string {
	switch code {
	case "registered":
		return "Account created. You are logged in."
	case "logged_in":
		return "Welcome back."
	case "logged_out":
		return "You have been logged out."
	case "task_created":
		return "Task created."
	case "task_updated":
		return "Task updated."
	case "task_deleted":
		return "Task deleted."
	case "task_archived":
		return "Task archived."
	case "task_unarchived":
		return "Task restored from archive."
	default:
		return ""
	}
}

func friendlyError(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return err.Error()
	case errors.Is(err, domain.ErrDuplicateEmail):
		return "That email is already registered."
	case errors.Is(err, domain.ErrUnauthorized):
		return "Invalid email or password."
	case errors.Is(err, domain.ErrNotFound):
		return "That item was not found."
	default:
		return "Something went wrong. Please try again."
	}
}
