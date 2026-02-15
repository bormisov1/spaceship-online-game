package main

import (
	"database/sql"
	"log"
	"math"
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

// XPForLevel returns the total XP required to reach a given level.
// Level 1 requires 0 XP, level 2 requires 100, etc.
// Formula: sum of 100 * i^1.5 for i in 1..level-1
func XPForLevel(level int) int {
	if level <= 1 {
		return 0
	}
	total := 0.0
	for i := 1; i < level; i++ {
		total += 100.0 * math.Pow(float64(i), 1.5)
	}
	return int(total)
}

// XPToNextLevel returns XP needed from current level to reach the next level
func XPToNextLevel(level int) int {
	return XPForLevel(level+1) - XPForLevel(level)
}

// CalculateLevel returns the level for a given total XP amount
func CalculateLevel(totalXP int) int {
	level := 1
	for {
		needed := XPForLevel(level + 1)
		if totalXP < needed {
			return level
		}
		level++
		if level > 100 { // cap at 100
			return 100
		}
	}
}

// UpdateStatsAfterMatch updates player stats after a match ends.
// Returns (newXP, newLevel) for client notification.
func (db *DB) UpdateStatsAfterMatch(playerID int64, kills, deaths, assists int, won bool, duration float64, xpEarned int) (int, int, error) {
	winInc := 0
	lossInc := 0
	if won {
		winInc = 1
	} else {
		lossInc = 1
	}

	// First update kills/deaths/wins/losses/playtime/xp
	_, err := db.conn.Exec(`
		UPDATE stats SET
			kills = kills + ?,
			deaths = deaths + ?,
			wins = wins + ?,
			losses = losses + ?,
			playtime = playtime + ?,
			xp = xp + ?
		WHERE player_id = ?`,
		kills, deaths, winInc, lossInc, duration, xpEarned, playerID,
	)
	if err != nil {
		return 0, 0, err
	}

	// Read back total XP and calculate proper level
	var totalXP int
	err = db.conn.QueryRow("SELECT xp FROM stats WHERE player_id = ?", playerID).Scan(&totalXP)
	if err != nil {
		return 0, 0, err
	}
	newLevel := CalculateLevel(totalXP)

	// Update level
	_, err = db.conn.Exec("UPDATE stats SET level = ? WHERE player_id = ?", newLevel, playerID)
	return totalXP, newLevel, err
}

// GetLeaderboard returns top players sorted by the given field
func (db *DB) GetLeaderboard(orderBy string, limit int) ([]LeaderboardEntry, error) {
	// Whitelist valid order columns
	validCols := map[string]string{
		"kills": "s.kills", "wins": "s.wins", "level": "s.level",
		"xp": "s.xp", "kd": "CASE WHEN s.deaths > 0 THEN CAST(s.kills AS REAL)/s.deaths ELSE s.kills END",
	}
	col, ok := validCols[orderBy]
	if !ok {
		col = "s.xp"
	}

	query := `SELECT p.username, s.level, s.xp, s.kills, s.deaths, s.wins, s.losses
		FROM stats s JOIN players p ON p.id = s.player_id
		WHERE p.is_guest = 0
		ORDER BY ` + col + ` DESC LIMIT ?`

	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []LeaderboardEntry
	rank := 1
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.Username, &e.Level, &e.XP, &e.Kills, &e.Deaths, &e.Wins, &e.Losses); err != nil {
			return nil, err
		}
		e.Rank = rank
		rank++
		result = append(result, e)
	}
	return result, rows.Err()
}

// LeaderboardEntry represents one row in the leaderboard
type LeaderboardEntry struct {
	Rank     int    `json:"rank"`
	Username string `json:"username"`
	Level    int    `json:"level"`
	XP       int    `json:"xp"`
	Kills    int    `json:"kills"`
	Deaths   int    `json:"deaths"`
	Wins     int    `json:"wins"`
	Losses   int    `json:"losses"`
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
