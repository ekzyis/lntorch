package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	db := &DB{sqlDB}
	if err := db.migrate(); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) migrate() error {
	// TODO: migrations?
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS games (
			id INTEGER PRIMARY KEY,
			status TEXT NOT NULL DEFAULT 'waiting',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS players (
			id INTEGER PRIMARY KEY,
			session TEXT UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS games_players (
			game_id INTEGER REFERENCES games(id),
			player_id INTEGER REFERENCES players(id),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (game_id, player_id)
		);

		CREATE TABLE IF NOT EXISTS invoices (
			id INTEGER PRIMARY KEY,
			payment_hash TEXT UNIQUE NOT NULL,
			session TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

type Player struct {
	ID      int64
	Session string
}

type Game struct {
	ID     int64
	Status string
}

func (db *DB) CreatePlayer(session string) (*Player, error) {
	result, err := db.Exec("INSERT INTO players (session) VALUES (?)", session)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Player{ID: id, Session: session}, nil
}

func (db *DB) GetPlayerBySession(session string) (*Player, error) {
	var p Player
	err := db.QueryRow("SELECT id, session FROM players WHERE session = ?", session).
		Scan(&p.ID, &p.Session)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (db *DB) DeletePlayer(id int64) error {
	_, err := db.Exec("DELETE FROM games_players WHERE player_id = ?", id)
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM players WHERE id = ?", id)
	return err
}

func (db *DB) JoinGame(playerID, gameID int64) error {
	_, err := db.Exec("INSERT INTO games_players (game_id, player_id) VALUES (?, ?)", gameID, playerID)
	return err
}

func (db *DB) LeaveGame(playerID, gameID int64) error {
	_, err := db.Exec("DELETE FROM games_players WHERE game_id = ? AND player_id = ?", gameID, playerID)
	return err
}

func (db *DB) GetPlayerGame(playerID int64) (*Game, error) {
	var g Game
	err := db.QueryRow(`
		SELECT g.id, g.status FROM games g
		JOIN games_players gp ON g.id = gp.game_id
		WHERE gp.player_id = ?
		LIMIT 1
	`, playerID).Scan(&g.ID, &g.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (db *DB) GetOrCreateWaitingGame() (*Game, error) {
	var g Game
	err := db.QueryRow("SELECT id, status FROM games WHERE status = 'waiting' LIMIT 1").
		Scan(&g.ID, &g.Status)
	if err == sql.ErrNoRows {
		result, err := db.Exec("INSERT INTO games (status) VALUES ('waiting')")
		if err != nil {
			return nil, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		return &Game{ID: id, Status: "waiting"}, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (db *DB) CreateInvoice(session, paymentHash string) error {
	_, err := db.Exec("INSERT INTO invoices (session, payment_hash) VALUES (?, ?)", session, paymentHash)
	return err
}

// InvoiceBelongsToSession reports whether the invoice with this payment hash
// was issued to this session — the cookie-as-authentication check.
func (db *DB) InvoiceBelongsToSession(session, paymentHash string) (bool, error) {
	var one int
	err := db.QueryRow(
		"SELECT 1 FROM invoices WHERE payment_hash = ? AND session = ?",
		paymentHash, session).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// LatestInvoiceHash returns the payment hash of the session's most recent
// invoice, or "" if it has none.
func (db *DB) LatestInvoiceHash(session string) (string, error) {
	var hash string
	err := db.QueryRow(
		"SELECT payment_hash FROM invoices WHERE session = ? ORDER BY id DESC LIMIT 1",
		session).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return hash, nil
}

func (db *DB) CountPlayersInGame(gameID int64) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM games_players WHERE game_id = ?", gameID).Scan(&count)
	return count, err
}
