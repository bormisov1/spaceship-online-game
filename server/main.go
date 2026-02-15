package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	clientRustDir := flag.String("client-rust", "", "Path to Rust client dist directory (default: ../client-rust/dist)")
	dbPath := flag.String("db", "spaceship.db", "Path to SQLite database file")
	flag.Parse()

	if *clientRustDir == "" {
		exe, _ := os.Executable()
		*clientRustDir = filepath.Join(filepath.Dir(exe), "..", "client-rust", "dist")
		// Fallback for development
		if _, err := os.Stat(*clientRustDir); os.IsNotExist(err) {
			*clientRustDir = "../client-rust/dist"
		}
		// If still doesn't exist, set to empty string (disable)
		if _, err := os.Stat(*clientRustDir); os.IsNotExist(err) {
			*clientRustDir = ""
		}
	}

	// Initialize database
	db, err := OpenDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	log.Printf("Database initialized at %s", *dbPath)

	hub := NewHub(db)
	go hub.Run()

	mux := SetupRoutes(hub, *clientRustDir)

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{Addr: *addr, Handler: mux}

	go func() {
		log.Printf("Server starting on %s", *addr)
		if *clientRustDir != "" {
			log.Printf("Serving Rust client from %s", *clientRustDir)
		} else {
			log.Printf("WARNING: No Rust client dist found")
		}
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down...")
	hub.analytics.Stop()
	server.Close()
}
