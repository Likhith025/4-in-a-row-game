package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/connect-four/internal/game"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore handles database operations
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(ctx context.Context) (*PostgresStore, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/connect_four?sslmode=disable"
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing database URL: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("error pinging database: %w", err)
	}

	store := &PostgresStore{pool: pool}

	// Initialize schema
	if err := store.initSchema(ctx); err != nil {
		return nil, fmt.Errorf("error initializing schema: %w", err)
	}

	log.Println("Connected to PostgreSQL database")
	return store, nil
}

// initSchema creates the necessary tables
func (s *PostgresStore) initSchema(ctx context.Context) error {
	schema := `
		CREATE TABLE IF NOT EXISTS games (
			id UUID PRIMARY KEY,
			player1 VARCHAR(50) NOT NULL,
			player2 VARCHAR(50) NOT NULL,
			winner VARCHAR(50),
			is_forfeit BOOLEAN DEFAULT FALSE,
			is_draw BOOLEAN DEFAULT FALSE,
			duration_seconds INTEGER,
			move_count INTEGER,
			moves JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			ended_at TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_games_player1 ON games(player1);
		CREATE INDEX IF NOT EXISTS idx_games_player2 ON games(player2);
		CREATE INDEX IF NOT EXISTS idx_games_winner ON games(winner);
		CREATE INDEX IF NOT EXISTS idx_games_created_at ON games(created_at);

		CREATE TABLE IF NOT EXISTS game_analytics (
			id SERIAL PRIMARY KEY,
			date DATE NOT NULL,
			hour INTEGER,
			games_played INTEGER DEFAULT 0,
			avg_duration_seconds NUMERIC(10,2),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(date, hour)
		);
	`

	_, err := s.pool.Exec(ctx, schema)
	return err
}

// SaveGame stores a completed game
func (s *PostgresStore) SaveGame(ctx context.Context, g *game.Game) error {
	state := g.GetState()
	
	movesJSON, err := json.Marshal(g.Moves)
	if err != nil {
		movesJSON = []byte("[]")
	}

	isDraw := state.Result == string(game.ResultDraw)
	isForfeit := state.Result == string(game.ResultForfeit)

	query := `
		INSERT INTO games (id, player1, player2, winner, is_forfeit, is_draw, 
		                   duration_seconds, move_count, moves, created_at, ended_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO NOTHING
	`

	_, err = s.pool.Exec(ctx, query,
		g.ID,
		g.Player1.Username,
		g.Player2.Username,
		state.Winner,
		isForfeit,
		isDraw,
		g.GetDuration(),
		len(g.Moves),
		movesJSON,
		g.StartTime,
		g.EndTime,
	)

	return err
}

// GetLeaderboard returns the top players by wins
func (s *PostgresStore) GetLeaderboard(ctx context.Context, limit int) ([]LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		WITH player_stats AS (
			SELECT 
				username,
				COUNT(*) FILTER (WHERE winner = username) as wins,
				COUNT(*) FILTER (WHERE winner IS NULL) as draws,
				COUNT(*) FILTER (WHERE winner != username AND winner IS NOT NULL) as losses,
				COUNT(*) as games
			FROM (
				SELECT player1 as username, winner FROM games
				UNION ALL
				SELECT player2 as username, winner FROM games WHERE player2 != 'BOT'
			) subq
			GROUP BY username
		)
		SELECT 
			username, wins, losses, draws, games,
			CASE WHEN games > 0 THEN ROUND(wins::numeric / games * 100, 1) ELSE 0 END as win_rate
		FROM player_stats
		ORDER BY wins DESC, win_rate DESC
		LIMIT $1
	`

	rows, err := s.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	rank := 1
	for rows.Next() {
		var entry LeaderboardEntry
		err := rows.Scan(&entry.Username, &entry.Wins, &entry.Losses, &entry.Draws, &entry.Games, &entry.WinRate)
		if err != nil {
			return nil, err
		}
		entry.Rank = rank
		entries = append(entries, entry)
		rank++
	}

	return entries, nil
}

// GetPlayerStats returns detailed statistics for a player
func (s *PostgresStore) GetPlayerStats(ctx context.Context, username string) (*PlayerStats, error) {
	query := `
		WITH player_games AS (
			SELECT 
				g.*,
				CASE 
					WHEN g.player1 = $1 THEN g.player2
					ELSE g.player1
				END as opponent
			FROM games g
			WHERE g.player1 = $1 OR g.player2 = $1
		)
		SELECT 
			COUNT(*) FILTER (WHERE winner = $1) as wins,
			COUNT(*) FILTER (WHERE winner IS NULL) as draws,
			COUNT(*) FILTER (WHERE winner != $1 AND winner IS NOT NULL) as losses,
			COUNT(*) as total_games,
			COUNT(*) FILTER (WHERE opponent = 'BOT' AND winner = $1) as bot_wins,
			COUNT(*) FILTER (WHERE opponent = 'BOT' AND winner = 'BOT') as bot_losses,
			COALESCE(AVG(duration_seconds), 0) as avg_game_length
		FROM player_games
	`

	var stats PlayerStats
	stats.Username = username

	err := s.pool.QueryRow(ctx, query, username).Scan(
		&stats.Wins,
		&stats.Draws,
		&stats.Losses,
		&stats.TotalGames,
		&stats.BotWins,
		&stats.BotLosses,
		&stats.AvgGameLength,
	)
	if err != nil {
		return nil, err
	}

	if stats.TotalGames > 0 {
		stats.WinRate = float64(stats.Wins) / float64(stats.TotalGames) * 100
	}

	return &stats, nil
}

// GetAnalytics returns aggregated game analytics
func (s *PostgresStore) GetAnalytics(ctx context.Context) (*GameAnalytics, error) {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	thisHour := now.Truncate(time.Hour)

	query := `
		SELECT 
			COUNT(*) as total_games,
			COUNT(DISTINCT player1) + COUNT(DISTINCT CASE WHEN player2 != 'BOT' THEN player2 END) as total_players,
			COALESCE(AVG(duration_seconds), 0) as avg_duration,
			COUNT(*) FILTER (WHERE player2 = 'BOT') as bot_games,
			COUNT(*) FILTER (WHERE created_at >= $1) as games_today,
			COUNT(*) FILTER (WHERE created_at >= $2) as games_this_hour,
			(SELECT winner FROM games WHERE winner IS NOT NULL GROUP BY winner ORDER BY COUNT(*) DESC LIMIT 1) as most_frequent_winner
		FROM games
	`

	var analytics GameAnalytics
	var mostFrequentWinner *string

	err := s.pool.QueryRow(ctx, query, today, thisHour).Scan(
		&analytics.TotalGames,
		&analytics.TotalPlayers,
		&analytics.AvgGameDuration,
		&analytics.BotGamesPlayed,
		&analytics.GamesToday,
		&analytics.GamesThisHour,
		&mostFrequentWinner,
	)
	if err != nil {
		return nil, err
	}

	if mostFrequentWinner != nil {
		analytics.MostFrequentWinner = *mostFrequentWinner
	}

	return &analytics, nil
}

// Close closes the database connection pool
func (s *PostgresStore) Close() {
	s.pool.Close()
}
