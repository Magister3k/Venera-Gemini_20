import React, { useState, useEffect } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';

export function Database() {
    const { data: wsData, status, send } = useWebSocket('/ws/db');
    const [search, setSearch] = useState('');
    const [dbData, setDbData] = useState([]);

    useEffect(() => {
        if (wsData) {
            if (wsData.error) {
                console.error("DB WS Error:", wsData.error);
            } else {
                setDbData(wsData);
            }
        }
    }, [wsData]);

    useEffect(() => {
        // Отправка фильтра при изменении поиска с задержкой (debounce)
        const timer = setTimeout(() => {
            if (status === 'connected' && send) {
                send(JSON.stringify({ search: search, limit: 100 }));
            }
        }, 500);
        return () => clearTimeout(timer);
    }, [search, status, send]);

    const handleExport = () => {
        window.open('/api/db/export/xlsx', '_blank');
    };

    return (
        <div>
            <h2>База Данных (PostgreSQL)</h2>
            <div className="card" style={{ display: 'flex', gap: '15px', alignItems: 'center' }}>
                <input 
                    type="text" 
                    placeholder="Быстрый поиск по БД (source, key, value)..." 
                    value={search} 
                    onChange={e => setSearch(e.target.value)} 
                    style={{ flex: 1, padding: '8px' }}
                />
                <button onClick={handleExport} className="btn-success">Экспорт в Excel (.xlsx)</button>
            </div>

            <div className="card">
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '10px' }}>
                    <h3>Выборка данных</h3>
                    <span className="status-badge" style={{ backgroundColor: status === 'connected' ? '#28a745' : '#dc3545', color: '#fff' }}>
                        WebSocket: {status}
                    </span>
                </div>
                
                <div style={{ overflowX: 'auto' }}>
                    <table className="data-table">
                        <thead>
                            <tr>
                                <th>Источник</th>
                                <th>Ключ</th>
                                <th>Значение</th>
                                <th>Дата первого появления</th>
                                <th>Дата последнего появления</th>
                            </tr>
                        </thead>
                        <tbody>
                            {dbData.length > 0 ? dbData.map((row, idx) => (
                                <tr key={idx}>
                                    <td>{row.source}</td>
                                    <td>{row.key}</td>
                                    <td>{row.value}</td>
                                    <td>{new Date(row.date_first).toLocaleString()}</td>
                                    <td>{new Date(row.date_last).toLocaleString()}</td>
                                </tr>
                            )) : (
                                <tr><td colSpan="5" style={{textAlign: 'center'}}>Нет данных или поиск не дал результатов</td></tr>
                            )}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    );
}
