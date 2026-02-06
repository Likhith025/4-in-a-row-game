import { useState, useEffect, useCallback } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';
import Lobby from './Lobby';
import Board from './Board';
import Leaderboard from './Leaderboard';

const GAME_STATES = {
    LOBBY: 'lobby',
    WAITING: 'waiting',
    PLAYING: 'playing',
    FINISHED: 'finished'
};

function App() {
    const [view, setView] = useState('game'); // 'game' or 'leaderboard'
    const [gameState, setGameState] = useState(GAME_STATES.LOBBY);
    const [username, setUsername] = useState('');
    const [gameData, setGameData] = useState(null);
    const [board, setBoard] = useState(Array(6).fill(null).map(() => Array(7).fill(0)));
    const [currentTurn, setCurrentTurn] = useState(1);
    const [playerNum, setPlayerNum] = useState(null);
    const [opponent, setOpponent] = useState('');
    const [gameId, setGameId] = useState('');
    const [winner, setWinner] = useState(null);
    const [result, setResult] = useState('');
    const [waitingTime, setWaitingTime] = useState(0);
    const [opponentDisconnected, setOpponentDisconnected] = useState(false);
    const [reconnectDeadline, setReconnectDeadline] = useState(null);

    const {
        isConnected,
        connect,
        disconnect,
        addMessageHandler,
        joinGame,
        makeMove
    } = useWebSocket();

    // Handle WebSocket messages
    useEffect(() => {
        const unsubscribe = addMessageHandler((message) => {
            console.log('Received message:', message);

            switch (message.type) {
                case 'waiting':
                    setGameState(GAME_STATES.WAITING);
                    setWaitingTime(0);
                    break;

                case 'matched':
                    setGameState(GAME_STATES.PLAYING);
                    setOpponent(message.opponent);
                    setGameId(message.gameId);
                    setPlayerNum(message.playerNum);
                    if (message.state) {
                        setBoard(message.state.board);
                        setCurrentTurn(message.state.currentTurn);
                    }
                    setOpponentDisconnected(false);
                    break;

                case 'state':
                    if (message.state) {
                        setBoard(message.state.board);
                        setCurrentTurn(message.state.currentTurn);
                    }
                    break;

                case 'gameOver':
                    setGameState(GAME_STATES.FINISHED);
                    setWinner(message.winner);
                    setResult(message.reason);
                    break;

                case 'opponentDisconnected':
                    setOpponentDisconnected(true);
                    setReconnectDeadline(message.reconnectDeadline);
                    break;

                case 'opponentReconnected':
                    setOpponentDisconnected(false);
                    setReconnectDeadline(null);
                    break;

                case 'error':
                    console.error('Game error:', message.message);
                    break;
            }
        });

        return unsubscribe;
    }, [addMessageHandler]);

    // Waiting timer
    useEffect(() => {
        let interval;
        if (gameState === GAME_STATES.WAITING) {
            interval = setInterval(() => {
                setWaitingTime(prev => prev + 1);
            }, 1000);
        }
        return () => clearInterval(interval);
    }, [gameState]);

    // State to track if we should join once connected
    const [pendingJoin, setPendingJoin] = useState(false);

    // Effect to join game once WebSocket is connected
    useEffect(() => {
        if (pendingJoin && isConnected) {
            joinGame();
            setPendingJoin(false);
        }
    }, [pendingJoin, isConnected, joinGame]);

    const handleJoinGame = useCallback((enteredUsername) => {
        setUsername(enteredUsername);
        connect(enteredUsername);
        setPendingJoin(true);  // Will trigger join once connected
    }, [connect]);

    const handleColumnClick = useCallback((column) => {
        if (gameState !== GAME_STATES.PLAYING) return;
        if (currentTurn !== playerNum) return;
        if (board[0][column] !== 0) return; // Column is full

        makeMove(column);
    }, [gameState, currentTurn, playerNum, board, makeMove]);

    const handlePlayAgain = useCallback(() => {
        setGameState(GAME_STATES.LOBBY);
        setBoard(Array(6).fill(null).map(() => Array(7).fill(0)));
        setWinner(null);
        setResult('');
        setOpponent('');
        setGameId('');
        setPlayerNum(null);
        disconnect();
    }, [disconnect]);

    const handleNewGame = useCallback(() => {
        setBoard(Array(6).fill(null).map(() => Array(7).fill(0)));
        setWinner(null);
        setResult('');
        setGameState(GAME_STATES.WAITING);
        joinGame();
    }, [joinGame]);

    const isMyTurn = currentTurn === playerNum;
    const didIWin = winner === username;
    const isDraw = result === 'draw';

    return (
        <div className="container">
            <header className="header">
                <h1>Connect Four</h1>
                <p>Real-time multiplayer game</p>
            </header>

            <nav className="nav">
                <button
                    className={`nav-btn ${view === 'game' ? 'active' : ''}`}
                    onClick={() => setView('game')}
                >
                    Play
                </button>
                <button
                    className={`nav-btn ${view === 'leaderboard' ? 'active' : ''}`}
                    onClick={() => setView('leaderboard')}
                >
                    Leaderboard
                </button>
            </nav>

            {view === 'leaderboard' ? (
                <Leaderboard />
            ) : (
                <>
                    {gameState === GAME_STATES.LOBBY && (
                        <Lobby onJoin={handleJoinGame} />
                    )}

                    {gameState === GAME_STATES.WAITING && (
                        <div className="card waiting">
                            <h2>Finding Opponent...</h2>
                            <div className="spinner"></div>
                            <div className="countdown">{waitingTime}s</div>
                            <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem' }}>
                                {waitingTime >= 8
                                    ? 'Bot will be assigned soon...'
                                    : 'Waiting for another player to join'}
                            </p>
                        </div>
                    )}

                    {(gameState === GAME_STATES.PLAYING || gameState === GAME_STATES.FINISHED) && (
                        <div className="game-container">
                            <div className="game-info">
                                <div className="player-info">
                                    <div className="player-disc player1"></div>
                                    <span className={`player-name ${playerNum === 1 ? 'current' : ''}`}>
                                        {playerNum === 1 ? `${username} (You)` : opponent}
                                    </span>
                                </div>
                                <span className="vs">VS</span>
                                <div className="player-info">
                                    <div className="player-disc player2"></div>
                                    <span className={`player-name ${playerNum === 2 ? 'current' : ''}`}>
                                        {playerNum === 2 ? `${username} (You)` : opponent}
                                    </span>
                                </div>
                            </div>

                            {opponentDisconnected && gameState === GAME_STATES.PLAYING && (
                                <div className="disconnect-notice">
                                    <p>⚠️ Opponent disconnected. Waiting for reconnection...</p>
                                </div>
                            )}

                            {gameState === GAME_STATES.PLAYING && (
                                <div className={`turn-indicator ${isMyTurn ? 'your-turn' : ''}`}>
                                    {isMyTurn ? "Your Turn - Click a column!" : `${opponent}'s Turn...`}
                                </div>
                            )}

                            <Board
                                board={board}
                                onColumnClick={handleColumnClick}
                                disabled={gameState !== GAME_STATES.PLAYING || !isMyTurn}
                                currentPlayer={currentTurn}
                            />

                            {gameState === GAME_STATES.FINISHED && (
                                <div className="modal-overlay">
                                    <div className="modal-content">
                                        <h2 className={isDraw ? 'draw' : didIWin ? 'win' : 'lose'}>
                                            {isDraw ? "It's a Draw!" : didIWin ? 'You Won! 🎉' : 'You Lost 😔'}
                                        </h2>
                                        <p>
                                            {result === 'forfeit'
                                                ? 'Opponent forfeited the game'
                                                : isDraw
                                                    ? 'The board is full'
                                                    : `${winner} connected 4 in a row!`}
                                        </p>
                                        <div className="modal-buttons">
                                            <button className="btn btn-primary" onClick={handleNewGame}>
                                                Play Again
                                            </button>
                                            <button className="btn btn-secondary" onClick={handlePlayAgain}>
                                                Back to Lobby
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            )}

                            <div className="status-bar">
                                <div style={{ display: 'flex', alignItems: 'center' }}>
                                    <div className={`status-dot ${isConnected ? '' : 'disconnected'}`}></div>
                                    <span>{isConnected ? 'Connected' : 'Disconnected'}</span>
                                </div>
                                <span>Game ID: {gameId.slice(0, 8)}...</span>
                            </div>
                        </div>
                    )}
                </>
            )}
        </div>
    );
}

export default App;
