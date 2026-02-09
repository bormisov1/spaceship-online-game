package main

import (
	"log"
	"net/http"
	"path/filepath"
	"regexp"

	"github.com/gorilla/websocket"
)

var uuidPathRe = regexp.MustCompile(`^/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
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
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade error: %v", err)
			return
		}
		client := NewClient(hub, conn)
		hub.register <- client

		go client.WritePump()
		go client.ReadPump()
	})

	return mux
}
