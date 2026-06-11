package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ekzyis/lntorch/db"
	"github.com/ekzyis/lntorch/lightning"
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
	return New(database, lightning.NewMock(false)), cleanup
}

// join posts to /join, then forces the mock invoice paid and lets the status
// handler admit the player. Returns the session cookie of a player now in the
// waiting room.
func join(t *testing.T, s *Server) *http.Cookie {
	t.Helper()
	jw := httptest.NewRecorder()
	s.ServeHTTP(jw, httptest.NewRequest("POST", "/join", nil))

	var cookie *http.Cookie
	for _, c := range jw.Result().Cookies() {
		if c.Name == "session" {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatal("expected session cookie from /join")
	}

	hash := strings.TrimPrefix(jw.Header().Get("Location"), "/invoice/")
	s.ln.(*lightning.Mock).Pay(hash)

	sw := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/invoice/status", nil)
	req.AddCookie(cookie)
	s.ServeHTTP(sw, req)
	if !strings.Contains(sw.Body.String(), `"paid":true`) {
		t.Fatalf("expected paid:true, got %s", sw.Body.String())
	}
	return cookie
}

func get(t *testing.T, s *Server, cookie *http.Cookie) string {
	t.Helper()
	req := httptest.NewRequest("GET", "/", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	return w.Body.String()
}

func TestIndexWithoutCookie(t *testing.T) {
	s, cleanup := setupTestServer(t)
	defer cleanup()

	body := get(t, s, nil)
	if !hasTestID(body, "join-btn") {
		t.Error("expected join-btn")
	}
	if hasTestID(body, "leave-btn") {
		t.Error("should not show leave-btn without cookie")
	}
}

func TestJoinRedirectsToInvoice(t *testing.T) {
	s, cleanup := setupTestServer(t)
	defer cleanup()

	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("POST", "/join", nil))

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected redirect (303), got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, "/invoice/") {
		t.Errorf("expected redirect to /invoice/<hash>, got %q", loc)
	}
	var cookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			cookie = c
		}
	}
	if cookie == nil || cookie.Value == "" {
		t.Fatal("expected a session cookie to be set")
	}

	// with the owning cookie, the hash URL renders the invoice
	iw := httptest.NewRecorder()
	ireq := httptest.NewRequest("GET", loc, nil)
	ireq.AddCookie(cookie)
	s.ServeHTTP(iw, ireq)
	if iw.Code != http.StatusPaymentRequired {
		t.Errorf("expected 402 Payment Required from %s, got %d", loc, iw.Code)
	}
	if !hasTestID(iw.Body.String(), "invoice") {
		t.Errorf("expected invoice screen at %s", loc)
	}

	// without the cookie, the same URL must not reveal the invoice
	nw := httptest.NewRecorder()
	s.ServeHTTP(nw, httptest.NewRequest("GET", loc, nil))
	if nw.Code != http.StatusSeeOther {
		t.Errorf("expected redirect without the auth cookie, got %d", nw.Code)
	}
}

func TestInvoiceForeignHashRedirects(t *testing.T) {
	s, cleanup := setupTestServer(t)
	defer cleanup()

	// a session with its own pending invoice
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("POST", "/join", nil))
	var cookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			cookie = c
		}
	}

	// a hash this session doesn't own must not render, even with a valid cookie
	req := httptest.NewRequest("GET", "/invoice/deadbeef", nil)
	req.AddCookie(cookie)
	rw := httptest.NewRecorder()
	s.ServeHTTP(rw, req)
	if rw.Code != http.StatusSeeOther {
		t.Errorf("expected redirect for a hash the session doesn't own, got %d", rw.Code)
	}
}

func TestUnpaidInvoiceDoesNotEnter(t *testing.T) {
	s, cleanup := setupTestServer(t)
	defer cleanup()

	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("POST", "/join", nil))
	var cookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			cookie = c
		}
	}

	// status without paying should report not paid and not admit
	sw := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/invoice/status", nil)
	req.AddCookie(cookie)
	s.ServeHTTP(sw, req)
	if !strings.Contains(sw.Body.String(), `"paid":false`) {
		t.Errorf("expected paid:false, got %s", sw.Body.String())
	}
	if hasTestID(get(t, s, cookie), "waiting-msg") {
		t.Error("should not be in the waiting room before paying")
	}
}

func TestPaidJoinEntersRoom(t *testing.T) {
	s, cleanup := setupTestServer(t)
	defer cleanup()

	cookie := join(t, s)
	body := get(t, s, cookie)
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

func TestLeaveClearsCookie(t *testing.T) {
	s, cleanup := setupTestServer(t)
	defer cleanup()

	cookie := join(t, s)

	req := httptest.NewRequest("POST", "/leave", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected redirect (303), got %d", w.Code)
	}
	var clearedCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			clearedCookie = c
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

	c1 := join(t, s)
	if !strings.Contains(get(t, s, c1), "Heroes: 1") {
		t.Error("expected player count 1")
	}

	c2 := join(t, s)
	if !strings.Contains(get(t, s, c2), "Heroes: 2") {
		t.Error("expected player count 2")
	}
}

func TestFullFlow(t *testing.T) {
	s, cleanup := setupTestServer(t)
	defer cleanup()

	// Start without cookie -> join screen
	if !hasTestID(get(t, s, nil), "join-btn") {
		t.Error("should show join-btn initially")
	}

	// Enter -> redirect to the invoice screen at /invoice/<hash>
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("POST", "/join", nil))
	loc := w.Header().Get("Location")
	if w.Code != http.StatusSeeOther || !strings.HasPrefix(loc, "/invoice/") {
		t.Errorf("should redirect to /invoice/<hash> after entering, got %d %q", w.Code, loc)
	}
	var cookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			cookie = c
		}
	}
	iw := httptest.NewRecorder()
	ireq := httptest.NewRequest("GET", loc, nil)
	ireq.AddCookie(cookie)
	s.ServeHTTP(iw, ireq)
	if !hasTestID(iw.Body.String(), "invoice") {
		t.Error("should show invoice at its URL after entering")
	}

	// Pay -> status admits -> waiting room
	hash := strings.TrimPrefix(loc, "/invoice/")
	s.ln.(*lightning.Mock).Pay(hash)
	sw := httptest.NewRecorder()
	statusReq := httptest.NewRequest("GET", "/invoice/status", nil)
	statusReq.AddCookie(cookie)
	s.ServeHTTP(sw, statusReq)

	if !hasTestID(get(t, s, cookie), "waiting-msg") {
		t.Error("should be in waiting room after paying")
	}

	// Leave -> back to join screen
	leaveReq := httptest.NewRequest("POST", "/leave", nil)
	leaveReq.AddCookie(cookie)
	s.ServeHTTP(httptest.NewRecorder(), leaveReq)
	if !hasTestID(get(t, s, nil), "join-btn") {
		t.Error("should show join-btn after leaving")
	}
}
