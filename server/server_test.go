package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ekzyis/lntorch/db"
)

func hasTestID(body, id string) bool {
	return strings.Contains(body, `data-testid="`+id+`"`)
}

func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()
	dbPath := "test_" + t.Name() + ".db"
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		database.Close()
		os.Remove(dbPath)
	}
	return New(database), cleanup
}

func TestIndexWithoutCookie(t *testing.T) {
	s, cleanup := setupTestServer(t)
	defer cleanup()

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
	s, cleanup := setupTestServer(t)
	defer cleanup()

	// First join to get a valid session
	joinReq := httptest.NewRequest("POST", "/join", nil)
	joinW := httptest.NewRecorder()
	s.ServeHTTP(joinW, joinReq)

	var sessionCookie *http.Cookie
	for _, c := range joinW.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !hasTestID(body, "waiting-msg") {
		t.Error("expected waiting-msg")
	}
	if !hasTestID(body, "player-count") {
		t.Error("expected player-count")
	}
	if !hasTestID(body, "leave-btn") {
		t.Error("expected leave-btn")
	}
}

func TestJoinSetsCookie(t *testing.T) {
	s, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/join", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected redirect (303), got %d", w.Code)
	}
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie to be set")
	}
	if sessionCookie.Value == "" {
		t.Error("expected non-empty session")
	}
}

func TestLeaveClearsCookie(t *testing.T) {
	s, cleanup := setupTestServer(t)
	defer cleanup()

	// First join
	joinReq := httptest.NewRequest("POST", "/join", nil)
	joinW := httptest.NewRecorder()
	s.ServeHTTP(joinW, joinReq)

	var sessionCookie *http.Cookie
	for _, c := range joinW.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	req := httptest.NewRequest("POST", "/leave", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected redirect (303), got %d", w.Code)
	}
	cookies := w.Result().Cookies()
	var clearedCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session" {
			clearedCookie = c
			break
		}
	}
	if clearedCookie == nil {
		t.Fatal("expected session cookie in response")
	}
	if clearedCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge -1 to clear cookie, got %d", clearedCookie.MaxAge)
	}
}

func TestPlayerCount(t *testing.T) {
	s, cleanup := setupTestServer(t)
	defer cleanup()

	// Join with first player
	req1 := httptest.NewRequest("POST", "/join", nil)
	w1 := httptest.NewRecorder()
	s.ServeHTTP(w1, req1)
	var cookie1 *http.Cookie
	for _, c := range w1.Result().Cookies() {
		if c.Name == "session" {
			cookie1 = c
			break
		}
	}

	// Check count is 1
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie1)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "Players: 1") {
		t.Error("expected player count 1")
	}

	// Join with second player
	req2 := httptest.NewRequest("POST", "/join", nil)
	w2 := httptest.NewRecorder()
	s.ServeHTTP(w2, req2)
	var cookie2 *http.Cookie
	for _, c := range w2.Result().Cookies() {
		if c.Name == "session" {
			cookie2 = c
			break
		}
	}

	// Check count is 2
	req = httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie2)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "Players: 2") {
		t.Error("expected player count 2")
	}
}

func TestFullFlow(t *testing.T) {
	s, cleanup := setupTestServer(t)
	defer cleanup()

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
	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	// Verify in room
	req = httptest.NewRequest("GET", "/", nil)
	req.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if !hasTestID(w.Body.String(), "waiting-msg") {
		t.Error("should show waiting-msg after join")
	}

	// Leave
	req = httptest.NewRequest("POST", "/leave", nil)
	req.AddCookie(sessionCookie)
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
