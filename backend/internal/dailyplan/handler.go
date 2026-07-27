package dailyplan

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Application is the daily planning behavior required by the HTML adapter.
type Application interface {
	Today(context.Context) (Today, error)
	AddJournalEntry(context.Context, string) error
	AddTask(context.Context, AddTaskInput) error
	CompleteTask(context.Context, string) error
	AcceptPlan(context.Context, string) error
}

// Handler serves the daily planning web interface.
type Handler struct {
	app    Application
	logger *slog.Logger
	tmpl   *template.Template
	static http.Handler
}

// NewHandler parses embedded assets and constructs the HTML adapter.
func NewHandler(app Application, logger *slog.Logger) (*Handler, error) {
	funcs := template.FuncMap{
		"date": func(value time.Time) string {
			return value.Format("Mon, 02 Jan 2006")
		},
		"dateOnly": func(value time.Time) string {
			return value.Format(time.DateOnly)
		},
		"timeOnly": func(value time.Time) string {
			return value.Format("15:04")
		},
	}
	tmpl, err := template.New("").Funcs(funcs).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse daily planning templates: %w", err)
	}
	staticRoot, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("open daily planning static files: %w", err)
	}
	return &Handler{
		app:    app,
		logger: logger,
		tmpl:   tmpl,
		static: http.StripPrefix("/static/", http.FileServer(http.FS(staticRoot))),
	}, nil
}

// Register mounts the feature's routes on the shared router.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /static/", h.static)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/today", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /today", h.today)
	mux.HandleFunc("POST /journal-entries", h.addJournalEntry)
	mux.HandleFunc("POST /tasks", h.addTask)
	mux.HandleFunc("POST /tasks/{id}/complete", h.completeTask)
	mux.HandleFunc("POST /daily-plans", h.acceptPlan)
}

func (h *Handler) today(w http.ResponseWriter, r *http.Request) {
	view, err := h.app.Today(r.Context())
	if err != nil {
		h.fail(w, "load today", err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "today.html", view); err != nil {
		h.fail(w, "render today", err)
	}
}

func (h *Handler) addJournalEntry(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	if err := h.app.AddJournalEntry(r.Context(), r.FormValue("body")); err != nil {
		h.writeApplicationError(w, err)
		return
	}
	http.Redirect(w, r, "/today#journal", http.StatusSeeOther)
}

func (h *Handler) addTask(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	err := h.app.AddTask(r.Context(), AddTaskInput{
		Title:    r.FormValue("title"),
		Notes:    r.FormValue("notes"),
		Priority: r.FormValue("priority"),
		DueDate:  r.FormValue("due_date"),
	})
	if err != nil {
		h.writeApplicationError(w, err)
		return
	}
	http.Redirect(w, r, "/today#tasks", http.StatusSeeOther)
}

func (h *Handler) completeTask(w http.ResponseWriter, r *http.Request) {
	if err := h.app.CompleteTask(r.Context(), r.PathValue("id")); err != nil {
		h.writeApplicationError(w, err)
		return
	}
	http.Redirect(w, r, "/today#tasks", http.StatusSeeOther)
}

func (h *Handler) acceptPlan(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	if err := h.app.AcceptPlan(r.Context(), r.FormValue("summary")); err != nil {
		h.writeApplicationError(w, err)
		return
	}
	http.Redirect(w, r, "/today#plan", http.StatusSeeOther)
}

func (h *Handler) writeApplicationError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrInvalidInput) {
		http.Error(w, strings.TrimPrefix(err.Error(), ErrInvalidInput.Error()+": "), http.StatusBadRequest)
		return
	}
	if errors.Is(err, ErrNotFound) {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	h.fail(w, "daily planning action", err)
}

func (h *Handler) fail(w http.ResponseWriter, operation string, err error) {
	h.logger.Error("daily planning request failed", "operation", operation, "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
