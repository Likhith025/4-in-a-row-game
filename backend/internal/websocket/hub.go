package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/connect-four/internal/game"
	"github.com/connect-four/internal/matchmaker"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients by username
	clients map[string]*Client

	// Clients by game ID
	gameClients map[string]map[string]*Client

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Matchmaker reference
	matchmaker *matchmaker.Matchmaker

	// Callbacks
	onGameEnd func(g *game.Game)

	mu sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub(mm *matchmaker.Matchmaker) *Hub {
	return &Hub{
		clients:     make(map[string]*Client),
		gameClients: make(map[string]map[string]*Client),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		matchmaker:  mm,
	}
}

// SetOnGameEnd sets the callback for when a game ends
func (h *Hub) SetOnGameEnd(callback func(g *game.Game)) {
	h.onGameEnd = callback
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.username] = client
			h.mu.Unlock()
			log.Printf("Client registered: %s", client.username)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.username]; ok {
				delete(h.clients, client.username)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("Client unregistered: %s", client.username)

			// Handle disconnect for active game
			h.handleDisconnect(client)
		}
	}
}

// handleDisconnect handles a player disconnect
func (h *Hub) handleDisconnect(client *Client) {
	if client.gameID == "" {
		// Player was not in a game, just leave queue
		h.matchmaker.LeaveQueue(client.username)
		return
	}

	g := h.matchmaker.GetGame(client.gameID)
	if g == nil {
		return
	}

	playerNum := g.GetPlayerByUsername(client.username)
	if playerNum == 0 {
		return
	}

	// Handle bot game - forfeit immediately since bot doesn't wait
	if g.Player2 != nil && g.Player2.IsBot && playerNum == game.Player1 {
		g.Forfeit(game.Player1)
		h.handleGameEnd(g)
		return
	}

	// Mark player as disconnected
	g.PlayerDisconnected(playerNum)

	// Notify opponent
	h.notifyOpponentDisconnected(g, playerNum)

	// Start 30-second timeout
	go h.handleReconnectTimeout(g, playerNum)
}

// handleReconnectTimeout waits 30 seconds for reconnection
func (h *Hub) handleReconnectTimeout(g *game.Game, disconnectedPlayer int) {
	time.Sleep(30 * time.Second)

	state := g.GetState()
	if state.Status == game.StatusDisconnect {
		// Player didn't reconnect, forfeit
		g.Forfeit(disconnectedPlayer)
		h.handleGameEnd(g)

		// Notify remaining player
		h.broadcastToGame(g.ID, Message{
			Type:   TypeGameOver,
			Winner: g.Winner.Username,
			Reason: "forfeit",
		})
	}
}

// notifyOpponentDisconnected notifies the opponent about disconnect
func (h *Hub) notifyOpponentDisconnected(g *game.Game, disconnectedPlayerNum int) {
	deadline := time.Now().Add(30 * time.Second)

	h.mu.RLock()
	clients := h.gameClients[g.ID]
	h.mu.RUnlock()

	for username, client := range clients {
		playerNum := g.GetPlayerByUsername(username)
		if playerNum != disconnectedPlayerNum {
			msg := Message{
				Type:              TypeOpponentDisconnected,
				ReconnectDeadline: deadline.Format(time.RFC3339),
			}
			client.sendMessage(msg)
		}
	}
}

// RegisterToGame adds a client to a game's client list
func (h *Hub) RegisterToGame(gameID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.gameClients[gameID] == nil {
		h.gameClients[gameID] = make(map[string]*Client)
	}
	h.gameClients[gameID][client.username] = client
	client.gameID = gameID
}

// BroadcastGameState sends game state to all players in a game
func (h *Hub) BroadcastGameState(g *game.Game) {
	state := g.GetState()
	h.broadcastToGame(g.ID, Message{
		Type:  TypeState,
		State: state,
	})
}

// broadcastToGame sends a message to all clients in a game
func (h *Hub) broadcastToGame(gameID string, msg Message) {
	h.mu.RLock()
	clients := h.gameClients[gameID]
	h.mu.RUnlock()

	log.Printf("[broadcast] Sending %s to game %s (clients: %d)", msg.Type, gameID, len(clients))

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	for username, client := range clients {
		select {
		case client.send <- data:
			log.Printf("[broadcast] Sent %s to %s", msg.Type, username)
		default:
			log.Printf("[broadcast] Failed to send %s to %s - buffer full", msg.Type, username)
		}
	}
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(username string, msg Message) {
	h.mu.RLock()
	client, ok := h.clients[username]
	h.mu.RUnlock()

	if !ok {
		return
	}

	client.sendMessage(msg)
}

// handleGameEnd processes game completion
func (h *Hub) handleGameEnd(g *game.Game) {
	if h.onGameEnd != nil {
		h.onGameEnd(g)
	}

	// Clean up after a delay
	go func() {
		time.Sleep(5 * time.Second)
		h.mu.Lock()
		delete(h.gameClients, g.ID)
		h.mu.Unlock()
		h.matchmaker.RemoveGame(g.ID)
	}()
}

// HandleBotMove processes the bot's move
func (h *Hub) HandleBotMove(g *game.Game) {
	log.Printf("HandleBotMove called for game %s", g.ID)
	
	if g.Player2 == nil || !g.Player2.IsBot {
		log.Printf("Bot move skipped: Player2 is nil or not a bot")
		return
	}

	state := g.GetState()
	log.Printf("Game state: Status=%s, CurrentTurn=%d", state.Status, state.CurrentTurn)
	
	if state.Status != game.StatusPlaying {
		log.Printf("Bot move skipped: game not playing (status=%s)", state.Status)
		return
	}
	
	if state.CurrentTurn != game.Player2 {
		log.Printf("Bot move skipped: not bot's turn (currentTurn=%d)", state.CurrentTurn)
		return
	}

	// Add a small delay to make it feel more natural
	time.Sleep(500 * time.Millisecond)

	log.Printf("Bot computing best move...")
	col, row, err := g.MakeBotMove()
	if err != nil {
		log.Printf("Bot move error: %v", err)
		return
	}
	log.Printf("Bot played column %d, row %d", col, row)

	// Broadcast the move
	h.broadcastToGame(g.ID, Message{
		Type:   TypeState,
		State:  g.GetState(),
		Column: col,
		Row:    row,
	})

	// Check if game ended
	newState := g.GetState()
	if newState.Status == game.StatusFinished {
		log.Printf("Game finished after bot move. Winner: %s", newState.Winner)
		h.broadcastToGame(g.ID, Message{
			Type:   TypeGameOver,
			Winner: newState.Winner,
			Reason: newState.Result,
		})
		h.handleGameEnd(g)
	}
}

// GetClient returns a client by username
func (h *Hub) GetClient(username string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.clients[username]
}
