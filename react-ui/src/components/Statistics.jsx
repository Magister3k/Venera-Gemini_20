import React from 'react';
import { useWebSocket } from '../hooks/useWebSocket';

// Компонент Статистики (п.10.5 ТЗ)
export function Statistics() {
    const { data, status } = useWebSocket('/ws/statistics');

    if (status !== 'connected') {
        return <div>Подключение к серверу статистики: {status}...</div>;
    }

    if (!data) return <div>Ожидание данных...</div>;

    const { system, processes } = data;

    return (
        <div>
            <h2>Статистика системы в реальном времени (п.1.13 ТЗ)</h2>
            <div className="card">
                <h3>Общая система</h3>
                <p>RAM свободно: {system.free_ram_percent?.toFixed(2)}% ({Math.floor((system.free_ram_bytes||0) / 1024 / 1024)} MB)</p>
                <p>Диск БД свободно: {system.db_disk_free_percent?.toFixed(2)}%</p>
                <p>Размер DragonflyDB: {system.dragonfly_db_size} байт</p>
                <p>Размер PostgreSQL: {system.postgresql_size} байт</p>
            </div>

            <h3>Процессы</h3>
            {Object.keys(processes || {}).map(id => {
                const p = processes[id];
                return (
                    <div key={id} className="card">
                        <h4>Процесс: {id}</h4>
                        <p>Скорость потока (Tshark): {p.input_speed_bps?.toFixed(2)} bps</p>
                        <p>Потребление RAM: {p.ram_consumption} байт</p>
                        <p>Загрузка CPU: {p.cpu_load_percent?.toFixed(2)}%</p>
                        <p>Всего пар JSON: {p.total_pairs}</p>
                        <p>Отобрано после фильтра: {p.filtered_pairs}</p>
                        <p>Уникальных в БД: {p.unique_pairs_sent}</p>
                    </div>
                );
            })}
        </div>
    );
}
