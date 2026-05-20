package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func hasTestID(body, id string) bool {
	return strings.Contains(body, `data-testid="`+id+`"`)
}

func TestIndexWithoutCookie(t *testing.T) {
	s := New()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !hasTestID(body, "join-btn") {
		t.Error("expected join-btn")
	}
	if hasTestID(body, "leave-btn") {
		t.Error("should not show leave-btn without cookie")
	}
}

func TestIndexWithCookie(t *testing.T) {
	s := New()
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "player_id", Value: "test123"})
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !hasTestID(body, "waiting-msg") {
		t.Error("expected waiting-msg")
	}
	if !hasTestID(body, "player-id") {
		t.Error("expected player-id")
	}
	if !strings.Contains(body, "test123") {
		t.Error("expected player ID in body")
	}
	if !hasTestID(body, "leave-btn") {
		t.Error("expected leave-btn")
	}
}

func TestJoinSetsCookie(t *testing.T) {
	s := New()
	req := httptest.NewRequest("POST", "/join", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected redirect (303), got %d", w.Code)
	}
	cookies := w.Result().Cookies()
	var playerCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "player_id" {
			playerCookie = c
			break
		}
	}
	if playerCookie == nil {
		t.Fatal("expected player_id cookie to be set")
	}
	if playerCookie.Value == "" {
		t.Error("expected non-empty player_id")
	}
}

func TestLeaveClearsCookie(t *testing.T) {
	s := New()
	req := httptest.NewRequest("POST", "/leave", nil)
	req.AddCookie(&http.Cookie{Name: "player_id", Value: "test123"})
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected redirect (303), got %d", w.Code)
	}
	cookies := w.Result().Cookies()
	var playerCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "player_id" {
			playerCookie = c
			break
		}
	}
	if playerCookie == nil {
		t.Fatal("expected player_id cookie in response")
	}
	if playerCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge -1 to clear cookie, got %d", playerCookie.MaxAge)
	}
}

func TestFullFlow(t *testing.T) {
	s := New()

	// Start without cookie
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if !hasTestID(w.Body.String(), "join-btn") {
		t.Error("should show join-btn initially")
	}

	// Join
	req = httptest.NewRequest("POST", "/join", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	var playerCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "player_id" {
			playerCookie = c
			break
		}
	}

	// Verify in room
	req = httptest.NewRequest("GET", "/", nil)
	req.AddCookie(playerCookie)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if !hasTestID(w.Body.String(), "waiting-msg") {
		t.Error("should show waiting-msg after join")
	}

	// Leave
	req = httptest.NewRequest("POST", "/leave", nil)
	req.AddCookie(playerCookie)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)

	// Verify out (no cookie)
	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if !hasTestID(w.Body.String(), "join-btn") {
		t.Error("should show join-btn after leave")
	}
}
