package main

import (
	"fmt"
	"log"
	"net/http"
	"pokemon-api/internal/api"
	"pokemon-api/internal/db"
)

func main() {
	// Initialize database
	database, err := db.NewDB("internal/db/pokemon.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create router
	router := api.NewRouter(database)

	// Start server
	port := ":3000"
	fmt.Printf("🚀 Pokemon API server starting on %s\n", port)
	if err := http.ListenAndServe(port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
