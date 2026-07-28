package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSameOriginRejectsCrossOriginPost(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := SameOrigin(next)
	request := httptest.NewRequest(
		http.MethodPost,
		"http://alt.local/plans/daily/2026-07-27/revisions",
		nil,
	)
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

func TestSameOriginAllowsMatchingOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := SameOrigin(next)
	request := httptest.NewRequest(
		http.MethodPost,
		"https://alt.example/plans/daily/2026-07-27/revisions",
		nil,
	)
	request.Header.Set("Origin", "https://alt.example")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
}
