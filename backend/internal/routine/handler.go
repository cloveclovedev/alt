package routine

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Application is the routine behavior required by the HTML adapter.
type Application interface {
	List(context.Context, bool) (ListView, error)
	NewForm(context.Context) (NewRoutineView, error)
	Detail(context.Context, string) (DetailView, error)
	Create(context.Context, RoutineInput) (string, error)
	Update(context.Context, string, RoutineInput) error
	Delete(context.Context, string) error
	Complete(context.Context, string, CompletionInput) error
	UpdateEvent(context.Context, string, string, EventInput) error
	DeleteEvent(context.Context, string, string) error
	Categories(context.Context) ([]Category, error)
	CreateCategory(context.Context, CategoryInput) error
	UpdateCategory(context.Context, string, CategoryInput) error
	DeleteCategory(context.Context, string) error
}

// Handler serves the routine HTML interface.
type Handler struct {
	app    Application
	logger *slog.Logger
	tmpl   *template.Template
}

// NewHandler parses embedded routine templates.
func NewHandler(app Application, logger *slog.Logger) (*Handler, error) {
	funcs := template.FuncMap{
		"dateOnly": func(value time.Time) string { return value.Format(time.DateOnly) },
		"hasInt16": func(values []int16, want int) bool {
			for _, value := range values {
				if value == int16(want) {
					return true
				}
			}
			return false
		},
		"slice": func(values ...int) []int { return values },
		"stateLabel": func(state DueState) string {
			switch state {
			case DueStateOverdue:
				return "Overdue"
			case DueStateToday:
				return "Today"
			default:
				return "Upcoming"
			}
		},
	}
	tmpl, err := template.New("").Funcs(funcs).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse routine templates: %w", err)
	}
	return &Handler{app: app, logger: logger, tmpl: tmpl}, nil
}

// Register mounts routine routes on the shared router.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /routines", h.list)
	mux.HandleFunc("GET /routines/new", h.newForm)
	mux.HandleFunc("POST /routines", h.create)
	mux.HandleFunc("GET /routines/{routine_id}", h.detail)
	mux.HandleFunc("POST /routines/{routine_id}", h.update)
	mux.HandleFunc("POST /routines/{routine_id}/delete", h.delete)
	mux.HandleFunc("POST /routines/{routine_id}/complete", h.complete)
	mux.HandleFunc("POST /routines/{routine_id}/events/{event_id}", h.updateEvent)
	mux.HandleFunc("POST /routines/{routine_id}/events/{event_id}/delete", h.deleteEvent)
	mux.HandleFunc("GET /routine-categories", h.categories)
	mux.HandleFunc("POST /routine-categories", h.createCategory)
	mux.HandleFunc("POST /routine-categories/{category_id}", h.updateCategory)
	mux.HandleFunc("POST /routine-categories/{category_id}/delete", h.deleteCategory)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	view, err := h.app.List(r.Context(), r.URL.Query().Get("inactive") == "1")
	if err != nil {
		h.fail(w, "list routines", err)
		return
	}
	h.render(w, "list.html", view)
}

func (h *Handler) newForm(w http.ResponseWriter, r *http.Request) {
	view, err := h.app.NewForm(r.Context())
	if err != nil {
		h.fail(w, "load routine form", err)
		return
	}
	h.render(w, "form.html", view)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	input, err := h.parseRoutineInput(r, true)
	if err == nil {
		var routineID string
		routineID, err = h.app.Create(r.Context(), input)
		if err == nil {
			http.Redirect(w, r, "/routines/"+routineID, http.StatusSeeOther)
			return
		}
	}
	h.writeApplicationError(w, err)
}

