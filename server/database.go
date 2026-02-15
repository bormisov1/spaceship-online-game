package main

import (
	"database/sql"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
}

// PlayerRow represents a player record in the database
type PlayerRow struct {
	ID        int64
	Username  string
	Email     string
	PassHash  string
	CreatedAt time.Time
}

// StatsRow represents player stats
type StatsRow struct {
	PlayerID int64
	Kills    int
	Deaths   int
	Wins     int
	Losses   int
	Playtime float64 // seconds
	XP       int
	Level    int
}

// MatchRow represents a completed match
type MatchRow struct {
	ID         int64
	Mode       int
	Duration   float64
	WinnerTeam int
	CreatedAt  time.Time
}

// MatchPlayerRow represents a player's participation in a match
type MatchPlayerRow struct {
	MatchID  int64
	PlayerID int64
	Team     int
	Kills    int
	Deaths   int
	Assists  int
	Score    int
	XPEarned int
}

// OpenDB opens (or creates) the SQLite database
func OpenDB(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrency
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		conn.Close()
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, err
	}
	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// migrate creates tables if they don't exist
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS players (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL DEFAULT '',
		pass_hash TEXT NOT NULL DEFAULT '',
		is_guest INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS stats (
		player_id INTEGER PRIMARY KEY REFERENCES players(id),
		kills INTEGER NOT NULL DEFAULT 0,
		deaths INTEGER NOT NULL DEFAULT 0,
		wins INTEGER NOT NULL DEFAULT 0,
		losses INTEGER NOT NULL DEFAULT 0,
		playtime REAL NOT NULL DEFAULT 0,
		xp INTEGER NOT NULL DEFAULT 0,
		level INTEGER NOT NULL DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS matches (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		mode INTEGER NOT NULL DEFAULT 0,
		duration REAL NOT NULL DEFAULT 0,
		winner_team INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS match_players (
		match_id INTEGER NOT NULL REFERENCES matches(id),
		player_id INTEGER NOT NULL REFERENCES players(id),
		team INTEGER NOT NULL DEFAULT 0,
		kills INTEGER NOT NULL DEFAULT 0,
		deaths INTEGER NOT NULL DEFAULT 0,
		assists INTEGER NOT NULL DEFAULT 0,
		score INTEGER NOT NULL DEFAULT 0,
		xp_earned INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (match_id, player_id)
	);

	CREATE INDEX IF NOT EXISTS idx_match_players_player ON match_players(player_id);
	CREATE INDEX IF NOT EXISTS idx_players_username ON players(username);
	`
	_, err := db.conn.Exec(schema)
	if err != nil {
		log.Printf("DB migration error: %v", err)
	}
	return err
}

// CreatePlayer creates a new player account (returns player ID)
func (db *DB) CreatePlayer(username, email, passHash string) (int64, error) {
	res, err := db.conn.Exec(
		"INSERT INTO players (username, email, pass_hash) VALUES (?, ?, ?)",
		username, email, passHash,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	// Create stats row
	_, err = db.conn.Exec("INSERT INTO stats (player_id) VALUES (?)", id)
	return id, err
}

// CreateGuest creates a guest player (no password, no email)
func (db *DB) CreateGuest(username string) (int64, error) {
	res, err := db.conn.Exec(
		"INSERT INTO players (username, is_guest) VALUES (?, 1)",
		username,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	_, err = db.conn.Exec("INSERT INTO stats (player_id) VALUES (?)", id)
	return id, err
}

// GetPlayerByUsername returns a player by username
func (db *DB) GetPlayerByUsername(username string) (*PlayerRow, error) {
	row := db.conn.QueryRow(
		"SELECT id, username, email, pass_hash, created_at FROM players WHERE username = ?",
		username,
	)
	p := &PlayerRow{}
	err := row.Scan(&p.ID, &p.Username, &p.Email, &p.PassHash, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// GetPlayerByID returns a player by ID
func (db *DB) GetPlayerByID(id int64) (*PlayerRow, error) {
	row := db.conn.QueryRow(
		"SELECT id, username, email, pass_hash, created_at FROM players WHERE id = ?",
		id,
	)
	p := &PlayerRow{}
	err := row.Scan(&p.ID, &p.Username, &p.Email, &p.PassHash, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// GetStats returns player stats
func (db *DB) GetStats(playerID int64) (*StatsRow, error) {
	row := db.conn.QueryRow(
		"SELECT player_id, kills, deaths, wins, losses, playtime, xp, level FROM stats WHERE player_id = ?",
		playerID,
	)
	s := &StatsRow{}
	err := row.Scan(&s.PlayerID, &s.Kills, &s.Deaths, &s.Wins, &s.Losses, &s.Playtime, &s.XP, &s.Level)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

// UpdateStatsAfterMatch updates player stats after a match ends
func (db *DB) UpdateStatsAfterMatch(playerID int64, kills, deaths, assists int, won bool, duration float64, xpEarned int) error {
	winInc := 0
	lossInc := 0
	if won {
		winInc = 1
	} else {
		lossInc = 1
	}
	_, err := db.conn.Exec(`
		UPDATE stats SET
			kills = kills + ?,
			deaths = deaths + ?,
			wins = wins + ?,
			losses = losses + ?,
			playtime = playtime + ?,
			xp = xp + ?,
			level = CASE WHEN xp + ? >= CAST(100 * pow(level, 1.5) AS INTEGER) THEN level + 1 ELSE level END
		WHERE player_id = ?`,
		kills, deaths, winInc, lossInc, duration, xpEarned, xpEarned, playerID,
	)
	return err
}

// RecordMatch records a completed match and returns its ID
func (db *DB) RecordMatch(mode int, duration float64, winnerTeam int) (int64, error) {
	res, err := db.conn.Exec(
		"INSERT INTO matches (mode, duration, winner_team) VALUES (?, ?, ?)",
		mode, duration, winnerTeam,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// RecordMatchPlayer records a player's stats for a match
func (db *DB) RecordMatchPlayer(matchID, playerID int64, team, kills, deaths, assists, score, xpEarned int) error {
	_, err := db.conn.Exec(
		`INSERT INTO match_players (match_id, player_id, team, kills, deaths, assists, score, xp_earned)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		matchID, playerID, team, kills, deaths, assists, score, xpEarned,
	)
	return err
}

// GetMatchHistory returns recent matches for a player
func (db *DB) GetMatchHistory(playerID int64, limit int) ([]MatchPlayerRow, error) {
	rows, err := db.conn.Query(`
		SELECT mp.match_id, mp.player_id, mp.team, mp.kills, mp.deaths, mp.assists, mp.score, mp.xp_earned
		FROM match_players mp
		JOIN matches m ON m.id = mp.match_id
		WHERE mp.player_id = ?
		ORDER BY m.created_at DESC
		LIMIT ?`,
		playerID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MatchPlayerRow
	for rows.Next() {
		var r MatchPlayerRow
		if err := rows.Scan(&r.MatchID, &r.PlayerID, &r.Team, &r.Kills, &r.Deaths, &r.Assists, &r.Score, &r.XPEarned); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// UsernameExists checks if a username is taken
func (db *DB) UsernameExists(username string) (bool, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM players WHERE username = ?", username).Scan(&count)
	return count > 0, err
}
