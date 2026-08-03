package ai

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
)

// settingsService is the behavior the settings page needs from the AI service.
type settingsService interface {
	SettingsView(context.Context) ([]PurposeSetting, error)
	Assign(context.Context, string, string) error
}

// SettingsHandler serves /settings/ai: per-purpose model selection filtered to
// the models each purpose's capabilities allow.
type SettingsHandler struct {
	service settingsService
	logger  *slog.Logger
	tmpl    *template.Template
}

// NewSettingsHandler parses the settings template and constructs the handler.
func NewSettingsHandler(service settingsService, logger *slog.Logger) (*SettingsHandler, error) {
	tmpl, err := template.New("ai-settings").Parse(aiSettingsTemplate)
	if err != nil {
		return nil, err
	}
	return &SettingsHandler{service: service, logger: logger, tmpl: tmpl}, nil
}

// Register wires the settings routes.
func (h *SettingsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /settings/ai", h.page)
	mux.HandleFunc("POST /settings/ai", h.save)
}

func (h *SettingsHandler) page(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.SettingsView(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.Execute(w, settings); err != nil {
		h.fail(w, err)
	}
}

func (h *SettingsHandler) save(w http.ResponseWriter, r *http.Request) {
	purpose := strings.TrimSpace(r.FormValue("purpose"))
	modelID := strings.TrimSpace(r.FormValue("model_id"))
	if err := h.service.Assign(r.Context(), purpose, modelID); err != nil {
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/settings/ai", http.StatusSeeOther)
}

func (h *SettingsHandler) fail(w http.ResponseWriter, err error) {
	h.logger.Error("AI settings request failed", "error", err)
	http.Error(w, "Settings request failed", http.StatusBadGateway)
}

const aiSettingsTemplate = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>AI settings · alt</title><link rel="stylesheet" href="/static/app.css"></head><body><header class="site-header"><a class="brand" href="/" aria-label="alt home">alt</a><p>Settings</p><nav class="header-nav"><a href="/settings/calendar">Calendar</a><a href="/settings/github">GitHub</a><a href="/settings/ai">AI</a></nav></header><main><h1>AI settings</h1><p>Choose a Zero Data Retention model for each inference purpose. Only compatible models are listed.</p>{{range .}}{{$selected := .SelectedModelID}}<section><h2>{{.Purpose.Label}}</h2>{{if .SelectedModelID}}<p>Current model: <code>{{.SelectedModelID}}</code></p>{{else}}<p class="empty">No model selected.</p>{{end}}{{if .Models}}<form action="/settings/ai" method="post"><input type="hidden" name="purpose" value="{{.Purpose.Key}}"><label>Model <select name="model_id">{{range .Models}}<option value="{{.ID}}" {{if eq .ID $selected}}selected{{end}}>{{.Name}} ({{.ID}})</option>{{end}}</select></label><button>Save model</button></form>{{else}}<p class="empty">No compatible ZDR model is available for this purpose.</p>{{end}}</section>{{end}}</main></body></html>`
