package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/connect-four/internal/kafka"
	"github.com/connect-four/internal/matchmaker"
	"github.com/connect-four/internal/storage"
	"github.com/go-chi/chi/v5"
)

// Handlers holds API handler dependencies
type Handlers struct {
	store      *storage.PostgresStore
	matchmaker *matchmaker.Matchmaker
	producer   *kafka.Producer
	consumer   *kafka.Consumer
}

// NewHandlers creates a new API handlers instance
func NewHandlers(store *storage.PostgresStore, mm *matchmaker.Matchmaker, producer *kafka.Producer, consumer *kafka.Consumer) *Handlers {
	return &Handlers{
		store:      store,
		matchmaker: mm,
		producer:   producer,
		consumer:   consumer,
	}
}

// RegisterRoutes registers API routes
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Get("/leaderboard", h.GetLeaderboard)
	r.Delete("/leaderboard", h.ClearLeaderboard)
	r.Get("/stats/{username}", h.GetPlayerStats)
	r.Get("/analytics", h.GetAnalytics)
	r.Get("/status", h.GetStatus)
}

// GetLeaderboard returns the top players
func (h *Handlers) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	entries, err := h.store.GetLeaderboard(ctx, 20)
	if err != nil {
		http.Error(w, "Failed to get leaderboard", http.StatusInternalServerError)
		return
	}

	respondJSON(w, entries)
}

// ClearLeaderboard deletes all games and resets the leaderboard
func (h *Handlers) ClearLeaderboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	err := h.store.ClearAllGames(ctx)
	if err != nil {
		http.Error(w, "Failed to clear leaderboard", http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]string{"message": "Leaderboard cleared successfully"})
}

// GetPlayerStats returns statistics for a specific player
func (h *Handlers) GetPlayerStats(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	stats, err := h.store.GetPlayerStats(ctx, username)
	if err != nil {
		http.Error(w, "Failed to get player stats", http.StatusInternalServerError)
		return
	}

	respondJSON(w, stats)
}

// GetAnalytics returns game analytics
func (h *Handlers) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	
	// Get DB analytics
	dbAnalytics, err := h.store.GetAnalytics(ctx)
	if err != nil {
		http.Error(w, "Failed to get analytics", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"database": dbAnalytics,
		"realtime": map[string]interface{}{
			"activeGames":    h.matchmaker.GetActiveGameCount(),
			"playersWaiting": h.matchmaker.GetWaitingCount(),
			"kafkaEnabled":   h.producer.IsEnabled(),
		},
	}

	// Add Kafka metrics if available
	if h.consumer != nil {
		response["kafka"] = map[string]interface{}{
			"avgGameDuration":    h.consumer.GetAverageGameDuration(),
			"mostFrequentWinner": h.consumer.GetMostFrequentWinner(),
			"gamesPerHour":       h.consumer.GetGamesPerHour(),
			"metrics":            h.consumer.GetMetrics(),
		}
	}

	respondJSON(w, response)
}

// GetStatus returns server status
func (h *Handlers) GetStatus(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, map[string]interface{}{
		"status":         "ok",
		"activeGames":    h.matchmaker.GetActiveGameCount(),
		"playersWaiting": h.matchmaker.GetWaitingCount(),
		"kafkaEnabled":   h.producer.IsEnabled(),
	})
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
