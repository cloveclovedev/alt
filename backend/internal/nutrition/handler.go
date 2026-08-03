package nutrition

import (
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

const maxPhotoBytes = 10 << 20 // 10 MiB

// Handler serves the nutrition logging input flow: a manual form, catalog
// quick-pick, and AI photo/text parsing behind a user confirmation gate. It works
// as plain forms and, for the LLM-backed parse actions, as HTMX fragments.
type Handler struct {
	service *Service
	logger  *slog.Logger
	tmpl    *template.Template
}

// NewHandler parses the embedded templates and constructs the handler.
func NewHandler(service *Service, logger *slog.Logger) (*Handler, error) {
	tmpl, err := template.New("nutrition").Funcs(template.FuncMap{
		"title": titleCase,
	}).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse nutrition templates: %w", err)
	}
	return &Handler{service: service, logger: logger, tmpl: tmpl}, nil
}

// Register wires the entry-creation routes.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /nutrition/entries/new", h.newForm)
	mux.HandleFunc("POST /nutrition/entries/manual", h.createManual)
	mux.HandleFunc("POST /nutrition/entries/catalog", h.createFromCatalog)
	mux.HandleFunc("POST /nutrition/entries/parse-text", h.parseText)
	mux.HandleFunc("POST /nutrition/entries/parse-photo", h.parsePhoto)
	mux.HandleFunc("POST /nutrition/entries/confirm", h.confirm)
}

// newFormView is the add-entry page model.
type newFormView struct {
	Date         string
	MealTypes    []MealType
	Catalog      []CatalogItem
	Candidates   []Candidate
	ParseSource  string
	AIEnabled    bool
	PhotoEnabled bool
	Message      string
}

func (h *Handler) newForm(w http.ResponseWriter, r *http.Request) {
	h.renderForm(w, r, "")
}

func (h *Handler) renderForm(w http.ResponseWriter, r *http.Request, message string) {
	catalog, err := h.service.ListCatalog(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	view := newFormView{
		Date:         h.service.LocalToday().Format(time.DateOnly),
		MealTypes:    MealTypes(),
		Catalog:      catalog,
		AIEnabled:    h.service.ai != nil,
		PhotoEnabled: h.service.ai != nil && h.service.photos != nil,
		Message:      message,
	}
	h.renderPage(w, "new.html", view)
}

func (h *Handler) createManual(w http.ResponseWriter, r *http.Request) {
	calories, protein, err := parseMetrics(r)
	if err != nil {
		h.badRequest(w, err)
		return
	}
	input := EntryInput{
		LoggedDate:   h.service.LocalToday(),
		MealType:     MealType(strings.TrimSpace(r.FormValue("meal_type"))),
		Name:         r.FormValue("name"),
		CaloriesKcal: calories,
		ProteinG:     protein,
		Source:       SourceManual,
	}
	if _, err := h.service.CreateEntry(r.Context(), input); err != nil {
		h.badRequest(w, err)
		return
	}
	http.Redirect(w, r, "/nutrition/entries/new", http.StatusSeeOther)
}

func (h *Handler) createFromCatalog(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetCatalogItem(r.Context(), r.FormValue("catalog_id"))
	if err != nil {
		h.badRequest(w, err)
		return
	}
	id := item.ID
	input := EntryInput{
		LoggedDate:   h.service.LocalToday(),
		MealType:     MealType(strings.TrimSpace(r.FormValue("meal_type"))),
		Name:         item.Name,
		CaloriesKcal: item.CaloriesKcal,
		ProteinG:     item.ProteinG,
		Source:       SourceCatalog,
		CatalogID:    &id,
	}
	if _, err := h.service.CreateEntry(r.Context(), input); err != nil {
		h.badRequest(w, err)
		return
	}
	http.Redirect(w, r, "/nutrition/entries/new", http.StatusSeeOther)
}

func (h *Handler) parseText(w http.ResponseWriter, r *http.Request) {
	candidates, err := h.service.ParseText(r.Context(), r.FormValue("text"))
	h.renderCandidates(w, r, candidates, err)
}

func (h *Handler) parsePhoto(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxPhotoBytes); err != nil {
		h.renderCandidates(w, r, nil, fmt.Errorf("%w: could not read the uploaded photo", ErrInvalidInput))
		return
	}
	file, header, err := r.FormFile("photo")
	if err != nil {
		h.renderCandidates(w, r, nil, fmt.Errorf("%w: attach a photo", ErrInvalidInput))
		return
	}
	defer file.Close()
	data := make([]byte, 0, header.Size)
	buf := make([]byte, 32<<10)
	for {
		n, readErr := file.Read(buf)
		data = append(data, buf[:n]...)
		if len(data) > maxPhotoBytes {
			h.renderCandidates(w, r, nil, fmt.Errorf("%w: photo is too large", ErrInvalidInput))
			return
		}
		if readErr != nil {
			break
		}
	}
	contentType := header.Header.Get("Content-Type")
	candidates, err := h.service.ParsePhoto(r.Context(), data, contentType)
	h.renderCandidates(w, r, candidates, err)
}

