import React, { useState, useEffect, useRef } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';

export function Logs() {
    const { data: incomingLog, status } = useWebSocket('/ws/logs');
    const [logs, setLogs] = useState([]);
    const [filter, setFilter] = useState(() => localStorage.getItem('venera-log-filter') || '');
    const [maxLines, setMaxLines] = useState(500);
    const searchInputRef = useRef(null);

    // Добавление новых логов
    useEffect(() => {
        if (incomingLog) {
            setLogs(prev => {
                const newLogs = [...prev, incomingLog];
                if (newLogs.length > maxLines) {
                    return newLogs.slice(newLogs.length - maxLines);
                }
                return newLogs;
            });
        }
    }, [incomingLog, maxLines]);

    // Сохранение фильтра
    useEffect(() => {
        localStorage.setItem('venera-log-filter', filter);
    }, [filter]);

    // Горячие клавиши (Ctrl+F, Esc)
    useEffect(() => {
        const handleKeyDown = (e) => {
            if (e.ctrlKey && e.key === 'f') {
                e.preventDefault();
                searchInputRef.current?.focus();
            }
            if (e.key === 'Escape') {
                setFilter('');
                searchInputRef.current?.blur();
            }
        };
        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    }, []);

    const filteredLogs = logs.filter(log => log.toLowerCase().includes(filter.toLowerCase()));

    const getLogColor = (text) => {
        const lower = text.toLowerCase();
        if (lower.includes('level=error') || lower.includes('level=fatal') || lower.includes('level=panic')) return '#ff4c4c';
        if (lower.includes('level=warning')) return '#ffc107';
        if (lower.includes('level=info')) return '#17a2b8'; // Или оставить белым, если тема темная
        return 'inherit';
    };

    return (
        <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
            <h2>Системный Журнал (Логи)</h2>
            <div className="card" style={{ display: 'flex', gap: '15px', alignItems: 'center' }}>
                <input 
                    ref={searchInputRef}
                    type="text" 
                    placeholder="Поиск по логам (Ctrl+F, очистить Esc)..." 
                    value={filter} 
                    onChange={e => setFilter(e.target.value)} 
                    style={{ flex: 1, padding: '8px' }}
                />
                <label>
                    Лимит строк: 
                    <select value={maxLines} onChange={e => setMaxLines(Number(e.target.value))} style={{ marginLeft: '10px' }}>
                        <option value={100}>100</option>
                        <option value={500}>500</option>
                        <option value={1000}>1000</option>
                    </select>
                </label>
                <span className="status-badge" style={{ backgroundColor: status === 'connected' ? '#28a745' : '#dc3545', color: '#fff' }}>
                    WS: {status}
                </span>
            </div>

            <div className="log-console" style={{ 
                flex: 1, 
                backgroundColor: '#1e1e1e', 
                color: '#d4d4d4', 
                padding: '15px', 
                fontFamily: 'Consolas, monospace', 
                overflowY: 'auto',
                borderRadius: '8px',
                minHeight: '500px'
            }}>
                {filteredLogs.map((logLine, idx) => (
                    <div key={idx} style={{ color: getLogColor(logLine), marginBottom: '2px', whiteSpace: 'pre-wrap' }}>
                        {logLine}
                    </div>
                ))}
                {filteredLogs.length === 0 && <div style={{ color: '#666' }}>Логи отсутствуют или не найдены по фильтру...</div>}
            </div>
        </div>
    );
}
