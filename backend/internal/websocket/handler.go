package websocket

import (
	"encoding/json"
	"log"

	"github.com/connect-four/internal/game"
	"github.com/connect-four/internal/matchmaker"
)

// Message types
const (
	TypeJoin                 = "join"
	TypeMove                 = "move"
	TypeReconnect            = "reconnect"
	TypeWaiting              = "waiting"
	TypeMatched              = "matched"
	TypeState                = "state"
	TypeGameOver             = "gameOver"
	TypeError                = "error"
	TypeOpponentDisconnected = "opponentDisconnected"
	TypeOpponentReconnected  = "opponentReconnected"
)

// Message represents a WebSocket message
type Message struct {
	Type              string           `json:"type"`
	Username          string           `json:"username,omitempty"`
	Column            int              `json:"column,omitempty"`
	Row               int              `json:"row,omitempty"`
	GameID            string           `json:"gameId,omitempty"`
	Opponent          string           `json:"opponent,omitempty"`
	YourTurn          bool             `json:"yourTurn,omitempty"`
	State             *game.GameState  `json:"state,omitempty"`
	Winner            string           `json:"winner,omitempty"`
	Reason            string           `json:"reason,omitempty"`
	Message           string           `json:"message,omitempty"`
	ReconnectDeadline string           `json:"reconnectDeadline,omitempty"`
	PlayerNum         int              `json:"playerNum,omitempty"`
}

// IncomingMessage represents a message from the client
type IncomingMessage struct {
	Type     string `json:"type"`
	Column   int    `json:"column,omitempty"`
	GameID   string `json:"gameId,omitempty"`
	Username string `json:"username,omitempty"`
}

// Handler processes WebSocket messages
type Handler struct {
	hub        *Hub
	matchmaker *matchmaker.Matchmaker
}

// NewHandler creates a new message handler
func NewHandler(hub *Hub, mm *matchmaker.Matchmaker) *Handler {
	return &Handler{
		hub:        hub,
		matchmaker: mm,
	}
}

// HandleMessage processes an incoming message
func (h *Handler) HandleMessage(client *Client, data []byte) {
	var msg IncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Error parsing message: %v", err)
		client.sendMessage(Message{Type: TypeError, Message: "Invalid message format"})
		return
	}

	switch msg.Type {
	case TypeJoin:
		h.handleJoin(client)
	case TypeMove:
		h.handleMove(client, msg.Column)
	case TypeReconnect:
		h.handleReconnect(client, msg.GameID)
	default:
		client.sendMessage(Message{Type: TypeError, Message: "Unknown message type"})
	}
}

// handleJoin handles a player joining the matchmaking queue
func (h *Handler) handleJoin(client *Client) {
	// Check for existing game to reconnect
	existingGame := h.matchmaker.GetGameByPlayer(client.username)
	if existingGame != nil && existingGame.GetState().Status != game.StatusFinished {
		h.handleReconnectToGame(client, existingGame)
		return
	}

	// Notify client they're waiting
	client.sendMessage(Message{
		Type:    TypeWaiting,
		Message: "Looking for opponent...",
	})

	// Join matchmaking queue
	gameChan, err := h.matchmaker.JoinQueue(client.username)
	if err != nil {
		client.sendMessage(Message{Type: TypeError, Message: err.Error()})
		return
	}

	// Wait for match in goroutine
	go func() {
		g := <-gameChan
		if g == nil {
			return
		}

		// Register client to game
		h.hub.RegisterToGame(g.ID, client)

		// Determine opponent
		state := g.GetState()
		opponent := state.Player2
		yourTurn := state.CurrentTurn == game.Player1
		if client.username == state.Player2 {
			opponent = state.Player1
			yourTurn = state.CurrentTurn == game.Player2
		}

		// Send matched message
		client.sendMessage(Message{
			Type:      TypeMatched,
			GameID:    g.ID,
			Opponent:  opponent,
			YourTurn:  yourTurn,
			PlayerNum: g.GetPlayerByUsername(client.username),
			State:     state,
		})
	}()
}

