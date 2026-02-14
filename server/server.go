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
var rustUuidPathRe = regexp.MustCompile(`^/rust/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 16384,
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
func SetupRoutes(hub *Hub, clientDir string, clientRustDir string) *http.ServeMux {
	mux := http.NewServeMux()

	// Serve static files with no-cache so browsers always revalidate
	fs := http.FileServer(http.Dir(clientDir))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		// SPA: serve index.html for root and UUID paths
		if r.URL.Path == "/" || uuidPathRe.MatchString(r.URL.Path) {
			http.ServeFile(w, r, filepath.Join(clientDir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	}))

	// Serve Rust/WASM client at /rust/
	if clientRustDir != "" {
		rustFs := http.FileServer(http.Dir(clientRustDir))
		mux.Handle("/rust/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache")
			// SPA: serve index.html for /rust/ and /rust/{uuid} paths
			if r.URL.Path == "/rust/" || r.URL.Path == "/rust" || rustUuidPathRe.MatchString(r.URL.Path) {
				http.ServeFile(w, r, filepath.Join(clientRustDir, "index.html"))
				return
			}
			// Strip /rust/ prefix for static files
			http.StripPrefix("/rust/", rustFs).ServeHTTP(w, r)
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
