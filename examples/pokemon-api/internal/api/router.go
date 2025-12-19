package api

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type API struct {
	db *sql.DB
}

func NewRouter(database interface{}) *chi.Mux {
	router := chi.NewRouter()

	// Mount API routes
	router.Route("/api", func(r chi.Router) {
		// Pokemon list endpoints
		r.Get("/pokemon", ListPokemon)
		r.Get("/pokemon/{id}", GetPokemon)
		r.Get("/pokemon/search", SearchPokemon)

		// Filter endpoints
		r.Get("/pokemon/type/{type}", FilterByType)
		r.Get("/pokemon/stats/{stat}/gte/{value}", FilterByStats)

		// Search results for HTMX
		r.Get("/search-results", SearchResults)
	})

	// Frontend routes
	router.Get("/", Home)

	return router
}

// Placeholder handlers - will be implemented by agents
func ListPokemon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"pokemon":[],"total":0,"limit":20,"offset":0}`))
}

func GetPokemon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error":"not found"}`))
}

func SearchPokemon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"pokemon":[],"total":0}`))
}

func FilterByType(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"pokemon":[],"total":0}`))
}

func FilterByStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"pokemon":[],"total":0}`))
}

func SearchResults(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<div id="results"></div>`))
}

func Home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<html><body><h1>Pokemon API</h1></body></html>`))
}
