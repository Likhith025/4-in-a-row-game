package game

import (
	"errors"
	"sync"
)

const (
	Rows    = 6
	Columns = 7
)

const (
	Empty   = 0
	Player1 = 1
	Player2 = 2
)

// Board represents the game board
type Board struct {
	cells [Rows][Columns]int
	mu    sync.RWMutex
}

// NewBoard creates a new empty board
func NewBoard() *Board {
	return &Board{}
}

// Clone creates a deep copy of the board
func (b *Board) Clone() *Board {
	b.mu.RLock()
	defer b.mu.RUnlock()

	newBoard := &Board{}
	for i := 0; i < Rows; i++ {
		for j := 0; j < Columns; j++ {
			newBoard.cells[i][j] = b.cells[i][j]
		}
	}
	return newBoard
}

// GetCell returns the value at a specific position
func (b *Board) GetCell(row, col int) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.cells[row][col]
}

// GetCells returns a copy of the board cells
func (b *Board) GetCells() [Rows][Columns]int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var cells [Rows][Columns]int
	for i := 0; i < Rows; i++ {
		for j := 0; j < Columns; j++ {
			cells[i][j] = b.cells[i][j]
		}
	}
	return cells
}

// DropDisc drops a disc into the specified column for the given player
// Returns the row where the disc landed, or error if column is full/invalid
func (b *Board) DropDisc(column, player int) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if column < 0 || column >= Columns {
		return -1, errors.New("invalid column")
	}

	if player != Player1 && player != Player2 {
		return -1, errors.New("invalid player")
	}

	// Find the lowest empty row in the column
	for row := Rows - 1; row >= 0; row-- {
		if b.cells[row][column] == Empty {
			b.cells[row][column] = player
			return row, nil
		}
	}

	return -1, errors.New("column is full")
}

// DropDiscUnsafe is like DropDisc but without locking (for bot calculations)
func (b *Board) DropDiscUnsafe(column, player int) (int, error) {
	if column < 0 || column >= Columns {
		return -1, errors.New("invalid column")
	}

	for row := Rows - 1; row >= 0; row-- {
		if b.cells[row][column] == Empty {
			b.cells[row][column] = player
			return row, nil
		}
	}

	return -1, errors.New("column is full")
}

// UndoMove removes the top disc from a column (for bot calculations)
func (b *Board) UndoMove(column int) {
	for row := 0; row < Rows; row++ {
		if b.cells[row][column] != Empty {
			b.cells[row][column] = Empty
			return
		}
	}
}

// CheckWin checks if the specified player has won
func (b *Board) CheckWin(player int) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.checkWinUnsafe(player)
}

// checkWinUnsafe checks win without locking
func (b *Board) checkWinUnsafe(player int) bool {
	// Check horizontal
	for row := 0; row < Rows; row++ {
		for col := 0; col <= Columns-4; col++ {
			if b.cells[row][col] == player &&
				b.cells[row][col+1] == player &&
				b.cells[row][col+2] == player &&
				b.cells[row][col+3] == player {
				return true
			}
		}
	}

	// Check vertical
	for row := 0; row <= Rows-4; row++ {
		for col := 0; col < Columns; col++ {
			if b.cells[row][col] == player &&
				b.cells[row+1][col] == player &&
				b.cells[row+2][col] == player &&
				b.cells[row+3][col] == player {
				return true
			}
		}
	}

	// Check diagonal (down-right)
	for row := 0; row <= Rows-4; row++ {
		for col := 0; col <= Columns-4; col++ {
			if b.cells[row][col] == player &&
				b.cells[row+1][col+1] == player &&
				b.cells[row+2][col+2] == player &&
				b.cells[row+3][col+3] == player {
				return true
			}
		}
	}

	// Check diagonal (up-right)
	for row := 3; row < Rows; row++ {
		for col := 0; col <= Columns-4; col++ {
			if b.cells[row][col] == player &&
				b.cells[row-1][col+1] == player &&
				b.cells[row-2][col+2] == player &&
				b.cells[row-3][col+3] == player {
				return true
			}
		}
	}

	return false
}

// IsFull checks if the board is completely full (draw condition)
func (b *Board) IsFull() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.isFullUnsafe()
}

// isFullUnsafe checks if board is full without locking
func (b *Board) isFullUnsafe() bool {
	for col := 0; col < Columns; col++ {
		if b.cells[0][col] == Empty {
			return false
		}
	}
	return true
}

// GetValidColumns returns a list of columns that can accept a disc
func (b *Board) GetValidColumns() []int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.getValidColumnsUnsafe()
}

// getValidColumnsUnsafe returns valid columns without locking
func (b *Board) getValidColumnsUnsafe() []int {
	validCols := make([]int, 0, Columns)
	for col := 0; col < Columns; col++ {
		if b.cells[0][col] == Empty {
			validCols = append(validCols, col)
		}
	}
	return validCols
}

// IsColumnValid checks if a column can accept a disc
func (b *Board) IsColumnValid(column int) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return column >= 0 && column < Columns && b.cells[0][column] == Empty
}

// ToSlice converts the board to a 2D slice for JSON serialization
func (b *Board) ToSlice() [][]int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([][]int, Rows)
	for i := 0; i < Rows; i++ {
		result[i] = make([]int, Columns)
		for j := 0; j < Columns; j++ {
			result[i][j] = b.cells[i][j]
		}
	}
	return result
}
