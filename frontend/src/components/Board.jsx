import { memo } from 'react';
import Cell from './Cell';

const Board = memo(function Board({ board, onColumnClick, disabled, currentPlayer }) {
    // Board is stored as [row][col] with row 0 at top
    // We need to render columns for click handling

    const columns = [];
    for (let col = 0; col < 7; col++) {
        const cells = [];
        for (let row = 0; row < 6; row++) {
            cells.push(
                <Cell
                    key={`${row}-${col}`}
                    value={board[row][col]}
                />
            );
        }

        const isColumnFull = board[0][col] !== 0;

        columns.push(
            <div
                key={col}
                className={`column ${disabled || isColumnFull ? 'disabled' : ''}`}
                onClick={() => !disabled && !isColumnFull && onColumnClick(col)}
            >
                <div className="column-indicator"></div>
                {cells}
            </div>
        );
    }

    return (
        <div className="board-wrapper">
            <div className="board">
                {columns}
            </div>
        </div>
    );
});

export default Board;
