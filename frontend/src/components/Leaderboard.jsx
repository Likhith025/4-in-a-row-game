import { useState, useEffect } from 'react';

const BACKEND_URL = import.meta.env.VITE_BACKEND_URL || 'localhost:8080';
const API_URL = `${import.meta.env.PROD ? 'https' : 'http'}://${BACKEND_URL}/api`;

function Leaderboard() {
    const [entries, setEntries] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        fetchLeaderboard();
    }, []);

    const fetchLeaderboard = async () => {
        try {
            setLoading(true);
            const response = await fetch(`${API_URL}/leaderboard`);
            if (!response.ok) {
                throw new Error('Failed to fetch leaderboard');
            }
            const data = await response.json();
            setEntries(data || []);
            setError(null);
        } catch (err) {
            console.error('Leaderboard error:', err);
            setError('Could not load leaderboard. Server may be starting up.');
            setEntries([]);
        } finally {
            setLoading(false);
        }
    };

    const getRankClass = (rank) => {
        if (rank === 1) return 'rank gold';
        if (rank === 2) return 'rank silver';
        if (rank === 3) return 'rank bronze';
        return 'rank';
    };

    const getRankEmoji = (rank) => {
        if (rank === 1) return '🥇';
        if (rank === 2) return '🥈';
        if (rank === 3) return '🥉';
        return `#${rank}`;
    };

    if (loading) {
        return (
            <div className="card leaderboard">
                <h2>🏆 Leaderboard</h2>
                <div style={{ textAlign: 'center', padding: '40px' }}>
                    <div className="spinner"></div>
                    <p style={{ marginTop: '15px', color: 'var(--text-secondary)' }}>Loading...</p>
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="card leaderboard">
                <h2>🏆 Leaderboard</h2>
                <div style={{ textAlign: 'center', padding: '40px' }}>
                    <p style={{ color: 'var(--text-secondary)' }}>{error}</p>
                    <button
                        className="btn btn-secondary"
                        style={{ marginTop: '15px' }}
                        onClick={fetchLeaderboard}
                    >
                        Retry
                    </button>
                </div>
            </div>
        );
    }

    if (entries.length === 0) {
        return (
            <div className="card leaderboard">
                <h2>🏆 Leaderboard</h2>
                <div style={{ textAlign: 'center', padding: '40px' }}>
                    <p style={{ color: 'var(--text-secondary)', fontSize: '3rem', marginBottom: '15px' }}>
                        🎮
                    </p>
                    <p style={{ color: 'var(--text-secondary)' }}>
                        No games played yet. Be the first champion!
                    </p>
                </div>
            </div>
        );
    }

    return (
        <div className="card leaderboard">
            <h2>🏆 Leaderboard</h2>
            <table className="leaderboard-table">
                <thead>
                    <tr>
                        <th>Rank</th>
                        <th>Player</th>
                        <th>Wins</th>
                        <th>Games</th>
                        <th>Win Rate</th>
                    </tr>
                </thead>
                <tbody>
                    {entries.map((entry) => (
                        <tr key={entry.username}>
                            <td className={getRankClass(entry.rank)}>
                                {getRankEmoji(entry.rank)}
                            </td>
                            <td>{entry.username}</td>
                            <td className="wins">{entry.wins}</td>
                            <td>{entry.games}</td>
                            <td className="win-rate">{entry.winRate}%</td>
                        </tr>
                    ))}
                </tbody>
            </table>
            <div style={{ marginTop: '20px', textAlign: 'center' }}>
                <button className="btn btn-secondary" onClick={fetchLeaderboard}>
                    Refresh
                </button>
            </div>
        </div>
    );
}

export default Leaderboard;
