import { useState } from 'react';

function Lobby({ onJoin }) {
    const [username, setUsername] = useState('');
    const [error, setError] = useState('');

    const handleSubmit = (e) => {
        e.preventDefault();

        const trimmedUsername = username.trim();

        if (!trimmedUsername) {
            setError('Please enter a username');
            return;
        }

        if (trimmedUsername.length < 2) {
            setError('Username must be at least 2 characters');
            return;
        }

        if (trimmedUsername.length > 20) {
            setError('Username must be less than 20 characters');
            return;
        }

        if (!/^[a-zA-Z0-9_]+$/.test(trimmedUsername)) {
            setError('Username can only contain letters, numbers, and underscores');
            return;
        }

        setError('');
        onJoin(trimmedUsername);
    };

    return (
        <div className="card lobby">
            <h2>Enter the Arena</h2>
            <form onSubmit={handleSubmit}>
                <div className="input-group">
                    <input
                        type="text"
                        placeholder="Enter your username"
                        value={username}
                        onChange={(e) => {
                            setUsername(e.target.value);
                            setError('');
                        }}
                        autoFocus
                    />
                    <button type="submit" className="btn btn-primary">
                        Find Game
                    </button>
                </div>
                {error && (
                    <p style={{ color: 'var(--error)', marginTop: '10px', fontSize: '0.9rem' }}>
                        {error}
                    </p>
                )}
            </form>
            <div style={{ marginTop: '30px', color: 'var(--text-secondary)', fontSize: '0.9rem' }}>
                <p>🎮 Play against other players in real-time</p>
                <p style={{ marginTop: '8px' }}>🤖 Bot will join if no opponent found in 10 seconds</p>
            </div>
        </div>
    );
}

export default Lobby;
