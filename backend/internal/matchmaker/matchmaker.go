package matchmaker

import (
	"log"
	"sync"
	"time"

	"github.com/connect-four/internal/game"
)

const MatchmakingTimeout = 10 * time.Second

// WaitingPlayer represents a player waiting for a match
type WaitingPlayer struct {
	Username  string
	JoinedAt  time.Time
	MatchChan chan *game.Game
}

// Matchmaker handles player matching
type Matchmaker struct {
	waitingQueue []*WaitingPlayer
	activeGames  map[string]*game.Game       // gameID -> game
	playerGames  map[string]string           // username -> gameID
	mu           sync.Mutex
	onGameStart  func(g *game.Game)
}

// NewMatchmaker creates a new matchmaker instance
func NewMatchmaker() *Matchmaker {
	return &Matchmaker{
		waitingQueue: make([]*WaitingPlayer, 0),
		activeGames:  make(map[string]*game.Game),
		playerGames:  make(map[string]string),
	}
}

// SetOnGameStart sets the callback for when a game starts
func (m *Matchmaker) SetOnGameStart(callback func(g *game.Game)) {
	m.onGameStart = callback
}

// JoinQueue adds a player to the matchmaking queue
// Returns a channel that will receive the game when matched
func (m *Matchmaker) JoinQueue(username string) (<-chan *game.Game, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if player is already in a game
	if gameID, exists := m.playerGames[username]; exists {
		if g, ok := m.activeGames[gameID]; ok {
			// Return existing game for reconnection
			ch := make(chan *game.Game, 1)
			ch <- g
			return ch, nil
		}
	}

	// Check if there's a waiting player to match with
	if len(m.waitingQueue) > 0 {
		// Match with the first waiting player
		opponent := m.waitingQueue[0]
		m.waitingQueue = m.waitingQueue[1:]

		// Create new game
		g := game.NewGame(opponent.Username)
		g.AddPlayer2(username, false)

		// Register the game
		m.activeGames[g.ID] = g
		m.playerGames[opponent.Username] = g.ID
		m.playerGames[username] = g.ID

		// Notify the waiting player
		opponent.MatchChan <- g

		// Return the game to the joining player
		ch := make(chan *game.Game, 1)
		ch <- g

		if m.onGameStart != nil {
			go m.onGameStart(g)
		}

		return ch, nil
	}

	// No opponent available, add to queue
	waiting := &WaitingPlayer{
		Username:  username,
		JoinedAt:  time.Now(),
		MatchChan: make(chan *game.Game, 1),
	}
	m.waitingQueue = append(m.waitingQueue, waiting)

	// Start timeout goroutine
	go m.handleMatchmakingTimeout(waiting)

	return waiting.MatchChan, nil
}

// handleMatchmakingTimeout handles the 10-second timeout for matchmaking
func (m *Matchmaker) handleMatchmakingTimeout(waiting *WaitingPlayer) {
	time.Sleep(MatchmakingTimeout)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if still in queue
	for i, w := range m.waitingQueue {
		if w == waiting {
			// Remove from queue
			m.waitingQueue = append(m.waitingQueue[:i], m.waitingQueue[i+1:]...)

			// Create game with bot
			log.Printf("[Matchmaker] Creating bot game for player: %s", waiting.Username)
			g := game.NewGame(waiting.Username)
			g.AddPlayer2("BOT", true)
			log.Printf("[Matchmaker] Bot game created: ID=%s, Player2IsBot=%v", g.ID, g.Player2.IsBot)

			// Register the game
			m.activeGames[g.ID] = g
			m.playerGames[waiting.Username] = g.ID

			// Notify the player
			waiting.MatchChan <- g

			if m.onGameStart != nil {
				go m.onGameStart(g)
			}

			return
		}
	}

	// Player was already matched, do nothing
}

// GetGame returns a game by ID
func (m *Matchmaker) GetGame(gameID string) *game.Game {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeGames[gameID]
}

// GetGameByPlayer returns a game by player username
func (m *Matchmaker) GetGameByPlayer(username string) *game.Game {
	m.mu.Lock()
	defer m.mu.Unlock()

	if gameID, exists := m.playerGames[username]; exists {
		return m.activeGames[gameID]
	}
	return nil
}

// RemoveGame removes a completed game from active games
func (m *Matchmaker) RemoveGame(gameID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if g, exists := m.activeGames[gameID]; exists {
		delete(m.playerGames, g.Player1.Username)
		if g.Player2 != nil && !g.Player2.IsBot {
			delete(m.playerGames, g.Player2.Username)
		}
		delete(m.activeGames, gameID)
	}
}

// LeaveQueue removes a player from the waiting queue
func (m *Matchmaker) LeaveQueue(username string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, w := range m.waitingQueue {
		if w.Username == username {
			m.waitingQueue = append(m.waitingQueue[:i], m.waitingQueue[i+1:]...)
			close(w.MatchChan)
			return
		}
	}
}

// GetActiveGameCount returns the number of active games
func (m *Matchmaker) GetActiveGameCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.activeGames)
}

// GetWaitingCount returns the number of players waiting
func (m *Matchmaker) GetWaitingCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.waitingQueue)
}
