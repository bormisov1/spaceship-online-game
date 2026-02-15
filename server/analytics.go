package main

import (
	"database/sql"
	"log"
	"sync"
	"time"
)

// Event types for analytics tracking
const (
	EvtMatchStart    = "match_start"
	EvtMatchEnd      = "match_end"
	EvtPlayerKill    = "player_kill"
	EvtPlayerDeath   = "player_death"
	EvtPurchase      = "purchase"
	EvtAchievement   = "achievement"
	EvtSessionStart  = "session_start"
	EvtSessionEnd    = "session_end"
	EvtDailyLogin    = "daily_login"
	EvtLevelUp       = "level_up"
)

// AnalyticsEvent represents a single trackable event
type AnalyticsEvent struct {
	Type      string
	PlayerID  int64
	SessionID string
	Data      string // JSON metadata (optional)
	Timestamp time.Time
}

// Analytics handles event tracking with batched background writes
type Analytics struct {
	db      *DB
	events  chan AnalyticsEvent
	stop    chan struct{}
	wg      sync.WaitGroup

	// Live metrics (atomic-safe via mutex)
	mu              sync.RWMutex
	concurrentPeers int
	activeSessions  int
}

// NewAnalytics creates and starts the analytics background writer
func NewAnalytics(db *DB) *Analytics {
	a := &Analytics{
		db:     db,
		events: make(chan AnalyticsEvent, 1024),
		stop:   make(chan struct{}),
	}
	a.wg.Add(1)
	go a.writer()
	return a
}

// Track enqueues an event for async persistence (non-blocking)
func (a *Analytics) Track(evtType string, playerID int64, sessionID string, data string) {
	select {
	case a.events <- AnalyticsEvent{
		Type:      evtType,
		PlayerID:  playerID,
		SessionID: sessionID,
		Data:      data,
		Timestamp: time.Now().UTC(),
	}:
	default:
		// Channel full — drop event rather than blocking game loop
	}
}

// SetConcurrentPeers updates live player count metric
func (a *Analytics) SetConcurrentPeers(n int) {
	a.mu.Lock()
	a.concurrentPeers = n
	a.mu.Unlock()
}

// SetActiveSessions updates live session count metric
func (a *Analytics) SetActiveSessions(n int) {
	a.mu.Lock()
	a.activeSessions = n
	a.mu.Unlock()
}

// GetLiveMetrics returns current live metrics
func (a *Analytics) GetLiveMetrics() (int, int) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.concurrentPeers, a.activeSessions
}

// Stop gracefully shuts down the analytics writer
func (a *Analytics) Stop() {
	close(a.stop)
	a.wg.Wait()
}

// writer is the background goroutine that batches and writes events to DB
func (a *Analytics) writer() {
	defer a.wg.Done()

	batch := make([]AnalyticsEvent, 0, 64)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case evt := <-a.events:
			batch = append(batch, evt)
			// Flush immediately if batch is large
			if len(batch) >= 50 {
				a.flush(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				a.flush(batch)
				batch = batch[:0]
			}
		case <-a.stop:
			// Drain remaining events
			close(a.events)
			for evt := range a.events {
				batch = append(batch, evt)
			}
			if len(batch) > 0 {
				a.flush(batch)
			}
			return
		}
	}
}

