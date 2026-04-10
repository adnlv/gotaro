package web

import (
	"context"
	"embed"
	"encoding/csv"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/adnlv/gotaro/internal/app"
	"github.com/adnlv/gotaro/internal/domain"
)

//go:embed templates/*.html
var templateFiles embed.FS

const sessionCookieName = "gotaro_session"

const (
	tplFileLogin     = "login.html"
	tplFileRegister  = "register.html"
	tplFileTasksList = "tasks_list.html"
	tplFileTaskForm  = "task_form.html"
)

type Server struct {
	log    *slog.Logger
	auth   *app.AuthService
	tasks  *app.TaskService
	secure bool

	// liveTemplates: parse HTML from disk on every request (dev). No stale embed; refresh browser to see edits.
	liveTemplates bool
	templateDir   string

	tplLogin     *template.Template
	tplRegister  *template.Template
	tplTasksList *template.Template
	tplTaskForm  *template.Template
}

func NewServer(log *slog.Logger, auth *app.AuthService, tasks *app.TaskService, secureCookie bool) (*Server, error) {
	live := os.Getenv("GOTARO_LIVE_TEMPLATES") == "1"
	tplDir := strings.TrimSpace(os.Getenv("GOTARO_TEMPLATE_DIR"))
	if tplDir == "" {
		tplDir = "internal/web/templates"
	}

	s := &Server{
		log:           log,
		auth:          auth,
		tasks:         tasks,
		secure:        secureCookie,
		liveTemplates: live,
		templateDir:   tplDir,
	}

	if live {
		return s, nil
	}

	parseEmbedded := func(page string) (*template.Template, error) {
		return template.ParseFS(templateFiles, "templates/layout.html", "templates/"+page)
	}
	login, err := parseEmbedded(tplFileLogin)
	if err != nil {
		return nil, err
	}
	reg, err := parseEmbedded(tplFileRegister)
	if err != nil {
		return nil, err
	}
	list, err := parseEmbedded(tplFileTasksList)
	if err != nil {
		return nil, err
	}
	form, err := parseEmbedded(tplFileTaskForm)
	if err != nil {
		return nil, err
	}
	s.tplLogin = login
	s.tplRegister = reg
	s.tplTasksList = list
	s.tplTaskForm = form
	return s, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleRoot)

	mux.Handle("GET /login", s.sessionMiddleware(http.HandlerFunc(s.getLogin)))
	mux.Handle("POST /login", s.chain(http.HandlerFunc(s.postLogin), s.csrf, s.parseForm, s.sessionMiddleware))

	mux.Handle("GET /register", s.sessionMiddleware(http.HandlerFunc(s.getRegister)))
	mux.Handle("POST /register", s.chain(http.HandlerFunc(s.postRegister), s.csrf, s.parseForm, s.sessionMiddleware))

	mux.Handle("POST /logout", s.chain(http.HandlerFunc(s.postLogout), s.csrf, s.parseForm, s.requireAuth, s.sessionMiddleware))

	mux.Handle("GET /tasks/completed", s.chain(http.HandlerFunc(s.getTasksCompleted), s.requireAuth, s.sessionMiddleware))
	mux.Handle("GET /tasks/archived", s.chain(http.HandlerFunc(s.getTasksArchived), s.requireAuth, s.sessionMiddleware))
	mux.Handle("GET /tasks/export.csv", s.chain(http.HandlerFunc(s.getTasksExportCSV), s.requireAuth, s.sessionMiddleware))
	mux.Handle("GET /tasks/new", s.chain(http.HandlerFunc(s.getTaskNew), s.requireAuth, s.sessionMiddleware))
	mux.Handle("POST /tasks", s.chain(http.HandlerFunc(s.postTaskCreate), s.csrf, s.parseForm, s.requireAuth, s.sessionMiddleware))

	mux.Handle("GET /tasks/", s.chain(http.HandlerFunc(s.handleTasksPath), s.requireAuth, s.sessionMiddleware))
	mux.Handle("POST /tasks/", s.chain(http.HandlerFunc(s.handleTasksPost), s.csrf, s.parseForm, s.requireAuth, s.sessionMiddleware))

	mux.Handle("GET /tasks", s.chain(http.HandlerFunc(s.getTasksOpen), s.requireAuth, s.sessionMiddleware))

	return mux
}

