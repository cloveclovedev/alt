package planning

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
)

type calendarSettingsClient interface {
	AuthorizationURL(context.Context, string) (string, error)
	CompleteAuthorization(context.Context, string, string, string) error
}
type settingsAIClient interface {
	ListCompatibleModels(context.Context) ([]AIModel, error)
}

// SettingsHandler configures the three external inputs without exposing credentials to the browser.
type SettingsHandler struct {
	store    *Store
	userID   string
	github   *GitHubClient
	calendar calendarSettingsClient
	ai       settingsAIClient
	logger   *slog.Logger
	tmpl     *template.Template
}

type aiSettingsView struct {
	Models          []AIModel
	SelectedModelID string
}

func NewSettingsHandler(store *Store, userID string, github *GitHubClient, calendar calendarSettingsClient, ai settingsAIClient, logger *slog.Logger) (*SettingsHandler, error) {
	tmpl, err := template.New("settings").Parse(settingsTemplate)
	if err != nil {
		return nil, err
	}
	return &SettingsHandler{store: store, userID: userID, github: github, calendar: calendar, ai: ai, logger: logger, tmpl: tmpl}, nil
}

func (h *SettingsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /settings/calendar", h.calendarPage)
	mux.HandleFunc("GET /settings/calendar/connect", h.connectCalendar)
	mux.HandleFunc("POST /settings/calendar/connect", h.connectCalendar)
	mux.HandleFunc("POST /settings/calendar/sources", h.saveCalendarSources)
	mux.HandleFunc("GET /settings/calendar/callback", h.calendarCallback)
	mux.HandleFunc("GET /settings/github", h.githubPage)
	mux.HandleFunc("POST /settings/github", h.saveGithub)
	mux.HandleFunc("GET /settings/ai", h.aiPage)
	mux.HandleFunc("POST /settings/ai", h.saveAI)
}

