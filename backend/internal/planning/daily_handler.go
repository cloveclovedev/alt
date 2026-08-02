package planning

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cloveclovedev/alt/internal/core/markdown"
)

//go:embed templates/daily.html templates/components.html
var dailyTemplatesFS embed.FS

// DailyApplication is the web-facing daily planning service boundary.
type DailyApplication interface {
	StartOrResume(context.Context, string) (DailyPlanningView, error)
	RefreshContext(context.Context, string, string) (DailyPlanningView, error)
	ContinueWithoutFailedSources(context.Context, string, string) (DailyPlanningView, error)
	SendMessage(context.Context, string, string, string) (DailyPlanningView, error)
	RetryLastMessage(context.Context, string, string) (DailyPlanningView, error)
	Review(context.Context, string, string) (DailyPlanningView, error)
	BackToChat(context.Context, string, string) (DailyPlanningView, error)
	Confirm(context.Context, string, string) (DailyPlanningView, error)
}

// DailyHandler serves the resumable planning flow. It works as normal forms and as HTMX fragments.
type DailyHandler struct {
	app    DailyApplication
	logger *slog.Logger
	tmpl   *template.Template
}

func NewDailyHandler(app DailyApplication, logger *slog.Logger) (*DailyHandler, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"dateOnly":    func(value time.Time) string { return value.Format(time.DateOnly) },
		"statusLabel": func(status DailySessionStatus) string { return strings.ReplaceAll(string(status), "_", " ") },
		"markdown":    markdown.ToHTML,
	}).ParseFS(dailyTemplatesFS, "templates/daily.html", "templates/components.html")
	if err != nil {
		return nil, fmt.Errorf("parse daily planning template: %w", err)
	}
	return &DailyHandler{app: app, logger: logger, tmpl: tmpl}, nil
}

func (h *DailyHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /planning/daily/{date}", h.show)
	mux.HandleFunc("POST /planning/daily/{date}/context", h.refresh)
	mux.HandleFunc("POST /planning/daily/{date}/continue", h.continueWithout)
	mux.HandleFunc("POST /planning/daily/{date}/messages", h.send)
	mux.HandleFunc("POST /planning/daily/{date}/retry", h.retry)
	mux.HandleFunc("POST /planning/daily/{date}/review", h.review)
	mux.HandleFunc("POST /planning/daily/{date}/chat", h.back)
	mux.HandleFunc("POST /planning/daily/{date}/confirm", h.confirm)
}

func (h *DailyHandler) show(w http.ResponseWriter, r *http.Request) {
	h.run(w, r, func() (DailyPlanningView, error) { return h.app.StartOrResume(r.Context(), r.PathValue("date")) })
}
func (h *DailyHandler) refresh(w http.ResponseWriter, r *http.Request) {
	h.run(w, r, func() (DailyPlanningView, error) {
		return h.app.RefreshContext(r.Context(), r.PathValue("date"), formValue(r, "session_id"))
	})
}
func (h *DailyHandler) continueWithout(w http.ResponseWriter, r *http.Request) {
	h.run(w, r, func() (DailyPlanningView, error) {
		return h.app.ContinueWithoutFailedSources(r.Context(), r.PathValue("date"), formValue(r, "session_id"))
	})
}
func (h *DailyHandler) send(w http.ResponseWriter, r *http.Request) {
	h.run(w, r, func() (DailyPlanningView, error) {
		return h.app.SendMessage(r.Context(), r.PathValue("date"), formValue(r, "session_id"), formValue(r, "content"))
	})
}
func (h *DailyHandler) retry(w http.ResponseWriter, r *http.Request) {
	h.run(w, r, func() (DailyPlanningView, error) {
		return h.app.RetryLastMessage(r.Context(), r.PathValue("date"), formValue(r, "session_id"))
	})
}
func (h *DailyHandler) review(w http.ResponseWriter, r *http.Request) {
	h.run(w, r, func() (DailyPlanningView, error) {
		return h.app.Review(r.Context(), r.PathValue("date"), formValue(r, "session_id"))
	})
}
func (h *DailyHandler) back(w http.ResponseWriter, r *http.Request) {
	h.run(w, r, func() (DailyPlanningView, error) {
		return h.app.BackToChat(r.Context(), r.PathValue("date"), formValue(r, "session_id"))
	})
}
func (h *DailyHandler) confirm(w http.ResponseWriter, r *http.Request) {
	h.run(w, r, func() (DailyPlanningView, error) {
		return h.app.Confirm(r.Context(), r.PathValue("date"), formValue(r, "session_id"))
	})
}

func (h *DailyHandler) run(w http.ResponseWriter, r *http.Request, action func() (DailyPlanningView, error)) {
	view, err := action()
	if err != nil {
		if errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrInvalidSessionState) || errors.Is(err, ErrNotFound) {
			http.Error(w, strings.TrimPrefix(err.Error(), ErrInvalidInput.Error()+": "), http.StatusBadRequest)
			return
		}
		h.logger.Error("daily planning request failed", "error", err)
		http.Error(w, "The request could not be completed. Your saved message remains available to retry.", http.StatusBadGateway)
		return
	}
	if r.Method == http.MethodPost && r.Header.Get("HX-Request") != "true" {
		http.Redirect(w, r, "/planning/daily/"+r.PathValue("date"), http.StatusSeeOther)
		return
	}
	h.render(w, view, r.Header.Get("HX-Request") == "true")
}

func (h *DailyHandler) render(w http.ResponseWriter, view DailyPlanningView, fragment bool) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	name := "daily.html"
	if fragment {
		name = "daily-content"
	}
	if err := h.tmpl.ExecuteTemplate(w, name, view); err != nil {
		h.logger.Error("render daily planning page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func formValue(r *http.Request, key string) string { _ = r.ParseForm(); return r.FormValue(key) }
