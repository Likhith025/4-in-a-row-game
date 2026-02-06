# Connect Four - Real-Time Multiplayer Game

A full-stack real-time multiplayer Connect Four (4 in a Row) game built with **GoLang** backend and **React** frontend.

![Connect Four](https://img.shields.io/badge/Game-Connect%20Four-blue)
![Go](https://img.shields.io/badge/Backend-Go%201.21-00ADD8?logo=go)
![React](https://img.shields.io/badge/Frontend-React%2018-61DAFB?logo=react)
![WebSocket](https://img.shields.io/badge/Realtime-WebSocket-green)

## 🎮 Features

### Core Gameplay
- **Real-time multiplayer** via WebSocket
- **7×6 game board** with smooth animations
- **Win detection** for horizontal, vertical, and diagonal connections
- **Draw detection** when board is full

### Matchmaking
- **10-second matchmaking timeout** - if no opponent joins, a bot starts
- **Competitive AI bot** using Minimax algorithm with alpha-beta pruning
- The bot strategically blocks opponent wins and creates winning opportunities

### Reconnection
- **30-second reconnection window** - disconnect and rejoin your game
- Automatic forfeit if player doesn't reconnect in time

### Persistence & Analytics
- **PostgreSQL** for game history and leaderboard
- **Kafka integration** for real-time game analytics (optional)
- **Leaderboard** showing top players by wins

## 🚀 Quick Start

### Prerequisites

- [Go 1.21+](https://golang.org/dl/) - **Required for backend**
- [Node.js 18+](https://nodejs.org/) - Required for frontend

### Installing Go (Windows)

1. Download Go from https://golang.org/dl/ (e.g., `go1.21.6.windows-amd64.msi`)
2. Run the installer
3. **Restart your terminal/PowerShell** after installation
4. Verify installation: `go version`

### 1. Configure Database

The backend is configured to use a PostgreSQL database. The `.env` file in the `backend` folder contains the database connection string.

### 2. Start Backend

```powershell
# Navigate to backend
cd backend

# Download dependencies
go mod download

# Run the server
go run cmd/server/main.go
```

The server starts on `http://localhost:8080`

### 3. Start Frontend

```powershell
# In a new terminal, navigate to frontend
cd frontend

# Install dependencies (if not done)
npm install

# Start development server
npm run dev
```

The frontend starts on `http://localhost:3000`

## 🎯 How to Play

1. **Enter your username** in the lobby
2. **Wait for an opponent** (or bot after 10 seconds)
3. **Click a column** to drop your disc
4. **Connect 4 discs** horizontally, vertically, or diagonally to win!

## 🏗️ Project Structure

```
├── backend/
│   ├── cmd/server/main.go          # Entry point
│   └── internal/
│       ├── game/
│       │   ├── board.go            # Game board & win detection
│       │   ├── bot.go              # AI bot (Minimax)
│       │   └── game.go             # Game state management
│       ├── matchmaker/             # Player matching
│       ├── websocket/              # Real-time communication
│       ├── storage/                # PostgreSQL persistence
│       ├── kafka/                  # Analytics events
│       └── api/                    # REST endpoints
│
├── frontend/
│   └── src/
│       ├── components/
│       │   ├── App.jsx             # Main component
│       │   ├── Board.jsx           # Game grid
│       │   ├── Lobby.jsx           # Username entry
│       │   └── Leaderboard.jsx     # Rankings
│       └── hooks/
│           └── useWebSocket.js     # WebSocket hook
│
└── docker-compose.yml              # Infrastructure
```

## 🔌 API Endpoints

### REST API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/leaderboard` | GET | Top players by wins |
| `/api/stats/:username` | GET | Player statistics |
| `/api/analytics` | GET | Game analytics |
| `/api/status` | GET | Server status |
| `/health` | GET | Health check |

### WebSocket

Connect to `ws://localhost:8080/ws?username=<username>`

**Client → Server Messages:**
```json
{"type": "join"}
{"type": "move", "column": 3}
{"type": "reconnect", "gameId": "uuid"}
```

**Server → Client Messages:**
```json
{"type": "waiting", "message": "Looking for opponent..."}
{"type": "matched", "opponent": "player2", "gameId": "uuid", "yourTurn": true}
{"type": "state", "board": [[...]], "currentTurn": 1}
{"type": "gameOver", "winner": "player1", "reason": "connect4"}
```

## 🤖 Bot Strategy

The AI bot uses **Minimax with Alpha-Beta Pruning**:

1. **Immediate Win**: Takes winning move if available
2. **Block Opponent**: Blocks opponent's winning move
3. **Strategic Evaluation**:
   - Center column preference
   - Connected piece scoring
   - Threat creation
4. **5-move lookahead** for optimal decisions

## 📊 Kafka Analytics (Bonus)

When Kafka is available, the system emits events for:

- `game_start` - New game created
- `move` - Player/bot move
- `game_end` - Game finished with result

The consumer service aggregates:
- Average game duration
- Most frequent winners
- Games per hour/day
- Per-player statistics

Access Kafka UI at `http://localhost:8081` to monitor events.

## 🧪 Testing

### Backend Tests
```bash
cd backend
go test ./...
```

### Manual Testing
1. Open two browser tabs
2. Join with different usernames
3. Play a game!

### Bot Testing
1. Enter username and wait 10+ seconds
2. Bot will be assigned automatically

## 📝 License

MIT License - feel free to use this project for learning or as a base for your own games!

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
