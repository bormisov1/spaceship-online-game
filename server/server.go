package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/gorilla/websocket"
	qrcode "github.com/skip2/go-qrcode"
)

var uuidPathRe = regexp.MustCompile(`^/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 8192,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Non-browser clients don't send Origin
		}
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		return u.Host == r.Host
	},
}

func extractIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// SetupRoutes configures HTTP routes
func SetupRoutes(hub *Hub, clientRustDir string) *http.ServeMux {
	mux := http.NewServeMux()

	// Serve Rust/WASM client
	if clientRustDir != "" {
		rustFs := http.FileServer(http.Dir(clientRustDir))

		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache")
			if r.URL.Path == "/" || uuidPathRe.MatchString(r.URL.Path) {
				http.ServeFile(w, r, filepath.Join(clientRustDir, "index.html"))
				return
			}
			rustFs.ServeHTTP(w, r)
		}))
	}

	// QR code endpoint – returns PNG for the given data parameter
	mux.HandleFunc("/api/qr", func(w http.ResponseWriter, r *http.Request) {
		data := r.URL.Query().Get("data")
		if data == "" {
			http.Error(w, "missing data param", http.StatusBadRequest)
			return
		}
		png, err := qrcode.Encode(data, qrcode.Medium, 256)
		if err != nil {
			http.Error(w, "qr encode error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(png)
	})

	// Debug endpoint
	mux.HandleFunc("/api/debug", func(w http.ResponseWriter, r *http.Request) {
		sessions := hub.sessions.ListSessions()
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		info := map[string]interface{}{
			"goroutines":  runtime.NumGoroutine(),
			"ws_clients":  hub.ClientCount(),
			"sessions":    sessions,
			"heap_mb":     float64(memStats.HeapAlloc) / 1024 / 1024,
			"total_conns": hub.TotalConns(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})

	// Analytics API endpoint
	mux.HandleFunc("/api/analytics", func(w http.ResponseWriter, r *http.Request) {
		if hub.analytics == nil {
			http.Error(w, "analytics not available", http.StatusServiceUnavailable)
			return
		}

		dau, _ := hub.analytics.DAUCount()
		wau, _ := hub.analytics.WAUCount()
		mau, _ := hub.analytics.MAUCount()
		peers, sessions := hub.analytics.GetLiveMetrics()
		eventCounts, _ := hub.analytics.EventCounts(7)
		matchStats, _ := hub.analytics.MatchStats(7)
		popularItems, _ := hub.analytics.PopularPurchases(10)
		dauHistory, _ := hub.analytics.DailyActiveHistory(30)

		resp := map[string]interface{}{
			"live": map[string]interface{}{
				"concurrent_players": peers,
				"active_sessions":    sessions,
				"ws_clients":         hub.ClientCount(),
			},
			"engagement": map[string]interface{}{
				"dau": dau,
				"wau": wau,
				"mau": mau,
			},
			"events_7d":      eventCounts,
			"matches_7d":     matchStats,
			"popular_items":  popularItems,
			"dau_history_30": dauHistory,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// WebSocket endpoint
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		if !hub.CanAccept(ip) {
			http.Error(w, "too many connections", http.StatusServiceUnavailable)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade error: %v", err)
			return
		}

		hub.TrackConnect(ip)

		client := NewClient(hub, conn, ip)
		hub.register <- client

		go client.WritePump()
		go client.ReadPump()
	})

	return mux
}
