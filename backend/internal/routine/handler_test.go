package routine

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type fakeApplication struct {
	listView  ListView
	newView   NewRoutineView
	detail    DetailView
	created   RoutineInput
	updated   RoutineInput
	completed CompletionInput
	category  CategoryInput
}

func (f *fakeApplication) List(context.Context, bool) (ListView, error)       { return f.listView, nil }
func (f *fakeApplication) NewForm(context.Context) (NewRoutineView, error)    { return f.newView, nil }
func (f *fakeApplication) Detail(context.Context, string) (DetailView, error) { return f.detail, nil }
func (f *fakeApplication) Create(_ context.Context, input RoutineInput) (string, error) {
	f.created = input
	return "routine-1", nil
}
func (f *fakeApplication) Update(_ context.Context, _ string, input RoutineInput) error {
	f.updated = input
	return nil
}
func (f *fakeApplication) Delete(context.Context, string) error { return nil }
func (f *fakeApplication) Complete(_ context.Context, _ string, input CompletionInput) error {
	f.completed = input
	return nil
}
func (f *fakeApplication) UpdateEvent(context.Context, string, string, EventInput) error { return nil }
func (f *fakeApplication) DeleteEvent(context.Context, string, string) error             { return nil }
func (f *fakeApplication) Categories(context.Context) ([]Category, error)                { return nil, nil }
func (f *fakeApplication) CreateCategory(_ context.Context, input CategoryInput) error {
	f.category = input
	return nil
}
func (f *fakeApplication) UpdateCategory(_ context.Context, _ string, input CategoryInput) error {
	f.category = input
	return nil
}
func (f *fakeApplication) DeleteCategory(context.Context, string) error { return nil }

func TestCreateRoutineUsesFormValues(t *testing.T) {
	app := &fakeApplication{}
	handler := testHandler(t, app)
	values := url.Values{
		"category_id": {"category-1"}, "name": {"Practice"}, "notes": {"Keep it short"},
		"interval_days": {"3"}, "available_weekdays": {"7", "1"}, "active_months": {"12", "1"}, "baseline_on": {"2026-07-28"},
	}
	request := httptest.NewRequest(http.MethodPost, "/routines", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/routines/routine-1" {
		t.Fatalf("response = %d %q", response.Code, response.Header().Get("Location"))
	}
	if app.created.CategoryID != "category-1" || app.created.Name != "Practice" || app.created.BaselineOn == nil || app.created.BaselineOn.Format(time.DateOnly) != "2026-07-28" {
		t.Fatalf("created = %#v", app.created)
	}
}

func TestCompleteRoutineUsesCompletionForm(t *testing.T) {
	app := &fakeApplication{}
	handler := testHandler(t, app)
	values := url.Values{"completed_on": {"2026-07-28"}, "note": {"Finished"}}
	request := httptest.NewRequest(http.MethodPost, "/routines/routine-1/complete", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther || app.completed.Note != "Finished" || app.completed.CompletedOn.Format(time.DateOnly) != "2026-07-28" {
		t.Fatalf("response = %d, completed = %#v", response.Code, app.completed)
	}
}

func TestListRendersStatusSections(t *testing.T) {
	app := &fakeApplication{listView: ListView{Today: time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC), DueSections: []DueSection{{State: DueStateOverdue, Categories: []CategorySection{{Category: Category{Name: "Health"}, Routines: []RoutineSummary{{Routine: Routine{ID: "routine-1", Name: "Walk"}}}}}}, {State: DueStateToday}, {State: DueStateUpcoming}}}}
	handler := testHandler(t, app)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/routines", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	for _, want := range []string{"Overdue", "Today", "Upcoming", "Health", "Walk"} {
		if !strings.Contains(response.Body.String(), want) {
			t.Fatalf("body does not contain %q", want)
		}
	}
}

func testHandler(t *testing.T, app Application) http.Handler {
	t.Helper()
	feature, err := NewHandler(app, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	feature.Register(mux)
	return mux
}
