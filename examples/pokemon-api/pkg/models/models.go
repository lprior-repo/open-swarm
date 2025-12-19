package models

// PokemonStats represents the stats for a single Pokemon
type PokemonStats struct {
	HP        int `json:"hp"`
	Attack    int `json:"attack"`
	Defense   int `json:"defense"`
	SpAttack  int `json:"sp_attack"`
	SpDefense int `json:"sp_defense"`
	Speed     int `json:"speed"`
}

// Ability represents a Pokemon ability
type Ability struct {
	Name     string `json:"name"`
	IsHidden bool   `json:"is_hidden"`
}

// Pokemon represents a single Pokemon
type Pokemon struct {
	ID              int              `json:"id"`
	Name            string           `json:"name"`
	Type            string           `json:"type"`
	Height          float64          `json:"height"`
	Weight          float64          `json:"weight"`
	BaseExperience  int              `json:"base_experience"`
	Stats           PokemonStats     `json:"stats"`
	Abilities       []Ability        `json:"abilities"`
}

// ListResponse wraps a list of Pokemon with pagination
type ListResponse struct {
	Pokemon []Pokemon `json:"pokemon"`
	Total   int       `json:"total"`
	Limit   int       `json:"limit"`
	Offset  int       `json:"offset"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
