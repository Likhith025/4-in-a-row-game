import { useState, useCallback, useRef, useEffect } from 'react';

// Use environment variable for backend URL, fallback to localhost for development
const BACKEND_URL = import.meta.env.VITE_BACKEND_URL || 'localhost:8080';
const WS_PROTOCOL = import.meta.env.PROD ? 'wss' : 'ws';
const WS_URL = `${WS_PROTOCOL}://${BACKEND_URL}/ws`;

export function useWebSocket() {
    const [isConnected, setIsConnected] = useState(false);
    const [lastMessage, setLastMessage] = useState(null);
    const wsRef = useRef(null);
    const usernameRef = useRef(null);
    const reconnectTimeoutRef = useRef(null);
    const messageHandlersRef = useRef([]);

    const connect = useCallback((username) => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
            return;
        }

        usernameRef.current = username;

        const ws = new WebSocket(`${WS_URL}?username=${encodeURIComponent(username)}`);

        ws.onopen = () => {
            console.log('WebSocket connected');
            setIsConnected(true);
        };

        ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                setLastMessage(data);
                messageHandlersRef.current.forEach(handler => handler(data));
            } catch (e) {
                console.error('Failed to parse message:', e);
            }
        };

        ws.onclose = () => {
            console.log('WebSocket disconnected');
            setIsConnected(false);

            // Auto-reconnect if we have a username
            if (usernameRef.current) {
                reconnectTimeoutRef.current = setTimeout(() => {
                    console.log('Attempting to reconnect...');
                    connect(usernameRef.current);
                }, 2000);
            }
        };

        ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        wsRef.current = ws;
    }, []);

    const disconnect = useCallback(() => {
        if (reconnectTimeoutRef.current) {
            clearTimeout(reconnectTimeoutRef.current);
        }
        usernameRef.current = null;
        if (wsRef.current) {
            wsRef.current.close();
            wsRef.current = null;
        }
        setIsConnected(false);
    }, []);

    const sendMessage = useCallback((message) => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify(message));
        } else {
            console.error('WebSocket not connected');
        }
    }, []);

    const addMessageHandler = useCallback((handler) => {
        messageHandlersRef.current.push(handler);
        return () => {
            messageHandlersRef.current = messageHandlersRef.current.filter(h => h !== handler);
        };
    }, []);

    const joinGame = useCallback(() => {
        sendMessage({ type: 'join' });
    }, [sendMessage]);

    const makeMove = useCallback((column) => {
        sendMessage({ type: 'move', column });
    }, [sendMessage]);

    const reconnectToGame = useCallback((gameId) => {
        sendMessage({ type: 'reconnect', gameId });
    }, [sendMessage]);

    useEffect(() => {
        return () => {
            if (reconnectTimeoutRef.current) {
                clearTimeout(reconnectTimeoutRef.current);
            }
            if (wsRef.current) {
                wsRef.current.close();
            }
        };
    }, []);

    return {
        isConnected,
        lastMessage,
        connect,
        disconnect,
        sendMessage,
        addMessageHandler,
        joinGame,
        makeMove,
        reconnectToGame
    };
}