// handleMove handles a player making a move
func (h *Handler) handleMove(client *Client, column int) {
	log.Printf("[handleMove] Player %s attempting move on column %d", client.username, column)
	
	if client.gameID == "" {
		log.Printf("[handleMove] Error: client has no gameID")
		client.sendMessage(Message{Type: TypeError, Message: "Not in a game"})
		return
	}

	g := h.matchmaker.GetGame(client.gameID)
	if g == nil {
		log.Printf("[handleMove] Error: game not found for ID %s", client.gameID)
		client.sendMessage(Message{Type: TypeError, Message: "Game not found"})
		return
	}

	playerNum := g.GetPlayerByUsername(client.username)
	if playerNum == 0 {
		log.Printf("[handleMove] Error: player not found in game")
		client.sendMessage(Message{Type: TypeError, Message: "Player not found"})
		return
	}

	log.Printf("[handleMove] Player %s (num=%d) making move on column %d", client.username, playerNum, column)
	row, err := g.MakeMove(playerNum, column)
	if err != nil {
		log.Printf("[handleMove] MakeMove error: %v", err)
		client.sendMessage(Message{Type: TypeError, Message: err.Error()})
		return
	}
	log.Printf("[handleMove] Move succeeded, disc placed at row %d", row)

	// Broadcast updated state
	h.hub.BroadcastGameState(g)

	// Check if game ended
	state := g.GetState()
	log.Printf("[handleMove] After move - Status=%s, CurrentTurn=%d, IsVsBot=%v", state.Status, state.CurrentTurn, state.IsVsBot)
	
	if state.Status == game.StatusFinished {
		log.Printf("[handleMove] Game finished, winner=%s", state.Winner)
		h.hub.broadcastToGame(g.ID, Message{
			Type:   TypeGameOver,
			Winner: state.Winner,
			Reason: state.Result,
		})
		h.hub.handleGameEnd(g)
		return
	}

	// If next turn is bot, make bot move
	log.Printf("[handleMove] Checking bot trigger: Player2=%v, IsBot=%v, CurrentTurn=%d", 
		g.Player2 != nil, g.Player2 != nil && g.Player2.IsBot, state.CurrentTurn)
	
	if g.Player2 != nil && g.Player2.IsBot && state.CurrentTurn == game.Player2 {
		log.Printf("[handleMove] Triggering bot move...")
		go h.hub.HandleBotMove(g)
	} else {
		log.Printf("[handleMove] Bot move NOT triggered")
	}
	
	_ = row // row is part of broadcast state
}

// handleReconnect handles a player trying to reconnect to a game
func (h *Handler) handleReconnect(client *Client, gameID string) {
	g := h.matchmaker.GetGame(gameID)
	if g == nil {
		// Try to find by player
		g = h.matchmaker.GetGameByPlayer(client.username)
	}

	if g == nil {
		client.sendMessage(Message{Type: TypeError, Message: "Game not found"})
		return
	}

	h.handleReconnectToGame(client, g)
}

// handleReconnectToGame handles reconnection to a specific game
func (h *Handler) handleReconnectToGame(client *Client, g *game.Game) {
	playerNum := g.GetPlayerByUsername(client.username)
	if playerNum == 0 {
		client.sendMessage(Message{Type: TypeError, Message: "Not a player in this game"})
		return
	}

	// Try to reconnect
	if !g.PlayerReconnected(playerNum) {
		state := g.GetState()
		if state.Status == game.StatusFinished {
			client.sendMessage(Message{Type: TypeError, Message: "Game has already ended"})
			return
		}
		client.sendMessage(Message{Type: TypeError, Message: "Reconnection failed"})
		return
	}

	// Register client to game
	h.hub.RegisterToGame(g.ID, client)

	// Notify opponent
	h.hub.broadcastToGame(g.ID, Message{
		Type: TypeOpponentReconnected,
	})

	// Send current state
	state := g.GetState()
	opponent := state.Player2
	yourTurn := state.CurrentTurn == game.Player1
	if client.username == state.Player2 {
		opponent = state.Player1
		yourTurn = state.CurrentTurn == game.Player2
	}

	client.sendMessage(Message{
		Type:      TypeMatched,
		GameID:    g.ID,
		Opponent:  opponent,
		YourTurn:  yourTurn,
		PlayerNum: playerNum,
		State:     state,
	})
}
