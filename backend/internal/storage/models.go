package storage

import (
	"time"
)

// CompletedGame represents a finished game stored in the database
type CompletedGame struct {
	ID              string    `json:"id"`
	Player1         string    `json:"player1"`
	Player2         string    `json:"player2"`
	Winner          string    `json:"winner"`
	IsForfeit       bool      `json:"isForfeit"`
	IsDraw          bool      `json:"isDraw"`
	DurationSeconds int       `json:"durationSeconds"`
	MoveCount       int       `json:"moveCount"`
	Moves           string    `json:"moves"` // JSON string
	CreatedAt       time.Time `json:"createdAt"`
	EndedAt         time.Time `json:"endedAt"`
}

// LeaderboardEntry represents a player's ranking
type LeaderboardEntry struct {
	Rank     int    `json:"rank"`
	Username string `json:"username"`
	Wins     int    `json:"wins"`
	Losses   int    `json:"losses"`
	Draws    int    `json:"draws"`
	Games    int    `json:"games"`
	WinRate  float64 `json:"winRate"`
}

// PlayerStats represents detailed player statistics
type PlayerStats struct {
	Username       string  `json:"username"`
	Wins           int     `json:"wins"`
	Losses         int     `json:"losses"`
	Draws          int     `json:"draws"`
	TotalGames     int     `json:"totalGames"`
	WinRate        float64 `json:"winRate"`
	BotWins        int     `json:"botWins"`
	BotLosses      int     `json:"botLosses"`
	AvgGameLength  float64 `json:"avgGameLength"`
	CurrentStreak  int     `json:"currentStreak"`
}

// GameAnalytics represents aggregated game analytics
type GameAnalytics struct {
	TotalGames         int     `json:"totalGames"`
	TotalPlayers       int     `json:"totalPlayers"`
	AvgGameDuration    float64 `json:"avgGameDuration"`
	BotGamesPlayed     int     `json:"botGamesPlayed"`
	GamesToday         int     `json:"gamesToday"`
	GamesThisHour      int     `json:"gamesThisHour"`
	MostFrequentWinner string  `json:"mostFrequentWinner"`
}
