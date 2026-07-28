package planning

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

// Application is the planning behavior required by the HTML adapter.
type Application interface {
	Home(context.Context) (View, error)
	ViewPlan(context.Context, string, string) (View, error)
	SavePlan(context.Context, SavePlanInput) error
}

// Handler serves the planning web interface.
type Handler struct {
	app    Application
	logger *slog.Logger
	tmpl   *template.Template
	static http.Handler
}

// NewHandler parses embedded assets and constructs the HTML adapter.
func NewHandler(app Application, logger *slog.Logger) (*Handler, error) {
	funcs := template.FuncMap{
		"dateOnly": func(value time.Time) string {
			return value.Format(time.DateOnly)
		},
		"periodLabel": func(kind Kind, value time.Time) string {
			if kind == KindWeekly {
				return "Week of " + value.Format("Mon, 02 Jan 2006")
			}
			return value.Format("Mon, 02 Jan 2006")
		},
		"planTitle": func(kind Kind) string {
			if kind == KindWeekly {
				return "Weekly plan"
			}
			return "Daily plan"
		},
	}
	tmpl, err := template.New("").Funcs(funcs).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse planning templates: %w", err)
	}
	staticRoot, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("open planning static files: %w", err)
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
	mux.HandleFunc("GET /{$}", h.home)
	mux.HandleFunc("GET /plans/{kind}/{period_start}", h.plan)
	mux.HandleFunc("POST /plans/{kind}/{period_start}/revisions", h.savePlan)
}

func (h *Handler) home(w http.ResponseWriter, r *http.Request) {
	view, err := h.app.Home(r.Context())
	if err != nil {
		h.fail(w, "load home", err)
		return
	}
	h.renderPlan(w, view)
}

func (h *Handler) plan(w http.ResponseWriter, r *http.Request) {
	view, err := h.app.ViewPlan(
		r.Context(),
		r.PathValue("kind"),
		r.PathValue("period_start"),
	)
	if err != nil {
		h.writeApplicationError(w, err)
		return
	}
	h.renderPlan(w, view)
}

func (h *Handler) savePlan(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	kind := r.PathValue("kind")
	periodStart := r.PathValue("period_start")
	if err := h.app.SavePlan(r.Context(), SavePlanInput{
		Kind:            kind,
		PeriodStart:     periodStart,
		SummaryMarkdown: r.FormValue("summary_markdown"),
		ContentMarkdown: r.FormValue("content_markdown"),
	}); err != nil {
		h.writeApplicationError(w, err)
		return
	}
	http.Redirect(
		w,
		r,
		fmt.Sprintf("/plans/%s/%s#plan", kind, periodStart),
		http.StatusSeeOther,
	)
}

func (h *Handler) renderPlan(w http.ResponseWriter, view View) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "plan.html", view); err != nil {
		h.fail(w, "render plan", err)
	}
}

func (h *Handler) writeApplicationError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrInvalidInput) {
		http.Error(
			w,
			strings.TrimPrefix(err.Error(), ErrInvalidInput.Error()+": "),
			http.StatusBadRequest,
		)
		return
	}
	h.fail(w, "planning action", err)
}

func (h *Handler) fail(w http.ResponseWriter, operation string, err error) {
	h.logger.Error("planning request failed", "operation", operation, "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
