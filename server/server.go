package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

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

	// Serve static files
	fs := http.FileServer(http.Dir(clientDir))
	mux.Handle("/", fs)

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