// flush writes a batch of events to the database
func (a *Analytics) flush(events []AnalyticsEvent) {
	if a.db == nil || len(events) == 0 {
		return
	}
	tx, err := a.db.conn.Begin()
	if err != nil {
		log.Printf("analytics: begin tx error: %v", err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO analytics_events (event_type, player_id, session_id, data, created_at) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		log.Printf("analytics: prepare error: %v", err)
		return
	}
	defer stmt.Close()

	for _, evt := range events {
		pid := sql.NullInt64{Int64: evt.PlayerID, Valid: evt.PlayerID > 0}
		sid := sql.NullString{String: evt.SessionID, Valid: evt.SessionID != ""}
		data := sql.NullString{String: evt.Data, Valid: evt.Data != ""}
		_, err := stmt.Exec(evt.Type, pid, sid, data, evt.Timestamp.Format(time.RFC3339))
		if err != nil {
			log.Printf("analytics: insert error: %v", err)
		}
	}
	tx.Commit()
}

// --- Query methods for the API ---

// DAUCount returns number of distinct players active today
func (a *Analytics) DAUCount() (int, error) {
	if a.db == nil {
		return 0, nil
	}
	var count int
	err := a.db.conn.QueryRow(`
		SELECT COUNT(DISTINCT player_id) FROM analytics_events
		WHERE player_id IS NOT NULL AND created_at >= date('now')
	`).Scan(&count)
	return count, err
}

// WAUCount returns number of distinct players active in the last 7 days
func (a *Analytics) WAUCount() (int, error) {
	if a.db == nil {
		return 0, nil
	}
	var count int
	err := a.db.conn.QueryRow(`
		SELECT COUNT(DISTINCT player_id) FROM analytics_events
		WHERE player_id IS NOT NULL AND created_at >= date('now', '-7 days')
	`).Scan(&count)
	return count, err
}

// MAUCount returns number of distinct players active in the last 30 days
func (a *Analytics) MAUCount() (int, error) {
	if a.db == nil {
		return 0, nil
	}
	var count int
	err := a.db.conn.QueryRow(`
		SELECT COUNT(DISTINCT player_id) FROM analytics_events
		WHERE player_id IS NOT NULL AND created_at >= date('now', '-30 days')
	`).Scan(&count)
	return count, err
}

// MatchStats returns match counts by mode for the last N days
func (a *Analytics) MatchStats(days int) ([]MatchAnalytics, error) {
	if a.db == nil {
		return nil, nil
	}
	rows, err := a.db.conn.Query(`
		SELECT COALESCE(data, ''), COUNT(*) as cnt,
			AVG(CAST(
				CASE WHEN json_valid(data) THEN json_extract(data, '$.duration') ELSE NULL END
			AS REAL)) as avg_dur
		FROM analytics_events
		WHERE event_type = ? AND created_at >= date('now', '-' || ? || ' days')
		GROUP BY COALESCE(json_extract(data, '$.mode'), 'unknown')
		ORDER BY cnt DESC
	`, EvtMatchEnd, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MatchAnalytics
	for rows.Next() {
		var m MatchAnalytics
		var data string
		var avgDur sql.NullFloat64
		if err := rows.Scan(&data, &m.Count, &avgDur); err != nil {
			continue
		}
		m.AvgDuration = avgDur.Float64
		result = append(result, m)
	}
	return result, rows.Err()
}

// EventCounts returns counts of each event type for the last N days
func (a *Analytics) EventCounts(days int) (map[string]int, error) {
	if a.db == nil {
		return nil, nil
	}
	rows, err := a.db.conn.Query(`
		SELECT event_type, COUNT(*) FROM analytics_events
		WHERE created_at >= date('now', '-' || ? || ' days')
		GROUP BY event_type ORDER BY COUNT(*) DESC
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var evtType string
		var count int
		if err := rows.Scan(&evtType, &count); err != nil {
			continue
		}
		result[evtType] = count
	}
	return result, rows.Err()
}

// PopularPurchases returns the most purchased items
func (a *Analytics) PopularPurchases(limit int) ([]ItemAnalytics, error) {
	if a.db == nil {
		return nil, nil
	}
	rows, err := a.db.conn.Query(`
		SELECT COALESCE(json_extract(data, '$.item_id'), 'unknown') as item, COUNT(*) as cnt
		FROM analytics_events
		WHERE event_type = ? AND json_valid(data)
		GROUP BY item ORDER BY cnt DESC LIMIT ?
	`, EvtPurchase, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ItemAnalytics
	for rows.Next() {
		var ia ItemAnalytics
		if err := rows.Scan(&ia.ItemID, &ia.Count); err != nil {
			continue
		}
		result = append(result, ia)
	}
	return result, rows.Err()
}

// DailyActiveHistory returns DAU for the last N days
func (a *Analytics) DailyActiveHistory(days int) ([]DayCount, error) {
	if a.db == nil {
		return nil, nil
	}
	rows, err := a.db.conn.Query(`
		SELECT date(created_at) as day, COUNT(DISTINCT player_id)
		FROM analytics_events
		WHERE player_id IS NOT NULL AND created_at >= date('now', '-' || ? || ' days')
		GROUP BY day ORDER BY day
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DayCount
	for rows.Next() {
		var dc DayCount
		if err := rows.Scan(&dc.Day, &dc.Count); err != nil {
			continue
		}
		result = append(result, dc)
	}
	return result, rows.Err()
}

// MatchAnalytics holds aggregated match statistics
type MatchAnalytics struct {
	Mode        string  `json:"mode"`
	Count       int     `json:"count"`
	AvgDuration float64 `json:"avg_duration"`
}

// ItemAnalytics holds purchase count per item
type ItemAnalytics struct {
	ItemID string `json:"item_id"`
	Count  int    `json:"count"`
}

// DayCount holds a count for a specific day
type DayCount struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}
