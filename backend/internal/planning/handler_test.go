package planning

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
	homeView View
	planView View
	viewKind string
	viewDate string
	saved    SavePlanInput
}

func (f *fakeApplication) Home(context.Context) (View, error) {
	return f.homeView, nil
}

func (f *fakeApplication) ViewPlan(
	_ context.Context,
	kind string,
	periodStart string,
) (View, error) {
	f.viewKind = kind
	f.viewDate = periodStart
	return f.planView, nil
}

func (f *fakeApplication) SavePlan(_ context.Context, input SavePlanInput) error {
	f.saved = input
	return nil
}

func TestHomeRendersCurrentDailyPlan(t *testing.T) {
	app := &fakeApplication{
		homeView: View{
			Kind:        KindDaily,
			PeriodStart: time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
			Timezone:    "UTC",
			Plan: &Plan{
				Revision:        2,
				SummaryMarkdown: "Build a useful planning loop.",
				ContentMarkdown: "<script>alert(1)</script>",
			},
		},
	}
	handler := testHandler(t, app)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, want := range []string{
		"Daily plan",
		"Build a useful planning loop.",
		`href="/routines"`,
		"&lt;script&gt;alert(1)&lt;/script&gt;",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("response does not contain %q", want)
		}
	}
	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Fatal("response contains unescaped plan HTML")
	}
}

func TestPlanUsesKindAndPeriodStartFromPath(t *testing.T) {
	app := &fakeApplication{
		planView: View{
			Kind:        KindWeekly,
			PeriodStart: time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
			Timezone:    "UTC",
		},
	}
	handler := testHandler(t, app)
	request := httptest.NewRequest(
		http.MethodGet,
		"/plans/weekly/2026-07-27",
		nil,
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if app.viewKind != "weekly" || app.viewDate != "2026-07-27" {
		t.Fatalf("view path = %q %q", app.viewKind, app.viewDate)
	}
}

func TestSavePlanUsesPathAndFormValues(t *testing.T) {
	app := &fakeApplication{}
	handler := testHandler(t, app)
	values := url.Values{
		"summary_markdown": {"Focus on the schema."},
		"content_markdown": {"# Plan\n\nReview the migration."},
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/plans/weekly/2026-07-27/revisions",
		strings.NewReader(values.Encode()),
	)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", response.Code)
	}
	if app.saved.Kind != "weekly" ||
		app.saved.PeriodStart != "2026-07-27" ||
		app.saved.SummaryMarkdown != "Focus on the schema." ||
		app.saved.ContentMarkdown != "# Plan\n\nReview the migration." {
		t.Fatalf("saved plan = %#v", app.saved)
	}
	if location := response.Header().Get("Location"); location != "/plans/weekly/2026-07-27#plan" {
		t.Fatalf("location = %q", location)
	}
}

func testHandler(t *testing.T, app Application) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	feature, err := NewHandler(app, logger)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	feature.Register(mux)
	return mux
}
