package server

import (
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/ekzyis/lntorch/db"
	"github.com/ekzyis/lntorch/lightning"
	"github.com/skip2/go-qrcode"
)

//go:embed index.html
var indexHTML string

var indexTmpl = template.Must(template.New("index").Parse(indexHTML))

// ticketMsats is the cost of a ticket to enter — sats_per_round (100 sats).
const ticketMsats = 100 * 1000

// commit is the short VCS revision stamped into the binary by `go build`.
var commit = readCommit()

func readCommit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			if len(s.Value) >= 7 {
				return s.Value[:7]
			}
			return s.Value
		}
	}
	return "unknown"
}

type Server struct {
	db  *db.DB
	ln  lightning.Lightning
	mux *http.ServeMux
}

func New(database *db.DB, ln lightning.Lightning) *Server {
	s := &Server{
		db:  database,
		ln:  ln,
		mux: http.NewServeMux(),
	}
	s.mux.HandleFunc("/", s.indexHandler)
	s.mux.HandleFunc("/join", s.joinHandler)
	s.mux.HandleFunc("/leave", s.leaveHandler)
	s.mux.HandleFunc("/invoice/{hash}", s.invoiceHandler)
	s.mux.HandleFunc("/invoice/status", s.invoiceStatusHandler)
	s.mux.HandleFunc("/pay", s.payHandler)
	return s
}

type HTMLContext struct {
	PlayerID    int64
	PlayerCount int
	Invoice     *InvoiceView
	Commit      string
}

// InvoiceView is the invoice as the template needs it.
type InvoiceView struct {
	Bolt11 string
	QR     template.URL
	Sats   int64
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) generateSession() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) sessionToken(r *http.Request) string {
	cookie, err := r.Cookie("session")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (s *Server) getPlayer(r *http.Request) *db.Player {
	session := s.sessionToken(r)
	if session == "" {
		return nil
	}
	player, err := s.db.GetPlayerBySession(session)
	if err != nil {
		return nil
	}
	return player
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	ctx := HTMLContext{Commit: commit}

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

// joinHandler mints a ticket invoice and redirects to its invoice screen at
// /invoice/{hash}. The player is only created once the invoice is paid (see
// invoiceStatusHandler).
func (s *Server) joinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	session := s.sessionToken(r)
	if session == "" {
		session = s.generateSession()
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    session,
			Path:     "/",
			HttpOnly: true,
		})
	}

	inv, err := s.ln.CreateInvoice(ticketMsats, "lntorch ticket")
	if err != nil {
		http.Error(w, "Failed to create invoice", http.StatusInternalServerError)
		return
	}

	if err := s.db.CreateInvoice(session, inv.Hash()); err != nil {
		http.Error(w, "Failed to record invoice", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/invoice/"+inv.Hash(), http.StatusSeeOther)
}

// invoiceHandler renders the invoice named by its payment hash in the URL
// (/invoice/{hash}), so the screen is refreshable and shareable as a real URL.
// The session cookie is authentication only: a client may view an invoice only
// if the database says that invoice belongs to its session; otherwise it falls
// back to the menu. The screen is served with 402 Payment Required.
func (s *Server) invoiceHandler(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	session := s.sessionToken(r)

	owned := false
	if session != "" {
		owned, _ = s.db.InvoiceBelongsToSession(session, hash)
	}

	if owned {
		if inv, err := s.ln.GetInvoice(hash); err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusPaymentRequired)
			indexTmpl.Execute(w, HTMLContext{Commit: commit, Invoice: invoiceView(inv)})
			return
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// invoiceStatusHandler reports whether this session's latest invoice has been
// paid. On payment it creates the player and joins them to the waiting game.
func (s *Server) invoiceStatusHandler(w http.ResponseWriter, r *http.Request) {
	session := s.sessionToken(r)

	hash, _ := s.db.LatestInvoiceHash(session)

	paid := false
	if hash != "" {
		if inv, err := s.ln.GetInvoice(hash); err == nil && inv.Paid {
			if err := s.admit(session); err == nil {
				paid = true
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"paid": paid})
}

// payHandler settles every outstanding mock invoice — the server side of the
// `lntorch pay` dev command, for exercising the payment flow when autopay is
// off. It only works with the mock backend; with a real node it 404s.
func (s *Server) payHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	mock, ok := s.ln.(*lightning.Mock)
	if !ok {
		http.Error(w, "pay is only available with the mock backend", http.StatusNotFound)
		return
	}

	n := mock.PayAll()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"paid": n})
}

// admit creates the player for this session (if needed) and joins the waiting
// game (if not already in one).
func (s *Server) admit(session string) error {
	player, err := s.db.GetPlayerBySession(session)
	if err != nil {
		player, err = s.db.CreatePlayer(session)
		if err != nil {
			return err
		}
	}
	if game, _ := s.db.GetPlayerGame(player.ID); game != nil {
		return nil
	}
	game, err := s.db.GetOrCreateWaitingGame()
	if err != nil {
		return err
	}
	return s.db.JoinGame(player.ID, game.ID)
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

func invoiceView(inv *lightning.Invoice) *InvoiceView {
	var qr template.URL
	if png, err := qrcode.Encode(strings.ToUpper(inv.Bolt11), qrcode.Medium, 320); err == nil {
		qr = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(png))
	}
	return &InvoiceView{
		Bolt11: inv.Bolt11,
		QR:     qr,
		Sats:   int64(inv.Msats) / 1000,
	}
}
