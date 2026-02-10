package main

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"

	"github.com/gorilla/websocket"
)

var uuidPathRe = regexp.MustCompile(`^/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
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
func SetupRoutes(hub *Hub, clientDir string) *http.ServeMux {
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
