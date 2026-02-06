package kafka

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/IBM/sarama"
	"github.com/connect-four/internal/game"
)

const (
	TopicGameEvents = "game-events"
)

// EventType represents the type of game event
type EventType string

const (
	EventGameStart EventType = "game_start"
	EventMove      EventType = "move"
	EventGameEnd   EventType = "game_end"
)

// GameEvent represents a game event for analytics
type GameEvent struct {
	Type      EventType `json:"type"`
	GameID    string    `json:"gameId"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// GameStartData contains data for game start events
type GameStartData struct {
	Player1 string `json:"player1"`
	Player2 string `json:"player2"`
	IsVsBot bool   `json:"isVsBot"`
}

// MoveData contains data for move events
type MoveData struct {
	Player  string `json:"player"`
	Column  int    `json:"column"`
	Row     int    `json:"row"`
	MoveNum int    `json:"moveNum"`
}

// GameEndData contains data for game end events
type GameEndData struct {
	Winner          string `json:"winner"`
	Result          string `json:"result"`
	DurationSeconds int    `json:"durationSeconds"`
	TotalMoves      int    `json:"totalMoves"`
	IsVsBot         bool   `json:"isVsBot"`
}

// Producer handles Kafka event production
type Producer struct {
	producer sarama.SyncProducer
	enabled  bool
}

// NewProducer creates a new Kafka producer
func NewProducer() (*Producer, error) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3

	producer, err := sarama.NewSyncProducer([]string{brokers}, config)
	if err != nil {
		log.Printf("Kafka producer not available: %v (analytics disabled)", err)
		return &Producer{enabled: false}, nil
	}

	log.Println("Kafka producer connected")
	return &Producer{producer: producer, enabled: true}, nil
}

// EmitGameStart emits a game start event
func (p *Producer) EmitGameStart(g *game.Game) {
	if !p.enabled {
		return
	}

	state := g.GetState()
	event := GameEvent{
		Type:      EventGameStart,
		GameID:    g.ID,
		Timestamp: time.Now(),
		Data: GameStartData{
			Player1: state.Player1,
			Player2: state.Player2,
			IsVsBot: state.IsVsBot,
		},
	}

	p.send(event)
}

// EmitMove emits a move event
func (p *Producer) EmitMove(g *game.Game, player string, column, row, moveNum int) {
	if !p.enabled {
		return
	}

	event := GameEvent{
		Type:      EventMove,
		GameID:    g.ID,
		Timestamp: time.Now(),
		Data: MoveData{
			Player:  player,
			Column:  column,
			Row:     row,
			MoveNum: moveNum,
		},
	}

	p.send(event)
}

// EmitGameEnd emits a game end event
func (p *Producer) EmitGameEnd(g *game.Game) {
	if !p.enabled {
		return
	}

	state := g.GetState()
	event := GameEvent{
		Type:      EventGameEnd,
		GameID:    g.ID,
		Timestamp: time.Now(),
		Data: GameEndData{
			Winner:          state.Winner,
			Result:          state.Result,
			DurationSeconds: g.GetDuration(),
			TotalMoves:      state.MoveCount,
			IsVsBot:         state.IsVsBot,
		},
	}

	p.send(event)
}

// send sends an event to Kafka
func (p *Producer) send(event GameEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: TopicGameEvents,
		Key:   sarama.StringEncoder(event.GameID),
		Value: sarama.ByteEncoder(data),
	}

	_, _, err = p.producer.SendMessage(msg)
	if err != nil {
		log.Printf("Error sending event to Kafka: %v", err)
	}
}

// Close closes the producer
func (p *Producer) Close() error {
	if p.producer != nil {
		return p.producer.Close()
	}
	return nil
}

// IsEnabled returns whether Kafka is enabled
func (p *Producer) IsEnabled() bool {
	return p.enabled
}
