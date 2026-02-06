package game

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// GameStatus represents the current state of the game
type GameStatus string

const (
	StatusWaiting    GameStatus = "waiting"
	StatusPlaying    GameStatus = "playing"
	StatusFinished   GameStatus = "finished"
	StatusDisconnect GameStatus = "disconnected"
)

// GameResult represents the outcome of a game
type GameResult string

const (
	ResultWinPlayer1 GameResult = "player1_win"
	ResultWinPlayer2 GameResult = "player2_win"
	ResultDraw       GameResult = "draw"
	ResultForfeit    GameResult = "forfeit"
)

// Player represents a player in the game
type Player struct {
	Username    string
	PlayerNum   int  // Player1 or Player2
	IsBot       bool
	IsConnected bool
}

// Move represents a single move in the game
type Move struct {
	PlayerNum int       `json:"playerNum"`
	Column    int       `json:"column"`
	Row       int       `json:"row"`
	Timestamp time.Time `json:"timestamp"`
}

// Game represents a Connect Four game instance
type Game struct {
	ID                 string
	Player1            *Player
	Player2            *Player
	Board              *Board
	CurrentTurn        int // Player1 or Player2
	Status             GameStatus
	Winner             *Player
	Result             GameResult
	Moves              []Move
	StartTime          time.Time
	EndTime            time.Time
	DisconnectTime     time.Time
	DisconnectedPlayer int
	Bot                *Bot
	mu                 sync.RWMutex
}

// NewGame creates a new game instance
func NewGame(player1Username string) *Game {
	return &Game{
		ID: uuid.New().String(),
		Player1: &Player{
			Username:    player1Username,
			PlayerNum:   Player1,
			IsBot:       false,
			IsConnected: true,
		},
		Board:       NewBoard(),
		CurrentTurn: Player1,
		Status:      StatusWaiting,
		Moves:       make([]Move, 0),
		StartTime:   time.Now(),
	}
}

// AddPlayer2 adds the second player to the game
func (g *Game) AddPlayer2(username string, isBot bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.Player2 = &Player{
		Username:    username,
		PlayerNum:   Player2,
		IsBot:       isBot,
		IsConnected: true,
	}
	g.Status = StatusPlaying

	if isBot {
		g.Bot = NewBot(Player2)
	}
}

// MakeMove makes a move for the specified player
func (g *Game) MakeMove(playerNum, column int) (int, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Status != StatusPlaying {
		return -1, ErrGameNotInProgress
	}

	if g.CurrentTurn != playerNum {
		return -1, ErrNotYourTurn
	}

	row, err := g.Board.DropDisc(column, playerNum)
	if err != nil {
		return -1, err
	}

	// Record the move
	g.Moves = append(g.Moves, Move{
		PlayerNum: playerNum,
		Column:    column,
		Row:       row,
		Timestamp: time.Now(),
	})

	// Check for win
	if g.Board.CheckWin(playerNum) {
		g.Status = StatusFinished
		g.EndTime = time.Now()
		if playerNum == Player1 {
			g.Winner = g.Player1
			g.Result = ResultWinPlayer1
		} else {
			g.Winner = g.Player2
			g.Result = ResultWinPlayer2
		}
		return row, nil
	}

	// Check for draw
	if g.Board.IsFull() {
		g.Status = StatusFinished
		g.EndTime = time.Now()
		g.Result = ResultDraw
		return row, nil
	}

	// Switch turns
	if g.CurrentTurn == Player1 {
		g.CurrentTurn = Player2
	} else {
		g.CurrentTurn = Player1
	}

	return row, nil
}

// MakeBotMove makes a move for the bot
func (g *Game) MakeBotMove() (int, int, error) {
	g.mu.Lock()
	if g.Bot == nil || g.CurrentTurn != Player2 {
		g.mu.Unlock()
		return -1, -1, ErrNotYourTurn
	}
	g.mu.Unlock()

	// Get best move from bot
	column := g.Bot.GetBestMove(g.Board)

	// Make the move
	row, err := g.MakeMove(Player2, column)
	return column, row, err
}