func (h *Handler) detail(w http.ResponseWriter, r *http.Request) {
	view, err := h.app.Detail(r.Context(), r.PathValue("routine_id"))
	if err != nil {
		h.writeApplicationError(w, err)
		return
	}
	h.render(w, "detail.html", view)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	input, err := h.parseRoutineInput(r, false)
	if err == nil {
		err = h.app.Update(r.Context(), r.PathValue("routine_id"), input)
	}
	if err != nil {
		h.writeApplicationError(w, err)
		return
	}
	http.Redirect(w, r, "/routines/"+r.PathValue("routine_id"), http.StatusSeeOther)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	err := h.app.Delete(r.Context(), r.PathValue("routine_id"))
	if err != nil {
		h.writeApplicationError(w, err)
		return
	}
	http.Redirect(w, r, "/routines", http.StatusSeeOther)
}

func (h *Handler) complete(w http.ResponseWriter, r *http.Request) {
	routineID := r.PathValue("routine_id")
	input, err := h.parseCompletionInput(r)
	if err == nil {
		err = h.app.Complete(r.Context(), routineID, input)
	}
	if err != nil {
		h.writeApplicationError(w, err)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		h.renderCompleteFragment(w, r, routineID, input.Note)
		return
	}
	http.Redirect(w, r, "/routines/"+routineID, http.StatusSeeOther)
}

// renderCompleteFragment swaps in the small "done" state after an HTMX
// completion request, so an embedding page (the planning home card) can
// complete a routine without a full-page navigation.
func (h *Handler) renderCompleteFragment(w http.ResponseWriter, r *http.Request, routineID, note string) {
	detail, err := h.app.Detail(r.Context(), routineID)
	if err != nil {
		h.writeApplicationError(w, err)
		return
	}
	h.render(w, "routine-complete-fragment", CompleteFragmentView{
		RoutineID:    routineID,
		Name:         detail.Routine.Name,
		CategoryName: detail.Routine.CategoryName,
		Note:         note,
	})
}

func (h *Handler) updateEvent(w http.ResponseWriter, r *http.Request) {
	input, err := h.parseEventInput(r)
	if err == nil {
		err = h.app.UpdateEvent(r.Context(), r.PathValue("routine_id"), r.PathValue("event_id"), input)
	}
	if err != nil {
		h.writeApplicationError(w, err)
		return
	}
	http.Redirect(w, r, "/routines/"+r.PathValue("routine_id")+"#history", http.StatusSeeOther)
}

func (h *Handler) deleteEvent(w http.ResponseWriter, r *http.Request) {
	err := h.app.DeleteEvent(r.Context(), r.PathValue("routine_id"), r.PathValue("event_id"))
	if err != nil {
		h.writeApplicationError(w, err)
		return
	}
	http.Redirect(w, r, "/routines/"+r.PathValue("routine_id")+"#history", http.StatusSeeOther)
}

func (h *Handler) categories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.app.Categories(r.Context())
	if err != nil {
		h.fail(w, "list routine categories", err)
		return
	}
	h.render(w, "categories.html", categories)
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	input, err := h.parseCategoryInput(r)
	if err == nil {
		err = h.app.CreateCategory(r.Context(), input)
	}
	if err != nil {
		h.writeApplicationError(w, err)
		return
	}
	http.Redirect(w, r, "/routine-categories", http.StatusSeeOther)
}

func (h *Handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	input, err := h.parseCategoryInput(r)
	if err == nil {
		err = h.app.UpdateCategory(r.Context(), r.PathValue("category_id"), input)
	}
	if err != nil {
		h.writeApplicationError(w, err)
		return
	}
	http.Redirect(w, r, "/routine-categories", http.StatusSeeOther)
}

func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	err := h.app.DeleteCategory(r.Context(), r.PathValue("category_id"))
	if err != nil {
		h.writeApplicationError(w, err)
		return
	}
	http.Redirect(w, r, "/routine-categories", http.StatusSeeOther)
}

