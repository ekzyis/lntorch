package server

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"html/template"
	"net/http"

	"github.com/ekzyis/lntorch/db"
)

//go:embed index.html
var indexHTML string

var indexTmpl = template.Must(template.New("index").Parse(indexHTML))

type Server struct {
	db  *db.DB
	mux *http.ServeMux
}

func New(database *db.DB) *Server {
	s := &Server{
		db:  database,
		mux: http.NewServeMux(),
	}
	s.mux.HandleFunc("/", s.indexHandler)
	s.mux.HandleFunc("/join", s.joinHandler)
	s.mux.HandleFunc("/leave", s.leaveHandler)
	return s
}

type HTMLContext struct {
	PlayerID    int64
	PlayerCount int
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) generateSession() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) getPlayer(r *http.Request) *db.Player {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil
	}
	player, err := s.db.GetPlayerBySession(cookie.Value)
	if err != nil {
		return nil
	}
	return player
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	var ctx HTMLContext

	player := s.getPlayer(r)
	if player != nil {
		game, _ := s.db.GetPlayerGame(player.ID)
		if game != nil {
			ctx.PlayerID = player.ID
			count, _ := s.db.CountPlayersInGame(game.ID)
			ctx.PlayerCount = count
		}
	} else {
		game, _ := s.db.GetOrCreateWaitingGame()
		if game != nil {
			count, _ := s.db.CountPlayersInGame(game.ID)
			ctx.PlayerCount = count
		}
	}

	indexTmpl.Execute(w, ctx)
}

func (s *Server) joinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	session := s.generateSession()
	player, err := s.db.CreatePlayer(session)
	if err != nil {
		http.Error(w, "Failed to create player", http.StatusInternalServerError)
		return
	}

	game, err := s.db.GetOrCreateWaitingGame()
	if err != nil {
		http.Error(w, "Failed to get game", http.StatusInternalServerError)
		return
	}

	if err := s.db.JoinGame(player.ID, game.ID); err != nil {
		http.Error(w, "Failed to join game", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session,
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

	player := s.getPlayer(r)
	if player != nil {
		s.db.DeletePlayer(player.ID)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
