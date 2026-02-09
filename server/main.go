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
	flag.Parse()

	if *clientDir == "" {
		exe, _ := os.Executable()
		*clientDir = filepath.Join(filepath.Dir(exe), "..", "client")
		// Fallback for development
		if _, err := os.Stat(*clientDir); os.IsNotExist(err) {
			*clientDir = "../client"
		}
	}

	hub := NewHub()
	go hub.Run()

	mux := SetupRoutes(hub, *clientDir)

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{Addr: *addr, Handler: mux}

	go func() {
		log.Printf("Server starting on %s", *addr)
		log.Printf("Serving client files from %s", *clientDir)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down...")
	server.Close()
}