func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.badRequest(w, fmt.Errorf("%w: could not read the confirmation", ErrInvalidInput))
		return
	}
	// Each candidate row is submitted with an index suffix so unchecked rows (whose
	// checkbox is simply absent) do not misalign the remaining fields.
	created := 0
	for _, index := range r.Form["row"] {
		if r.FormValue("include_"+index) != "on" {
			continue
		}
		calories, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("calories_kcal_" + index)))
		protein, _ := strconv.ParseFloat(strings.TrimSpace(r.FormValue("protein_g_"+index)), 64)
		input := EntryInput{
			LoggedDate:   h.service.LocalToday(),
			MealType:     MealType(strings.TrimSpace(r.FormValue("meal_type_" + index))),
			Name:         r.FormValue("name_" + index),
			CaloriesKcal: calories,
			ProteinG:     protein,
			Source:       EntrySource(strings.TrimSpace(r.FormValue("source_" + index))),
			CatalogID:    optionalID(r.FormValue("catalog_id_" + index)),
		}
		if _, err := h.service.CreateEntry(r.Context(), input); err != nil {
			h.badRequest(w, err)
			return
		}
		created++
	}
	h.renderForm(w, r, fmt.Sprintf("Recorded %d %s.", created, plural(created, "entry", "entries")))
}

func (h *Handler) renderCandidates(w http.ResponseWriter, r *http.Request, candidates []Candidate, err error) {
	if err != nil {
		if errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrAIUnavailable) || errors.Is(err, ErrPhotoStorageUnavailable) {
			message := strings.TrimPrefix(err.Error(), ErrInvalidInput.Error()+": ")
			if r.Header.Get("HX-Request") == "true" {
				h.renderFragment(w, "candidates.html", candidatesView{Message: message})
				return
			}
			h.renderForm(w, r, message)
			return
		}
		h.fail(w, err)
		return
	}
	view := candidatesView{Candidates: candidates, MealTypes: MealTypes()}
	if len(candidates) == 0 {
		view.Message = "No entries were found. Try a clearer photo or a more specific description."
	}
	if r.Header.Get("HX-Request") == "true" {
		h.renderFragment(w, "candidates.html", view)
		return
	}
	h.renderForm2(w, r, candidates)
}

// renderForm2 re-renders the full page with the parsed candidates for the
// no-JavaScript fallback path.
func (h *Handler) renderForm2(w http.ResponseWriter, r *http.Request, candidates []Candidate) {
	catalog, err := h.service.ListCatalog(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	h.renderPage(w, "new.html", newFormView{
		Date:         h.service.LocalToday().Format(time.DateOnly),
		MealTypes:    MealTypes(),
		Catalog:      catalog,
		Candidates:   candidates,
		AIEnabled:    h.service.ai != nil,
		PhotoEnabled: h.service.ai != nil && h.service.photos != nil,
	})
}

type candidatesView struct {
	Candidates []Candidate
	MealTypes  []MealType
	Message    string
}

func (h *Handler) renderPage(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		h.fail(w, err)
	}
}

func (h *Handler) renderFragment(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		h.fail(w, err)
	}
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	h.logger.Error("nutrition request failed", "error", err)
	http.Error(w, "The request could not be completed.", http.StatusBadGateway)
}

func (h *Handler) badRequest(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrConflict) || errors.Is(err, ErrNotFound) {
		http.Error(w, strings.TrimPrefix(err.Error(), ErrInvalidInput.Error()+": "), http.StatusBadRequest)
		return
	}
	h.fail(w, err)
}

func parseMetrics(r *http.Request) (int, float64, error) {
	calories, err := strconv.Atoi(strings.TrimSpace(r.FormValue("calories_kcal")))
	if err != nil {
		return 0, 0, fmt.Errorf("%w: calories must be a whole number", ErrInvalidInput)
	}
	protein, err := strconv.ParseFloat(strings.TrimSpace(r.FormValue("protein_g")), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: protein must be a number", ErrInvalidInput)
	}
	return calories, protein, nil
}

func titleCase(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func optionalID(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
