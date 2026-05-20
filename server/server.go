package server

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"html/template"
	"net/http"
)

//go:embed index.html
var indexHTML string

var indexTmpl = template.Must(template.New("index").Parse(indexHTML))

type Server struct {
	mux *http.ServeMux
}

type HTMLContext struct {
	PlayerID string
}

func New() *Server {
	s := &Server{
		mux: http.NewServeMux(),
	}
	s.mux.HandleFunc("/", s.indexHandler)
	s.mux.HandleFunc("/join", s.joinHandler)
	s.mux.HandleFunc("/leave", s.leaveHandler)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) generatePlayerID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	var ctx HTMLContext
	if cookie, err := r.Cookie("player_id"); err == nil {
		ctx.PlayerID = cookie.Value
	}
	indexTmpl.Execute(w, ctx)
}

func (s *Server) joinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	playerID := s.generatePlayerID()
	http.SetCookie(w, &http.Cookie{
		Name:     "player_id",
		Value:    playerID,
		Path:     "/",
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) leaveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "player_id",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
