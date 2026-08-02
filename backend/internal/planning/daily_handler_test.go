package planning

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeDailyApplication struct {
	view DailyPlanningView
}

func (f *fakeDailyApplication) StartOrResume(context.Context, string) (DailyPlanningView, error) {
	return f.view, nil
}
func (f *fakeDailyApplication) RefreshContext(context.Context, string, string) (DailyPlanningView, error) {
	return f.view, nil
}
func (f *fakeDailyApplication) ContinueWithoutFailedSources(context.Context, string, string) (DailyPlanningView, error) {
	return f.view, nil
}
func (f *fakeDailyApplication) SendMessage(context.Context, string, string, string) (DailyPlanningView, error) {
	return f.view, nil
}
func (f *fakeDailyApplication) RetryLastMessage(context.Context, string, string) (DailyPlanningView, error) {
	return f.view, nil
}
func (f *fakeDailyApplication) Review(context.Context, string, string) (DailyPlanningView, error) {
	return f.view, nil
}
func (f *fakeDailyApplication) BackToChat(context.Context, string, string) (DailyPlanningView, error) {
	return f.view, nil
}
func (f *fakeDailyApplication) Confirm(context.Context, string, string) (DailyPlanningView, error) {
	return f.view, nil
}

func dailyTestHandler(t *testing.T, view DailyPlanningView) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler, err := NewDailyHandler(&fakeDailyApplication{view: view}, logger)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	handler.Register(mux)
	return mux
}

func getDaily(t *testing.T, handler http.Handler) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/planning/daily/2026-07-27", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	return response.Body.String()
}

func TestDailyReviewRendersStructuredSelections(t *testing.T) {
	view := DailyPlanningView{
		Date:     time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
		Timezone: "UTC",
		Session: &DailyPlanningSession{
			ID:     "session-1",
			Status: DailySessionReviewing,
			Preview: &DailyPlanProposal{
				SummaryMarkdown: "Ship the UI.",
				ContentMarkdown: "Focus on **rendering** first.",
			},
		},
		PreviewComponents: &PlanComponents{
			GitHubIssues: []PlanGitHubIssue{{
				RepositoryOwner: "cloveclovedev", RepositoryName: "alt", Number: 61,
				Title: "Restructure daily planning UI", HTMLURL: "https://github.com/cloveclovedev/alt/issues/61",
			}},
			ActionItems:    []ActionItem{{Title: "Draft release notes"}},
			CalendarEvents: []PlanCalendarEvent{{TimeLabel: "10:00", Title: "Standup"}},
		},
	}
	body := getDaily(t, dailyTestHandler(t, view))

	for _, want := range []string{
		"<strong>rendering</strong>",                            // prose rendered as Markdown
		`href="https://github.com/cloveclovedev/alt/issues/61"`, // issue link
		"cloveclovedev/alt#61",
		"Draft release notes",
		"10:00",
		"Standup",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("review body missing %q", want)
		}
	}
}

func TestDailyChatEscapesUserAndRendersAssistant(t *testing.T) {
	view := DailyPlanningView{
		Date:     time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
		Timezone: "UTC",
		Session: &DailyPlanningSession{
			ID:     "session-1",
			Status: DailySessionChatting,
			Messages: []DailyPlanningMessage{
				{Role: MessageRoleUser, Content: "<b>plan</b> today"},
				{Role: MessageRoleAssistant, Content: "Do **this** now."},
			},
		},
	}
	body := getDaily(t, dailyTestHandler(t, view))

	if !strings.Contains(body, "&lt;b&gt;plan&lt;/b&gt; today") {
		t.Fatalf("user message not escaped: %s", body)
	}
	if strings.Contains(body, "<b>plan</b>") {
		t.Fatal("user message rendered as raw HTML")
	}
	if !strings.Contains(body, "<strong>this</strong>") {
		t.Fatalf("assistant Markdown not rendered: %s", body)
	}
}