func (h *Handler) parseRoutineInput(r *http.Request, creating bool) (RoutineInput, error) {
	if err := r.ParseForm(); err != nil {
		return RoutineInput{}, fmt.Errorf("%w: invalid form", ErrInvalidInput)
	}
	interval, err := strconv.Atoi(strings.TrimSpace(r.FormValue("interval_days")))
	if err != nil {
		return RoutineInput{}, fmt.Errorf("%w: interval must be a whole number", ErrInvalidInput)
	}
	weekdays, err := parseInt16Values(r.Form["available_weekdays"])
	if err != nil {
		return RoutineInput{}, err
	}
	months, err := parseInt16Values(r.Form["active_months"])
	if err != nil {
		return RoutineInput{}, err
	}
	input := RoutineInput{
		CategoryID: r.FormValue("category_id"), Name: r.FormValue("name"), Notes: r.FormValue("notes"),
		Status: RoutineStatusActive, IntervalDays: interval, AvailableWeekdays: weekdays, ActiveMonths: months,
	}
	if r.FormValue("status") == string(RoutineStatusInactive) {
		input.Status = RoutineStatusInactive
	}
	if creating && r.FormValue("never_completed") != "on" {
		lastCompleted, err := parseDate(r.FormValue("last_completed_on"))
		if err != nil {
			return RoutineInput{}, fmt.Errorf("%w: last completed date is required", ErrInvalidInput)
		}
		input.LastCompletedOn = &lastCompleted
	}
	return input, nil
}

// parseCompletionInput leaves CompletedOn zero when the field is absent, so a
// quick completion (no date input, e.g. the planning home card) defaults to
// today in the service layer. A non-empty value must still parse.
func (h *Handler) parseCompletionInput(r *http.Request) (CompletionInput, error) {
	if err := r.ParseForm(); err != nil {
		return CompletionInput{}, fmt.Errorf("%w: invalid form", ErrInvalidInput)
	}
	raw := strings.TrimSpace(r.FormValue("completed_on"))
	if raw == "" {
		return CompletionInput{Note: r.FormValue("note")}, nil
	}
	date, err := parseDate(raw)
	if err != nil {
		return CompletionInput{}, fmt.Errorf("%w: completion date is required", ErrInvalidInput)
	}
	return CompletionInput{CompletedOn: date, Note: r.FormValue("note")}, nil
}

func (h *Handler) parseEventInput(r *http.Request) (EventInput, error) {
	completion, err := h.parseCompletionInput(r)
	return EventInput{CompletedOn: completion.CompletedOn, Note: completion.Note}, err
}

func (h *Handler) parseCategoryInput(r *http.Request) (CategoryInput, error) {
	if err := r.ParseForm(); err != nil {
		return CategoryInput{}, fmt.Errorf("%w: invalid form", ErrInvalidInput)
	}
	position, err := strconv.Atoi(strings.TrimSpace(r.FormValue("position")))
	if err != nil {
		return CategoryInput{}, fmt.Errorf("%w: position must be a whole number", ErrInvalidInput)
	}
	return CategoryInput{Name: r.FormValue("name"), Position: position, Active: r.FormValue("active") == "on"}, nil
}

func parseInt16Values(values []string) ([]int16, error) {
	parsed := make([]int16, 0, len(values))
	for _, raw := range values {
		value, err := strconv.ParseInt(raw, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid selection", ErrInvalidInput)
		}
		parsed = append(parsed, int16(value))
	}
	return parsed, nil
}

func parseDate(raw string) (time.Time, error) {
	return time.Parse(time.DateOnly, strings.TrimSpace(raw))
}

func (h *Handler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		h.fail(w, "render routine page", err)
	}
}

func (h *Handler) writeApplicationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		http.Error(w, strings.TrimPrefix(err.Error(), ErrInvalidInput.Error()+": "), http.StatusBadRequest)
	case errors.Is(err, ErrNotFound):
		http.Error(w, "Not found", http.StatusNotFound)
	case errors.Is(err, ErrDeleteNotAllowed):
		http.Error(w, "This item has history or is still in use. Deactivate it instead.", http.StatusConflict)
	default:
		h.fail(w, "routine action", err)
	}
}

func (h *Handler) fail(w http.ResponseWriter, operation string, err error) {
	h.logger.Error("routine request failed", "operation", operation, "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
