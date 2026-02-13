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
	clientDir := flag.String("client", "", "Path to client directory (default: ../client)")
	clientRustDir := flag.String("client-rust", "", "Path to Rust client dist directory (default: ../client-rust/dist)")
	flag.Parse()

	if *clientDir == "" {
		exe, _ := os.Executable()
		*clientDir = filepath.Join(filepath.Dir(exe), "..", "client")
		// Fallback for development
		if _, err := os.Stat(*clientDir); os.IsNotExist(err) {
			*clientDir = "../client"
		}
	}

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

	hub := NewHub()
	go hub.Run()

	mux := SetupRoutes(hub, *clientDir, *clientRustDir)

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{Addr: *addr, Handler: mux}

	go func() {
		log.Printf("Server starting on %s", *addr)
		log.Printf("Serving client files from %s", *clientDir)
		if *clientRustDir != "" {
			log.Printf("Serving Rust client from %s at /rust/", *clientRustDir)
		}
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down...")
	server.Close()
}
