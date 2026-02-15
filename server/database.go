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

	CREATE TABLE IF NOT EXISTS player_achievements (
		player_id INTEGER NOT NULL REFERENCES players(id),
		achievement TEXT NOT NULL,
		unlocked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (player_id, achievement)
	);

	CREATE TABLE IF NOT EXISTS player_skins (
		player_id INTEGER NOT NULL REFERENCES players(id),
		skin_id TEXT NOT NULL,
		purchased_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (player_id, skin_id)
	);

	CREATE TABLE IF NOT EXISTS friends (
		player_id INTEGER NOT NULL REFERENCES players(id),
		friend_id INTEGER NOT NULL REFERENCES players(id),
		status INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (player_id, friend_id)
	);
	CREATE INDEX IF NOT EXISTS idx_friends_friend ON friends(friend_id);

	CREATE TABLE IF NOT EXISTS analytics_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		player_id INTEGER,
		session_id TEXT,
		data TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_analytics_type ON analytics_events(event_type);
	CREATE INDEX IF NOT EXISTS idx_analytics_player ON analytics_events(player_id);
	CREATE INDEX IF NOT EXISTS idx_analytics_created ON analytics_events(created_at);

	CREATE INDEX IF NOT EXISTS idx_match_players_player ON match_players(player_id);
	CREATE INDEX IF NOT EXISTS idx_players_username ON players(username);
	`
	_, err := db.conn.Exec(schema)
	if err != nil {
		log.Printf("DB migration error: %v", err)
		return err
	}

	// Add credits and equipped columns (safe to run multiple times)
	for _, col := range []struct{ table, col, def string }{
		{"stats", "credits", "INTEGER NOT NULL DEFAULT 0"},
		{"stats", "last_login", "DATETIME DEFAULT NULL"},
		{"stats", "login_streak", "INTEGER NOT NULL DEFAULT 0"},
		{"stats", "equipped_skin", "TEXT NOT NULL DEFAULT ''"},
		{"stats", "equipped_trail", "TEXT NOT NULL DEFAULT ''"},
	} {
		_, _ = db.conn.Exec("ALTER TABLE " + col.table + " ADD COLUMN " + col.col + " " + col.def)
	}

	return nil
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

// Friend status constants
const (
	FriendPending  = 0 // request sent, awaiting acceptance
	FriendAccepted = 1 // mutual friends
)

// FriendRow represents a friend relationship
type FriendRow struct {
	PlayerID int64
	FriendID int64
	Username string
	Level    int
	Status   int // 0=pending, 1=accepted
}

// SendFriendRequest creates a pending friend request
func (db *DB) SendFriendRequest(fromID, toID int64) error {
	if fromID == toID {
		return nil
	}
	_, err := db.conn.Exec(
		"INSERT OR IGNORE INTO friends (player_id, friend_id, status) VALUES (?, ?, ?)",
		fromID, toID, FriendPending,
	)
	return err
}

// AcceptFriendRequest accepts a friend request (creates mutual entries)
func (db *DB) AcceptFriendRequest(playerID, fromID int64) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update the incoming request to accepted
	_, err = tx.Exec(
		"UPDATE friends SET status = ? WHERE player_id = ? AND friend_id = ? AND status = ?",
		FriendAccepted, fromID, playerID, FriendPending,
	)
	if err != nil {
		return err
	}

	// Create the reverse relationship
	_, err = tx.Exec(
		"INSERT OR REPLACE INTO friends (player_id, friend_id, status) VALUES (?, ?, ?)",
		playerID, fromID, FriendAccepted,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// DeclineFriendRequest removes a pending friend request
func (db *DB) DeclineFriendRequest(playerID, fromID int64) error {
	_, err := db.conn.Exec(
		"DELETE FROM friends WHERE player_id = ? AND friend_id = ? AND status = ?",
		fromID, playerID, FriendPending,
	)
	return err
}

// RemoveFriend removes a mutual friendship
func (db *DB) RemoveFriend(playerID, friendID int64) error {
	_, err := db.conn.Exec(
		"DELETE FROM friends WHERE (player_id = ? AND friend_id = ?) OR (player_id = ? AND friend_id = ?)",
		playerID, friendID, friendID, playerID,
	)
	return err
}

// GetFriends returns accepted friends for a player
func (db *DB) GetFriends(playerID int64) ([]FriendRow, error) {
	rows, err := db.conn.Query(`
		SELECT f.friend_id, p.username, COALESCE(s.level, 1)
		FROM friends f
		JOIN players p ON p.id = f.friend_id
		LEFT JOIN stats s ON s.player_id = f.friend_id
		WHERE f.player_id = ? AND f.status = ?
		ORDER BY p.username`,
		playerID, FriendAccepted,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []FriendRow
	for rows.Next() {
		var r FriendRow
		if err := rows.Scan(&r.FriendID, &r.Username, &r.Level); err != nil {
			return nil, err
		}
		r.PlayerID = playerID
		r.Status = FriendAccepted
		result = append(result, r)
	}
	return result, rows.Err()
}

// GetPendingRequests returns incoming friend requests for a player
func (db *DB) GetPendingRequests(playerID int64) ([]FriendRow, error) {
	rows, err := db.conn.Query(`
		SELECT f.player_id, p.username, COALESCE(s.level, 1)
		FROM friends f
		JOIN players p ON p.id = f.player_id
		LEFT JOIN stats s ON s.player_id = f.player_id
		WHERE f.friend_id = ? AND f.status = ?
		ORDER BY f.created_at DESC`,
		playerID, FriendPending,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []FriendRow
	for rows.Next() {
		var r FriendRow
		if err := rows.Scan(&r.FriendID, &r.Username, &r.Level); err != nil {
			return nil, err
		}
		r.PlayerID = playerID
		r.Status = FriendPending
		result = append(result, r)
	}
	return result, rows.Err()
}

// GetCredits returns a player's credit balance
func (db *DB) GetCredits(playerID int64) (int, error) {
	var credits int
	err := db.conn.QueryRow("SELECT credits FROM stats WHERE player_id = ?", playerID).Scan(&credits)
	return credits, err
}

// AddCredits adds credits to a player's balance
func (db *DB) AddCredits(playerID int64, amount int) error {
	_, err := db.conn.Exec("UPDATE stats SET credits = credits + ? WHERE player_id = ?", amount, playerID)
	return err
}

// SpendCredits deducts credits if balance is sufficient. Returns false if insufficient.
func (db *DB) SpendCredits(playerID int64, amount int) (bool, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var credits int
	err = tx.QueryRow("SELECT credits FROM stats WHERE player_id = ?", playerID).Scan(&credits)
	if err != nil {
		return false, err
	}
	if credits < amount {
		return false, nil
	}
	_, err = tx.Exec("UPDATE stats SET credits = credits - ? WHERE player_id = ?", amount, playerID)
	if err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// ClaimDailyLogin processes daily login bonus. Returns (creditsAwarded, newStreak, error).
func (db *DB) ClaimDailyLogin(playerID int64) (int, int, error) {
	var lastLogin sql.NullString
	var streak int
	err := db.conn.QueryRow("SELECT last_login, login_streak FROM stats WHERE player_id = ?", playerID).
		Scan(&lastLogin, &streak)
	if err != nil {
		return 0, 0, err
	}

	now := time.Now().UTC()
	today := now.Format("2006-01-02")

	if lastLogin.Valid {
		lastDate := lastLogin.String[:10] // "YYYY-MM-DD"
		if lastDate == today {
			return 0, streak, nil // already claimed today
		}
		// Check if yesterday for streak continuation
		yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
		if lastDate == yesterday {
			streak++
		} else {
			streak = 1 // streak broken
		}
	} else {
		streak = 1
	}

	// Cap streak at 7 for bonus calculation
	bonusStreak := streak
	if bonusStreak > 7 {
		bonusStreak = 7
	}
	credits := 25 + bonusStreak*5 // 30, 35, 40, 45, 50, 55, 60

	_, err = db.conn.Exec(
		"UPDATE stats SET credits = credits + ?, last_login = ?, login_streak = ? WHERE player_id = ?",
		credits, now.Format(time.RFC3339), streak, playerID,
	)
	return credits, streak, err
}

// PurchaseSkin records a skin purchase
func (db *DB) PurchaseSkin(playerID int64, skinID string) error {
	_, err := db.conn.Exec(
		"INSERT OR IGNORE INTO player_skins (player_id, skin_id) VALUES (?, ?)",
		playerID, skinID,
	)
	return err
}

// HasSkin checks if a player owns a skin
func (db *DB) HasSkin(playerID int64, skinID string) (bool, error) {
	var count int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM player_skins WHERE player_id = ? AND skin_id = ?",
		playerID, skinID,
	).Scan(&count)
	return count > 0, err
}

// GetOwnedSkins returns all skin IDs a player owns
func (db *DB) GetOwnedSkins(playerID int64) ([]string, error) {
	rows, err := db.conn.Query(
		"SELECT skin_id FROM player_skins WHERE player_id = ? ORDER BY purchased_at",
		playerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// EquipSkin sets the player's equipped skin
func (db *DB) EquipSkin(playerID int64, skinID string) error {
	_, err := db.conn.Exec("UPDATE stats SET equipped_skin = ? WHERE player_id = ?", skinID, playerID)
	return err
}

// EquipTrail sets the player's equipped trail
func (db *DB) EquipTrail(playerID int64, trailID string) error {
	_, err := db.conn.Exec("UPDATE stats SET equipped_trail = ? WHERE player_id = ?", trailID, playerID)
	return err
}

// GetEquipped returns the player's equipped skin and trail
func (db *DB) GetEquipped(playerID int64) (string, string, error) {
	var skin, trail string
	err := db.conn.QueryRow("SELECT equipped_skin, equipped_trail FROM stats WHERE player_id = ?", playerID).
		Scan(&skin, &trail)
	return skin, trail, err
}

// UnlockAchievement records a player's achievement. Returns true if newly unlocked.
func (db *DB) UnlockAchievement(playerID int64, achievement string) (bool, error) {
	res, err := db.conn.Exec(
		"INSERT OR IGNORE INTO player_achievements (player_id, achievement) VALUES (?, ?)",
		playerID, achievement,
	)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	return affected > 0, err
}

// GetAchievements returns all achievements for a player
func (db *DB) GetAchievements(playerID int64) ([]string, error) {
	rows, err := db.conn.Query(
		"SELECT achievement FROM player_achievements WHERE player_id = ? ORDER BY unlocked_at",
		playerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