// chain wraps handler with middleware in order: the first middleware in mw is the innermost
// (runs closest to the handler); the last is outermost (runs first on the request).
func (s *Server) chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for _, m := range mw {
		h = m(h)
	}
	return h
}

func (s *Server) parseForm(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		sess, ok := sessionFrom(r.Context())
		if !ok || sess == nil {
			next.ServeHTTP(w, r)
			return
		}
		if r.FormValue("csrf_token") != sess.CSRFToken {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		c, err := r.Cookie(sessionCookieName)
		if err != nil || strings.TrimSpace(c.Value) == "" {
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		sess, err := s.auth.Session(ctx, c.Value)
		if err != nil {
			if !errors.Is(err, domain.ErrUnauthorized) {
				s.log.Error("session", "err", err)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		next.ServeHTTP(w, r.WithContext(withSession(ctx, sess)))
	})
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := sessionFrom(r.Context())
		if !ok || sess == nil || sess.User == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

func (s *Server) pageUser(sess *domain.Session) *PageUser {
	if sess == nil || sess.User == nil {
		return nil
	}
	return &PageUser{ID: sess.User.ID, Email: sess.User.Email}
}

func (s *Server) loadPageTemplates(pageFile string) (*template.Template, error) {
	if s.liveTemplates {
		return template.ParseFS(os.DirFS(s.templateDir), "layout.html", pageFile)
	}
	switch pageFile {
	case tplFileLogin:
		return s.tplLogin, nil
	case tplFileRegister:
		return s.tplRegister, nil
	case tplFileTasksList:
		return s.tplTasksList, nil
	case tplFileTaskForm:
		return s.tplTaskForm, nil
	default:
		return nil, fmt.Errorf("unknown template %q", pageFile)
	}
}

func (s *Server) render(w http.ResponseWriter, pageFile string, d PageData) {
	tpl, err := s.loadPageTemplates(pageFile)
	if err != nil {
		s.log.Error("template load", "page", pageFile, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if s.liveTemplates {
		w.Header().Set("Cache-Control", "no-store")
	}
	if err := tpl.ExecuteTemplate(w, "layout", d); err != nil {
		s.log.Error("render", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (s *Server) setSessionCookie(w http.ResponseWriter, sess *domain.Session) {
	maxAge := int(time.Until(sess.ExpiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.Token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secure,
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secure,
	})
}

func (s *Server) getLogin(w http.ResponseWriter, r *http.Request) {
	if sess, ok := sessionFrom(r.Context()); ok && sess != nil {
		http.Redirect(w, r, "/tasks", http.StatusSeeOther)
		return
	}
	s.render(w, tplFileLogin, PageData{Title: "Log in", Flash: flashMessage(r.URL.Query().Get("flash"))})
}

func (s *Server) postLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, err := s.auth.Login(ctx, r.FormValue("email"), r.FormValue("password"), r.RemoteAddr, r.UserAgent())
	if err != nil {
		msg := friendlyError(err)
		if errors.Is(err, domain.ErrInvalidInput) {
			msg = err.Error()
		}
		s.render(w, tplFileLogin, PageData{Title: "Log in", Error: msg})
		return
	}
	s.setSessionCookie(w, sess)
	http.Redirect(w, r, "/tasks?flash=logged_in", http.StatusSeeOther)
}

func (s *Server) getRegister(w http.ResponseWriter, r *http.Request) {
	if sess, ok := sessionFrom(r.Context()); ok && sess != nil {
		http.Redirect(w, r, "/tasks", http.StatusSeeOther)
		return
	}
	s.render(w, tplFileRegister, PageData{Title: "Register"})
}

func (s *Server) postRegister(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, err := s.auth.Register(ctx, r.FormValue("email"), r.FormValue("password"), r.RemoteAddr, r.UserAgent())
	if err != nil {
		msg := friendlyError(err)
		if errors.Is(err, domain.ErrInvalidInput) {
			msg = err.Error()
		}
		s.render(w, tplFileRegister, PageData{Title: "Register", Error: msg})
		return
	}
	s.setSessionCookie(w, sess)
	http.Redirect(w, r, "/tasks?flash=registered", http.StatusSeeOther)
}

func (s *Server) postLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, _ := sessionFrom(r.Context())
	if sess != nil {
		_ = s.auth.Logout(ctx, sess.Token)
	}
	s.clearSessionCookie(w)
	http.Redirect(w, r, "/login?flash=logged_out", http.StatusSeeOther)
}

func (s *Server) getTasksOpen(w http.ResponseWriter, r *http.Request) {
	s.renderTaskList(w, r, false, false)
}

func (s *Server) getTasksCompleted(w http.ResponseWriter, r *http.Request) {
	s.renderTaskList(w, r, true, false)
}

func (s *Server) getTasksArchived(w http.ResponseWriter, r *http.Request) {
	s.renderTaskList(w, r, false, true)
}

func (s *Server) getTasksExportCSV(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, _ := sessionFrom(ctx)
	q := r.URL.Query()
	view := q.Get("view")
	completed := view == "completed"
	archived := view == "archived"

	st, err := optionalStatus(q.Get("status"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	pr, err := optionalPriority(q.Get("priority"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	dueFrom, err := parseDatePtr(q.Get("due_from"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	dueTo, err := parseDatePtr(q.Get("due_to"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	params := app.TaskListParams{
		UserID:        sess.User.ID,
		CompletedView: completed,
		ArchivedView:  archived,
		Status:        st,
		Priority:      pr,
		Tag:           q.Get("tag"),
		Project:       q.Get("project"),
		DueFrom:       dueFrom,
		DueTo:         dueTo,
		Search:        q.Get("q"),
		SortField:     firstNonEmpty(q.Get("sort"), "created_at"),
		SortDir:       firstNonEmpty(q.Get("dir"), "desc"),
	}

	tasks, err := s.tasks.ListAllMatching(ctx, params)
	if err != nil {
		s.log.Error("export tasks", "err", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="gotaro-tasks.csv"`)
	cw := csv.NewWriter(w)
	header := []string{"id", "title", "description", "status", "priority", "project", "tags", "due_date", "archived_at", "created_at", "updated_at"}
	if err := cw.Write(header); err != nil {
		return
	}
	for _, t := range tasks {
		desc := ""
		if t.Description != nil {
			desc = *t.Description
		}
		tagNames := make([]string, 0, len(t.Tags))
		for _, tg := range t.Tags {
			tagNames = append(tagNames, tg.Name)
		}
		proj := ""
		if t.Project != nil {
			proj = t.Project.Name
		}
		due := ""
		if t.DueDate != nil {
			due = t.DueDate.UTC().Format(time.RFC3339)
		}
		arch := ""
		if t.ArchivedAt != nil {
			arch = t.ArchivedAt.UTC().Format(time.RFC3339)
		}
		row := []string{
			strconv.FormatUint(t.ID, 10),
			t.Title,
			desc,
			t.Status.String(),
			t.Priority.String(),
			proj,
			strings.Join(tagNames, "; "),
			due,
			arch,
			t.CreatedAt.UTC().Format(time.RFC3339),
			t.UpdatedAt.UTC().Format(time.RFC3339),
		}
		if err := cw.Write(row); err != nil {
			return
		}
	}
	cw.Flush()
}

func (s *Server) renderTaskList(w http.ResponseWriter, r *http.Request, completed, archived bool) {
	ctx := r.Context()
	sess, _ := sessionFrom(r.Context())

	st, _ := optionalStatus(r.URL.Query().Get("status"))
	pr, _ := optionalPriority(r.URL.Query().Get("priority"))
	dueFrom, _ := parseDatePtr(r.URL.Query().Get("due_from"))
	dueTo, _ := parseDatePtr(r.URL.Query().Get("due_to"))

	limit := 25
	offset := intQuery(r, "offset", 0)
	if offset < 0 {
		offset = 0
	}

	params := app.TaskListParams{
		UserID:        sess.User.ID,
		CompletedView: completed,
		ArchivedView:  archived,
		Status:        st,
		Priority:      pr,
		Tag:           r.URL.Query().Get("tag"),
		Project:       r.URL.Query().Get("project"),
		DueFrom:       dueFrom,
		DueTo:         dueTo,
		Search:        r.URL.Query().Get("q"),
		SortField:     r.URL.Query().Get("sort"),
		SortDir:       r.URL.Query().Get("dir"),
		Offset:        offset,
		Limit:         limit,
	}

	tasks, err := s.tasks.List(ctx, params)
	if err != nil {
		s.log.Error("list tasks", "err", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	stats, err := s.tasks.Stats(ctx, sess.User.ID)
	if err != nil {
		s.log.Error("task stats", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var activity []domain.DailyActivityPoint
	act, err := s.tasks.DailyActivity(ctx, sess.User.ID, taskActivityChartDays)
	if err != nil {
		s.log.Error("daily activity", "err", err)
	} else {
		activity = act
	}

	hasMore := len(tasks) == limit
	listPath := "open"
	if archived {
		listPath = "archived"
	} else if completed {
		listPath = "completed"
	}
	listBase := "/tasks"
	if archived {
		listBase = "/tasks/archived"
	} else if completed {
		listBase = "/tasks/completed"
	}
	prev, next, hasPrev, hasNext := paginationLinks(r, listPath, limit, offset, hasMore)

	qv := ListQueryView{
		Status:   r.URL.Query().Get("status"),
		Priority: r.URL.Query().Get("priority"),
		Tag:      r.URL.Query().Get("tag"),
		Project:  r.URL.Query().Get("project"),
		DueFrom:  r.URL.Query().Get("due_from"),
		DueTo:    r.URL.Query().Get("due_to"),
		Search:   r.URL.Query().Get("q"),
	}
	if strings.TrimSpace(qv.Status) != "" {
		_, qv.StatusAccentBG, _ = statusPresentationFromSlug(qv.Status)
	}
	if strings.TrimSpace(qv.Priority) != "" {
		_, qv.PriorityAccentBG, _ = priorityPresentationFromSlug(qv.Priority)
	}
	projectOptions, err := s.tasks.ExistingProjectNames(ctx, sess.User.ID)
	if err != nil {
		s.log.Error("list project filter suggestions", "err", err)
	}
	tagOptions, err := s.tasks.ExistingTagNames(ctx, sess.User.ID)
	if err != nil {
		s.log.Error("list tag filter suggestions", "err", err)
	}

	view := TaskListView{
		Tasks:          taskRows(tasks, time.Now().UTC()),
		CompletedView:  completed,
		ArchivedView:   archived,
		Stats:          stats,
		ActivityChart:  activityChartSVG(activity),
		FiltersActive:  taskListFiltersActive(r),
		ListBasePath:   listBase,
		ExportURL:      taskExportPath(r, completed, archived),
		Query:          qv,
		SortField:      firstNonEmpty(r.URL.Query().Get("sort"), "created_at"),
		SortDir:        firstNonEmpty(r.URL.Query().Get("dir"), "desc"),
		HasPrev:        hasPrev,
		HasNext:        hasNext,
		PrevLink:       prev,
		NextLink:       next,
		ProjectOptions: projectOptions,
		TagOptions:     tagOptions,
	}

	s.render(w, tplFileTasksList, PageData{
		Title:  "Tasks",
		User:   s.pageUser(sess),
		CSRF:   sess.CSRFToken,
		Flash:  flashMessage(r.URL.Query().Get("flash")),
		Data:   view,
		Active: "tasks",
	})
}

func firstNonEmpty(a, b string) string {
	a = strings.TrimSpace(a)
	if a != "" {
		return a
	}
	return b
}

func taskRows(tasks []domain.Task, now time.Time) []TaskRow {
	out := make([]TaskRow, 0, len(tasks))
	for _, t := range tasks {
		sl, sbg, sfg := statusPresentation(t.Status)
		pl, pbg, pfg := priorityPresentation(t.Priority)
		row := TaskRow{
			ID:            t.ID,
			Title:         t.Title,
			Status:        t.Status.String(),
			StatusLabel:   sl,
			StatusBG:      sbg,
			StatusFG:      sfg,
			Priority:      t.Priority.String(),
			PriorityLabel: pl,
			PriorityBG:    pbg,
			PriorityFG:    pfg,
			CreatedAt:     formatTaskCreated(t.CreatedAt),
		}
		if t.Description != nil {
			row.Description = *t.Description
		}
		if t.DueDate != nil {
			row.DueDate = formatTaskDue(*t.DueDate, now)
		}
		row.Overdue = t.IsOverdue(now)
		if t.Project != nil {
			row.Project = t.Project.Name
		}
		for _, tg := range t.Tags {
			c := strings.TrimSpace(tg.Color)
			if c == "" {
				c = "#6c757d"
			}
			row.Tags = append(row.Tags, TagRow{Name: tg.Name, Color: c})
		}
		out = append(out, row)
	}
	return out
}

func (s *Server) getTaskNew(w http.ResponseWriter, r *http.Request) {
	sess, _ := sessionFrom(r.Context())
	fv := TaskFormView{
		Status:   domain.StatusTodo.String(),
		Priority: domain.PriorityMedium.String(),
	}
	decorateTaskFormColors(&fv)
	s.decorateTaskFormSuggestions(r.Context(), sess.User.ID, &fv)
	s.render(w, tplFileTaskForm, PageData{
		Title:  "New task",
		User:   s.pageUser(sess),
		CSRF:   sess.CSRFToken,
		Data:   fv,
		Active: "tasks",
	})
}

func (s *Server) postTaskCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, _ := sessionFrom(r.Context())
	tw, err := taskWriteFromForm(r)
	if err != nil {
		fv := taskFormFromRequest(r)
		s.decorateTaskFormSuggestions(ctx, sess.User.ID, &fv)
		s.render(w, tplFileTaskForm, PageData{
			Title:  "New task",
			User:   s.pageUser(sess),
			CSRF:   sess.CSRFToken,
			Error:  friendlyError(err),
			Data:   fv,
			Active: "tasks",
		})
		return
	}
	_, err = s.tasks.Create(ctx, sess.User.ID, tw)
	if err != nil {
		fv := taskFormFromRequest(r)
		s.decorateTaskFormSuggestions(ctx, sess.User.ID, &fv)
		s.render(w, tplFileTaskForm, PageData{
			Title:  "New task",
			User:   s.pageUser(sess),
			CSRF:   sess.CSRFToken,
			Error:  friendlyError(err),
			Data:   fv,
			Active: "tasks",
		})
		return
	}
	http.Redirect(w, r, "/tasks?flash=task_created", http.StatusSeeOther)
}

func taskFormFromRequest(r *http.Request) TaskFormView {
	fv := TaskFormView{
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Status:      r.FormValue("status"),
		Priority:    r.FormValue("priority"),
		DueDate:     r.FormValue("due_date"),
		Project:     r.FormValue("project"),
		Tags:        r.FormValue("tags"),
	}
	decorateTaskFormColors(&fv)
	return fv
}

func (s *Server) handleTasksPath(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, _ := sessionFrom(r.Context())
	sub := strings.Trim(strings.TrimPrefix(r.URL.Path, "/tasks/"), "/")
	if sub == "" {
		http.Redirect(w, r, "/tasks", http.StatusSeeOther)
		return
	}
	if after, ok := strings.CutSuffix(sub, "/edit"); ok {
		id, err := strconv.ParseUint(after, 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		t, err := s.tasks.Get(ctx, sess.User.ID, id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		s.render(w, tplFileTaskForm, PageData{
			Title:  "Edit task",
			User:   s.pageUser(sess),
			CSRF:   sess.CSRFToken,
			Data:   s.taskToFormViewWithSuggestions(ctx, sess.User.ID, t, true),
			Active: "tasks",
		})
		return
	}
	http.NotFound(w, r)
}

func taskToFormView(t *domain.Task, edit bool) TaskFormView {
	fv := TaskFormView{
		ID:       t.ID,
		Title:    t.Title,
		Archived: t.IsArchived(),
		Status:   t.Status.String(),
		Priority: t.Priority.String(),
		IsEdit:   edit,
	}
	if t.Description != nil {
		fv.Description = *t.Description
	}
	if t.DueDate != nil {
		fv.DueDate = t.DueDate.UTC().Format("2006-01-02")
	}
	names := make([]string, 0, len(t.Tags))
	for _, tg := range t.Tags {
		names = append(names, tg.Name)
	}
	fv.Tags = strings.Join(names, ", ")
	if t.Project != nil {
		fv.Project = t.Project.Name
	}
	decorateTaskFormColors(&fv)
	return fv
}

func (s *Server) handleTasksPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, _ := sessionFrom(r.Context())
	sub := strings.Trim(strings.TrimPrefix(r.URL.Path, "/tasks/"), "/")

	if after, ok := strings.CutSuffix(sub, "/archive"); ok {
		id, err := strconv.ParseUint(after, 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if err := s.tasks.Archive(ctx, sess.User.ID, id); err != nil {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/tasks?flash=task_archived", http.StatusSeeOther)
		return
	}

	if after, ok := strings.CutSuffix(sub, "/unarchive"); ok {
		id, err := strconv.ParseUint(after, 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if err := s.tasks.Unarchive(ctx, sess.User.ID, id); err != nil {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/tasks?flash=task_unarchived", http.StatusSeeOther)
		return
	}

	if after, ok := strings.CutSuffix(sub, "/delete"); ok {
		id, err := strconv.ParseUint(after, 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if err := s.tasks.Delete(ctx, sess.User.ID, id); err != nil {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/tasks?flash=task_deleted", http.StatusSeeOther)
		return
	}

	if after, ok := strings.CutSuffix(sub, "/status"); ok {
		id, err := strconv.ParseUint(after, 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		st, ok := domain.StatusFromString(r.FormValue("status"))
		if !ok {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := s.tasks.SetStatus(ctx, sess.User.ID, id, st); err != nil {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/tasks", http.StatusSeeOther)
		return
	}

	id, err := strconv.ParseUint(sub, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	tw, err := taskWriteFromForm(r)
	if err != nil {
		t, gerr := s.tasks.Get(ctx, sess.User.ID, id)
		if gerr != nil {
			http.NotFound(w, r)
			return
		}
		fv := taskToFormView(t, true)
		s.decorateTaskFormSuggestions(ctx, sess.User.ID, &fv)
		s.render(w, tplFileTaskForm, PageData{
			Title:  "Edit task",
			User:   s.pageUser(sess),
			CSRF:   sess.CSRFToken,
			Error:  friendlyError(err),
			Data:   fv,
			Active: "tasks",
		})
		return
	}
	_, err = s.tasks.Update(ctx, sess.User.ID, id, tw)
	if err != nil {
		t, gerr := s.tasks.Get(ctx, sess.User.ID, id)
		if gerr != nil {
			http.NotFound(w, r)
			return
		}
		fv := taskToFormView(t, true)
		s.decorateTaskFormSuggestions(ctx, sess.User.ID, &fv)
		s.render(w, tplFileTaskForm, PageData{
			Title:  "Edit task",
			User:   s.pageUser(sess),
			CSRF:   sess.CSRFToken,
			Error:  friendlyError(err),
			Data:   fv,
			Active: "tasks",
		})
		return
	}
	http.Redirect(w, r, "/tasks?flash=task_updated", http.StatusSeeOther)
}

func (s *Server) taskToFormViewWithSuggestions(ctx context.Context, userID uint64, t *domain.Task, edit bool) TaskFormView {
	fv := taskToFormView(t, edit)
	s.decorateTaskFormSuggestions(ctx, userID, &fv)
	return fv
}

func (s *Server) decorateTaskFormSuggestions(ctx context.Context, userID uint64, fv *TaskFormView) {
	projs, err := s.tasks.ExistingProjectNames(ctx, userID)
	if err != nil {
		s.log.Error("list project suggestions", "err", err)
	} else {
		fv.ProjectOptions = projs
	}
	tags, err := s.tasks.ExistingTagNames(ctx, userID)
	if err != nil {
		s.log.Error("list tag suggestions", "err", err)
	} else {
		fv.TagOptions = tags
	}
}

// ListenAndServe is a thin wrapper for http.Server using this handler.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: s.Handler(),
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
