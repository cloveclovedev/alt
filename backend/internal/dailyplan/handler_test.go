package dailyplan

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
	today       Today
	journalBody string
	task        AddTaskInput
	completedID string
	planSummary string
}

func (f *fakeApplication) Today(context.Context) (Today, error) {
	return f.today, nil
}

func (f *fakeApplication) AddJournalEntry(_ context.Context, body string) error {
	f.journalBody = body
	return nil
}

func (f *fakeApplication) AddTask(_ context.Context, input AddTaskInput) error {
	f.task = input
	return nil
}

func (f *fakeApplication) CompleteTask(_ context.Context, taskID string) error {
	f.completedID = taskID
	return nil
}

func (f *fakeApplication) AcceptPlan(_ context.Context, summary string) error {
	f.planSummary = summary
	return nil
}

func TestTodayRendersDailyPlanningData(t *testing.T) {
	app := &fakeApplication{
		today: Today{
			Date:     time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
			Timezone: "UTC",
			JournalEntries: []JournalEntry{{
				Body:       "<script>alert(1)</script>",
				Kind:       "note",
				OccurredAt: time.Date(2026, 7, 27, 8, 30, 0, 0, time.UTC),
			}},
			Tasks: []Task{{Title: "Ship the first slice", Priority: "P1", Status: "active"}},
			Plan:  &Plan{Revision: 2, Summary: "Build a useful daily loop."},
		},
	}
	handler := testHandler(t, app)

	request := httptest.NewRequest(http.MethodGet, "/today", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, want := range []string{
		"Build a useful daily loop.",
		"Ship the first slice",
		"&lt;script&gt;alert(1)&lt;/script&gt;",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("response does not contain %q", want)
		}
	}
	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Fatal("response contains unescaped journal HTML")
	}
}

func TestAddTaskUsesFormValues(t *testing.T) {
	app := &fakeApplication{}
	handler := testHandler(t, app)
	values := url.Values{
		"title":    {"Write the design"},
		"notes":    {"Keep it small"},
		"priority": {"P1"},
		"due_date": {"2026-07-28"},
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/tasks",
		strings.NewReader(values.Encode()),
	)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", response.Code)
	}
	if app.task.Title != "Write the design" || app.task.Priority != "P1" {
		t.Fatalf("task = %#v", app.task)
	}
}

func TestCompleteTaskUsesPathID(t *testing.T) {
	app := &fakeApplication{}
	handler := testHandler(t, app)
	request := httptest.NewRequest(http.MethodPost, "/tasks/task-123/complete", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", response.Code)
	}
	if app.completedID != "task-123" {
		t.Fatalf("completed ID = %q, want task-123", app.completedID)
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