func (h *SettingsHandler) calendarPage(w http.ResponseWriter, r *http.Request) {
	sources, err := h.store.ListCalendarSources(r.Context(), h.userID)
	if err != nil {
		sources = nil
	}
	h.render(w, "Calendar settings", sources, "")
}
func (h *SettingsHandler) connectCalendar(w http.ResponseWriter, r *http.Request) {
	if h.calendar == nil {
		http.Error(w, "Google Calendar OAuth is not configured", http.StatusServiceUnavailable)
		return
	}
	target, err := h.calendar.AuthorizationURL(r.Context(), h.userID)
	if err != nil {
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
func (h *SettingsHandler) saveCalendarSources(w http.ResponseWriter, r *http.Request) {
	sources, err := h.store.ListCalendarSources(r.Context(), h.userID)
	if err == nil {
		_ = r.ParseForm()
		for index := range sources {
			id := sources[index].ID
			sources[index].Enabled = r.FormValue("enabled_"+id) == "on"
			sources[index].Role = CalendarRole(r.FormValue("role_" + id))
			sources[index].PlanningInstructions = strings.TrimSpace(r.FormValue("instructions_" + id))
			if sources[index].Role != CalendarRoleCommitment && sources[index].Role != CalendarRoleOptional && sources[index].Role != CalendarRoleContext {
				err = fmt.Errorf("%w: invalid Calendar source role", ErrInvalidInput)
				break
			}
		}
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := h.store.ReplaceCalendarSources(r.Context(), h.userID, sources); err != nil {
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/settings/calendar", http.StatusSeeOther)
}
func (h *SettingsHandler) calendarCallback(w http.ResponseWriter, r *http.Request) {
	if h.calendar == nil {
		http.Error(w, "Google Calendar OAuth is not configured", http.StatusServiceUnavailable)
		return
	}
	if errorValue := r.URL.Query().Get("error"); errorValue != "" {
		http.Error(w, "Google authorization was not granted", http.StatusBadRequest)
		return
	}
	if err := h.calendar.CompleteAuthorization(r.Context(), h.userID, r.URL.Query().Get("state"), r.URL.Query().Get("code")); err != nil {
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/settings/calendar", http.StatusSeeOther)
}
func (h *SettingsHandler) githubPage(w http.ResponseWriter, r *http.Request) {
	repositories, err := h.store.ListGitHubRepositories(r.Context(), h.userID)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.render(w, "GitHub settings", repositories, "")
}
func (h *SettingsHandler) saveGithub(w http.ResponseWriter, r *http.Request) {
	repository, err := normalizeRepository(GitHubRepository{Owner: formValue(r, "owner"), Name: formValue(r, "name"), Enabled: true, PlanningInstructions: formValue(r, "planning_instructions")})
	if err == nil {
		err = h.github.ValidatePublicRepository(r.Context(), repository.Owner, repository.Name)
	}
	if err == nil {
		err = h.store.SaveGitHubRepository(r.Context(), h.userID, repository)
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/settings/github", http.StatusSeeOther)
}
func (h *SettingsHandler) aiPage(w http.ResponseWriter, r *http.Request) {
	if h.ai == nil {
		http.Error(w, "OpenRouter is not configured", http.StatusServiceUnavailable)
		return
	}
	models, err := h.ai.ListCompatibleModels(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	selected, err := h.store.LoadModelAssignment(r.Context(), h.userID, dailyPlanningPurpose)
	if err != nil && !errors.Is(err, ErrNotFound) {
		h.fail(w, err)
		return
	}
	if errors.Is(err, ErrNotFound) {
		selected = ""
	}
	h.render(w, "AI settings", aiSettingsView{Models: models, SelectedModelID: selected}, "")
}
func (h *SettingsHandler) saveAI(w http.ResponseWriter, r *http.Request) {
	if h.ai == nil {
		http.Error(w, "OpenRouter is not configured", http.StatusServiceUnavailable)
		return
	}
	selected := strings.TrimSpace(formValue(r, "model_id"))
	models, err := h.ai.ListCompatibleModels(r.Context())
	if err == nil {
		valid := false
		for _, model := range models {
			if model.ID == selected {
				valid = true
			}
		}
		if !valid {
			err = fmt.Errorf("%w: choose a ZDR-compatible model", ErrInvalidInput)
		}
	}
	if err == nil {
		err = h.store.SaveModelAssignment(r.Context(), h.userID, dailyPlanningPurpose, selected)
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/settings/ai", http.StatusSeeOther)
}
func (h *SettingsHandler) render(w http.ResponseWriter, title string, data any, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.Execute(w, map[string]any{"Title": title, "Data": data, "Message": message}); err != nil {
		h.fail(w, err)
	}
}
func (h *SettingsHandler) fail(w http.ResponseWriter, err error) {
	h.logger.Error("planning settings request failed", "error", err)
	http.Error(w, "Settings request failed", http.StatusBadGateway)
}

const settingsTemplate = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>{{.Title}} · alt</title><link rel="stylesheet" href="/static/app.css"></head><body><header class="site-header"><a class="brand" href="/" aria-label="alt home">alt</a><p>Settings</p><nav class="header-nav"><a href="/settings/calendar">Calendar</a><a href="/settings/github">GitHub</a><a href="/settings/ai">AI</a></nav></header><main><h1>{{.Title}}</h1>{{if eq .Title "Calendar settings"}}<p><a class="button-link" href="/settings/calendar/connect">Connect Google Calendar</a></p><form action="/settings/calendar/sources" method="post">{{range .Data}}<fieldset><legend>{{.DisplayName}}</legend><label><input type="checkbox" name="enabled_{{.ID}}" {{if .Enabled}}checked{{end}}> Enable for planning</label><label>Role <select name="role_{{.ID}}"><option value="commitment" {{if eq .Role "commitment"}}selected{{end}}>Commitment</option><option value="optional" {{if eq .Role "optional"}}selected{{end}}>Optional</option><option value="context" {{if eq .Role "context"}}selected{{end}}>Context</option></select></label><label>Instructions <textarea name="instructions_{{.ID}}">{{.PlanningInstructions}}</textarea></label></fieldset>{{end}}<button>Save Calendar sources</button></form>{{else if eq .Title "GitHub settings"}}<form action="/settings/github" method="post"><label>Owner <input name="owner" required></label><label>Repository <input name="name" required></label><label>Planning instructions <textarea name="planning_instructions"></textarea></label><button>Add public repository</button></form><ul>{{range .Data}}<li>{{.Owner}}/{{.Name}}</li>{{end}}</ul>{{else}}{{if .Data.SelectedModelID}}<p>Current model: <code>{{.Data.SelectedModelID}}</code></p>{{else}}<p class="empty">No daily planning model has been selected.</p>{{end}}<form action="/settings/ai" method="post"><label>Daily planning model <select name="model_id">{{range .Data.Models}}<option value="{{.ID}}" {{if eq .ID $.Data.SelectedModelID}}selected{{end}}>{{.Name}} ({{.ID}})</option>{{end}}</select></label><button>Save model</button></form>{{end}}</main></body></html>`