// PlayerDisconnected marks a player as disconnected
func (g *Game) PlayerDisconnected(playerNum int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Status != StatusPlaying {
		return
	}

	g.DisconnectedPlayer = playerNum
	g.DisconnectTime = time.Now()
	g.Status = StatusDisconnect

	if playerNum == Player1 {
		g.Player1.IsConnected = false
	} else {
		g.Player2.IsConnected = false
	}
}

// PlayerReconnected marks a player as reconnected
func (g *Game) PlayerReconnected(playerNum int) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Status != StatusDisconnect || g.DisconnectedPlayer != playerNum {
		return false
	}

	// Check if within 30-second window
	if time.Since(g.DisconnectTime) > 30*time.Second {
		return false
	}

	g.Status = StatusPlaying
	g.DisconnectedPlayer = 0
	g.DisconnectTime = time.Time{}

	if playerNum == Player1 {
		g.Player1.IsConnected = true
	} else {
		g.Player2.IsConnected = true
	}

	return true
}

// Forfeit ends the game with a forfeit
func (g *Game) Forfeit(loserPlayerNum int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.Status = StatusFinished
	g.EndTime = time.Now()
	g.Result = ResultForfeit

	if loserPlayerNum == Player1 {
		g.Winner = g.Player2
	} else {
		g.Winner = g.Player1
	}
}

// GetState returns the current game state for serialization
func (g *Game) GetState() *GameState {
	g.mu.RLock()
	defer g.mu.RUnlock()

	state := &GameState{
		ID:          g.ID,
		Board:       g.Board.ToSlice(),
		CurrentTurn: g.CurrentTurn,
		Status:      g.Status,
		MoveCount:   len(g.Moves),
	}

	if g.Player1 != nil {
		state.Player1 = g.Player1.Username
	}
	if g.Player2 != nil {
		state.Player2 = g.Player2.Username
		state.IsVsBot = g.Player2.IsBot
	}
	if g.Winner != nil {
		state.Winner = g.Winner.Username
	}
	if len(g.Moves) > 0 {
		lastMove := g.Moves[len(g.Moves)-1]
		state.LastMove = &MoveInfo{
			Column: lastMove.Column,
			Row:    lastMove.Row,
		}
	}
	if g.Result != "" {
		state.Result = string(g.Result)
	}

	return state
}

// GetPlayerByUsername returns the player number for a username
func (g *Game) GetPlayerByUsername(username string) int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.Player1 != nil && g.Player1.Username == username {
		return Player1
	}
	if g.Player2 != nil && g.Player2.Username == username {
		return Player2
	}
	return 0
}

// GetDuration returns the game duration in seconds
func (g *Game) GetDuration() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.EndTime.IsZero() {
		return int(time.Since(g.StartTime).Seconds())
	}
	return int(g.EndTime.Sub(g.StartTime).Seconds())
}

// GameState represents the serializable game state
type GameState struct {
	ID          string     `json:"id"`
	Player1     string     `json:"player1"`
	Player2     string     `json:"player2"`
	IsVsBot     bool       `json:"isVsBot"`
	Board       [][]int    `json:"board"`
	CurrentTurn int        `json:"currentTurn"`
	Status      GameStatus `json:"status"`
	Winner      string     `json:"winner,omitempty"`
	Result      string     `json:"result,omitempty"`
	LastMove    *MoveInfo  `json:"lastMove,omitempty"`
	MoveCount   int        `json:"moveCount"`
}

// MoveInfo represents info about a move
type MoveInfo struct {
	Column int `json:"column"`
	Row    int `json:"row"`
}

// Errors
var (
	ErrGameNotInProgress = &GameError{"game is not in progress"}
	ErrNotYourTurn       = &GameError{"not your turn"}
	ErrGameNotFound      = &GameError{"game not found"}
	ErrPlayerNotFound    = &GameError{"player not found"}
)

type GameError struct {
	msg string
}

func (e *GameError) Error() string {
	return e.msg
}
