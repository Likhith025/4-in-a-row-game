package game

import (
	"math"
	"math/rand"
)

// Bot represents the AI player
type Bot struct {
	player   int
	opponent int
	maxDepth int
}

// NewBot creates a new bot instance
func NewBot(player int) *Bot {
	opponent := Player1
	if player == Player1 {
		opponent = Player2
	}
	return &Bot{
		player:   player,
		opponent: opponent,
		maxDepth: 5, // Search depth for minimax
	}
}

// GetBestMove returns the best column to play using minimax with alpha-beta pruning
func (bot *Bot) GetBestMove(board *Board) int {
	// Clone the board for calculations
	b := board.Clone()

	// First, check for immediate winning move
	validCols := b.getValidColumnsUnsafe()
	for _, col := range validCols {
		b.DropDiscUnsafe(col, bot.player)
		if b.checkWinUnsafe(bot.player) {
			b.UndoMove(col)
			return col
		}
		b.UndoMove(col)
	}

	// Check for blocking opponent's winning move
	for _, col := range validCols {
		b.DropDiscUnsafe(col, bot.opponent)
		if b.checkWinUnsafe(bot.opponent) {
			b.UndoMove(col)
			return col // Block the winning move
		}
		b.UndoMove(col)
	}

	// Use minimax for strategic move
	bestScore := math.MinInt32
	bestCol := validCols[0]

	// Prefer center column for tie-breaking
	columnOrder := []int{3, 2, 4, 1, 5, 0, 6}
	orderedCols := make([]int, 0, len(validCols))
	for _, col := range columnOrder {
		for _, validCol := range validCols {
			if col == validCol {
				orderedCols = append(orderedCols, col)
				break
			}
		}
	}

	for _, col := range orderedCols {
		b.DropDiscUnsafe(col, bot.player)
		score := bot.minimax(b, bot.maxDepth-1, math.MinInt32, math.MaxInt32, false)
		b.UndoMove(col)

		if score > bestScore {
			bestScore = score
			bestCol = col
		}
	}

	return bestCol
}

// minimax implements the minimax algorithm with alpha-beta pruning
func (bot *Bot) minimax(board *Board, depth int, alpha, beta int, isMaximizing bool) int {
	// Terminal conditions
	if board.checkWinUnsafe(bot.player) {
		return 10000 + depth // Prefer winning sooner
	}
	if board.checkWinUnsafe(bot.opponent) {
		return -10000 - depth // Prefer losing later
	}
	if board.isFullUnsafe() || depth == 0 {
		return bot.evaluateBoard(board)
	}

	validCols := board.getValidColumnsUnsafe()
	if len(validCols) == 0 {
		return 0
	}

	if isMaximizing {
		maxScore := math.MinInt32
		for _, col := range validCols {
			board.DropDiscUnsafe(col, bot.player)
			score := bot.minimax(board, depth-1, alpha, beta, false)
			board.UndoMove(col)

			maxScore = max(maxScore, score)
			alpha = max(alpha, score)
			if beta <= alpha {
				break // Alpha-beta pruning
			}
		}
		return maxScore
	} else {
		minScore := math.MaxInt32
		for _, col := range validCols {
			board.DropDiscUnsafe(col, bot.opponent)
			score := bot.minimax(board, depth-1, alpha, beta, true)
			board.UndoMove(col)

			minScore = min(minScore, score)
			beta = min(beta, score)
			if beta <= alpha {
				break // Alpha-beta pruning
			}
		}
		return minScore
	}
}

// evaluateBoard scores the current board position for the bot
func (bot *Bot) evaluateBoard(board *Board) int {
	score := 0

	// Score center column (strategic advantage)
	centerCol := Columns / 2
	centerCount := 0
	for row := 0; row < Rows; row++ {
		if board.cells[row][centerCol] == bot.player {
			centerCount++
		}
	}
	score += centerCount * 3

	// Score all windows of 4
	score += bot.scoreAllWindows(board)

	return score
}

// scoreAllWindows evaluates all possible 4-cell windows
func (bot *Bot) scoreAllWindows(board *Board) int {
	score := 0

	// Horizontal windows
	for row := 0; row < Rows; row++ {
		for col := 0; col <= Columns-4; col++ {
			window := [4]int{
				board.cells[row][col],
				board.cells[row][col+1],
				board.cells[row][col+2],
				board.cells[row][col+3],
			}
			score += bot.scoreWindow(window)
		}
	}

	// Vertical windows
	for row := 0; row <= Rows-4; row++ {
		for col := 0; col < Columns; col++ {
			window := [4]int{
				board.cells[row][col],
				board.cells[row+1][col],
				board.cells[row+2][col],
				board.cells[row+3][col],
			}
			score += bot.scoreWindow(window)
		}
	}

	// Diagonal (down-right) windows
	for row := 0; row <= Rows-4; row++ {
		for col := 0; col <= Columns-4; col++ {
			window := [4]int{
				board.cells[row][col],
				board.cells[row+1][col+1],
				board.cells[row+2][col+2],
				board.cells[row+3][col+3],
			}
			score += bot.scoreWindow(window)
		}
	}

	// Diagonal (up-right) windows
	for row := 3; row < Rows; row++ {
		for col := 0; col <= Columns-4; col++ {
			window := [4]int{
				board.cells[row][col],
				board.cells[row-1][col+1],
				board.cells[row-2][col+2],
				board.cells[row-3][col+3],
			}
			score += bot.scoreWindow(window)
		}
	}

	return score
}

// scoreWindow evaluates a window of 4 cells
func (bot *Bot) scoreWindow(window [4]int) int {
	botCount := 0
	oppCount := 0
	emptyCount := 0

	for _, cell := range window {
		switch cell {
		case bot.player:
			botCount++
		case bot.opponent:
			oppCount++
		case Empty:
			emptyCount++
		}
	}

	// Scoring heuristics
	if botCount == 4 {
		return 100
	}
	if botCount == 3 && emptyCount == 1 {
		return 5
	}
	if botCount == 2 && emptyCount == 2 {
		return 2
	}

	// Penalize opponent threats
	if oppCount == 3 && emptyCount == 1 {
		return -4
	}

	return 0
}

// GetRandomValidMove returns a random valid column (fallback)
func GetRandomValidMove(board *Board) int {
	validCols := board.GetValidColumns()
	if len(validCols) == 0 {
		return -1
	}
	return validCols[rand.Intn(len(validCols))]
}
