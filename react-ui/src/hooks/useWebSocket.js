import { useState, useEffect, useRef } from 'react';

// Хук для работы с WebSocket с авто-переподключением (п.19.3 ТЗ)
export function useWebSocket(url) {
    const [data, setData] = useState(null);
    const [status, setStatus] = useState('connecting');
    const wsRef = useRef(null);
    const reconnectTimeoutRef = useRef(null);

    const connect = () => {
        setStatus('connecting');
        
        // Формируем полный URL
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}${url}`;
        
        wsRef.current = new WebSocket(wsUrl);

        wsRef.current.onopen = () => {
            setStatus('connected');
            if (reconnectTimeoutRef.current) {
                clearTimeout(reconnectTimeoutRef.current);
            }
        };

        wsRef.current.onmessage = (event) => {
            try {
                const parsed = JSON.parse(event.data);
                setData(parsed);
            } catch (e) {
                console.error("Ошибка парсинга WS:", e);
            }
        };

        wsRef.current.onclose = () => {
            setStatus('disconnected');
            // Авто переподключение при сбоях (п.19.3 ТЗ)
            reconnectTimeoutRef.current = setTimeout(connect, 3000);
        };

        wsRef.current.onerror = () => {
            wsRef.current.close();
        };
    };

    useEffect(() => {
        connect();
        return () => {
            if (wsRef.current) {
                wsRef.current.close();
            }
            if (reconnectTimeoutRef.current) {
                clearTimeout(reconnectTimeoutRef.current);
            }
        };
    }, [url]);

    return { data, status };
}
