package kafka

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"github.com/IBM/sarama"
)

// AnalyticsMetrics holds aggregated analytics data
type AnalyticsMetrics struct {
	TotalGames       int64              `json:"totalGames"`
	TotalMoves       int64              `json:"totalMoves"`
	BotGames         int64              `json:"botGames"`
	TotalDuration    int64              `json:"totalDuration"`
	WinCounts        map[string]int     `json:"winCounts"`
	GamesPerHour     map[string]int     `json:"gamesPerHour"`
	GamesPerDay      map[string]int     `json:"gamesPerDay"`
	PlayerStats      map[string]*PlayerMetrics `json:"playerStats"`
	mu               sync.RWMutex
}

// PlayerMetrics holds per-player analytics
type PlayerMetrics struct {
	Wins        int   `json:"wins"`
	Losses      int   `json:"losses"`
	Draws       int   `json:"draws"`
	TotalGames  int   `json:"totalGames"`
	TotalMoves  int64 `json:"totalMoves"`
	AvgDuration int64 `json:"avgDuration"`
}

// Consumer handles Kafka event consumption for analytics
type Consumer struct {
	consumer sarama.ConsumerGroup
	metrics  *AnalyticsMetrics
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewConsumer creates a new Kafka consumer
func NewConsumer() (*Consumer, error) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}

	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumer, err := sarama.NewConsumerGroup([]string{brokers}, "analytics-consumer", config)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Consumer{
		consumer: consumer,
		metrics: &AnalyticsMetrics{
			WinCounts:    make(map[string]int),
			GamesPerHour: make(map[string]int),
			GamesPerDay:  make(map[string]int),
			PlayerStats:  make(map[string]*PlayerMetrics),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	return c, nil
}

// Start begins consuming events
func (c *Consumer) Start() {
	go func() {
		for {
			if err := c.consumer.Consume(c.ctx, []string{TopicGameEvents}, c); err != nil {
				log.Printf("Consumer error: %v", err)
			}
			if c.ctx.Err() != nil {
				return
			}
		}
	}()
	log.Println("Kafka consumer started")
}

// Setup is called at the beginning of a new session
func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is called at the end of a session
func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim processes messages from a partition
func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		c.processMessage(msg)
		session.MarkMessage(msg, "")
	}
	return nil
}

// processMessage handles a single event message
func (c *Consumer) processMessage(msg *sarama.ConsumerMessage) {
	var event GameEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("Error unmarshaling event: %v", err)
		return
	}

	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()

	switch event.Type {
	case EventGameStart:
		c.handleGameStart(event)
	case EventMove:
		c.handleMove(event)
	case EventGameEnd:
		c.handleGameEnd(event)
	}
}

// handleGameStart processes game start events
func (c *Consumer) handleGameStart(event GameEvent) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}

	c.metrics.TotalGames++

	if isVsBot, ok := data["isVsBot"].(bool); ok && isVsBot {
		c.metrics.BotGames++
	}

	// Track games per hour and day
	hourKey := event.Timestamp.Format("2006-01-02-15")
	dayKey := event.Timestamp.Format("2006-01-02")
	c.metrics.GamesPerHour[hourKey]++
	c.metrics.GamesPerDay[dayKey]++

	// Initialize player stats
	if player1, ok := data["player1"].(string); ok {
		if c.metrics.PlayerStats[player1] == nil {
			c.metrics.PlayerStats[player1] = &PlayerMetrics{}
		}
		c.metrics.PlayerStats[player1].TotalGames++
	}
	if player2, ok := data["player2"].(string); ok && player2 != "BOT" {
		if c.metrics.PlayerStats[player2] == nil {
			c.metrics.PlayerStats[player2] = &PlayerMetrics{}
		}
		c.metrics.PlayerStats[player2].TotalGames++
	}
}

// handleMove processes move events
func (c *Consumer) handleMove(event GameEvent) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}

	c.metrics.TotalMoves++

	if player, ok := data["player"].(string); ok {
		if c.metrics.PlayerStats[player] == nil {
			c.metrics.PlayerStats[player] = &PlayerMetrics{}
		}
		c.metrics.PlayerStats[player].TotalMoves++
	}
}

// handleGameEnd processes game end events
func (c *Consumer) handleGameEnd(event GameEvent) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}

	// Track winner
	if winner, ok := data["winner"].(string); ok && winner != "" {
		c.metrics.WinCounts[winner]++
		if c.metrics.PlayerStats[winner] != nil {
			c.metrics.PlayerStats[winner].Wins++
		}
	}

	// Track duration
	if duration, ok := data["durationSeconds"].(float64); ok {
		c.metrics.TotalDuration += int64(duration)
	}
}

// GetMetrics returns a copy of the current metrics
func (c *Consumer) GetMetrics() *AnalyticsMetrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	// Return a copy to avoid race conditions
	copy := &AnalyticsMetrics{
		TotalGames:    c.metrics.TotalGames,
		TotalMoves:    c.metrics.TotalMoves,
		BotGames:      c.metrics.BotGames,
		TotalDuration: c.metrics.TotalDuration,
		WinCounts:     make(map[string]int),
		GamesPerHour:  make(map[string]int),
		GamesPerDay:   make(map[string]int),
		PlayerStats:   make(map[string]*PlayerMetrics),
	}

	for k, v := range c.metrics.WinCounts {
		copy.WinCounts[k] = v
	}
	for k, v := range c.metrics.GamesPerHour {
		copy.GamesPerHour[k] = v
	}
	for k, v := range c.metrics.GamesPerDay {
		copy.GamesPerDay[k] = v
	}
	for k, v := range c.metrics.PlayerStats {
		copy.PlayerStats[k] = &PlayerMetrics{
			Wins:       v.Wins,
			Losses:     v.Losses,
			Draws:      v.Draws,
			TotalGames: v.TotalGames,
			TotalMoves: v.TotalMoves,
		}
	}

	return copy
}

// GetAverageGameDuration returns the average game duration in seconds
func (c *Consumer) GetAverageGameDuration() float64 {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	if c.metrics.TotalGames == 0 {
		return 0
	}
	return float64(c.metrics.TotalDuration) / float64(c.metrics.TotalGames)
}

// GetMostFrequentWinner returns the player with most wins
func (c *Consumer) GetMostFrequentWinner() string {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	maxWins := 0
	winner := ""
	for player, wins := range c.metrics.WinCounts {
		if wins > maxWins {
			maxWins = wins
			winner = player
		}
	}
	return winner
}

// GetGamesPerHour returns games played in the last 24 hours by hour
func (c *Consumer) GetGamesPerHour() map[string]int {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	now := time.Now()
	result := make(map[string]int)
	
	for i := 0; i < 24; i++ {
		t := now.Add(-time.Duration(i) * time.Hour)
		key := t.Format("2006-01-02-15")
		result[key] = c.metrics.GamesPerHour[key]
	}
	
	return result
}

// Stop stops the consumer
func (c *Consumer) Stop() {
	c.cancel()
	c.consumer.Close()
}
