import React, { useState, useEffect } from 'react';

export function Processes() {
    const [processes, setProcesses] = useState([]);
    const [loading, setLoading] = useState(true);
    
    // Состояние формы добавления
    const [type, setType] = useState('network');
    const [name, setName] = useState('');
    const [ip, setIp] = useState('');
    const [port, setPort] = useState('');
    const [folderPath, setFolderPath] = useState('');
    const [scanSubfolders, setScanSubfolders] = useState(false);
    const [monitorNewFiles, setMonitorNewFiles] = useState(false);
    const [filePath, setFilePath] = useState('');

    const fetchProcesses = async () => {
        try {
            const res = await fetch('/api/processes');
            const data = await res.json();
            setProcesses(data || []);
        } catch (e) {
            console.error(e);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchProcesses();
        const interval = setInterval(fetchProcesses, 5000);
        return () => clearInterval(interval);
    }, []);

    const handleAction = async (id, action) => {
        await fetch(`/api/process/${id}/action?action=${action}`, { method: 'POST' });
        fetchProcesses();
    };

    const handleDelete = async (id) => {
        await fetch(`/api/processes/${id}`, { method: 'DELETE' });
        fetchProcesses();
    };

    const handleGroupAction = async (action) => {
        await fetch(`/api/processes/actions?action=${action}`, { method: 'POST' });
        fetchProcesses();
    };

    const handleSubmit = async (e) => {
        e.preventDefault();
        const payload = {
            id: '',
            type: type,
            name: name,
            ip: type === 'network' ? ip : '',
            udp_port: type === 'network' ? parseInt(port, 10) : 0,
            folder_path: type === 'folder' ? folderPath : '',
            scan_subfolders: type === 'folder' ? scanSubfolders : false,
            monitor_new_files: type === 'folder' ? monitorNewFiles : false,
            file_path: type === 'file' ? filePath : ''
        };

        await fetch('/api/processes', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        
        // Reset form
        setName(''); setIp(''); setPort(''); setFolderPath(''); setFilePath('');
        fetchProcesses();
    };

    return (
        <div>
            <h2>Управление процессами</h2>
            
            <div className="card">
                <h3>Групповые операции</h3>
                <div style={{ display: 'flex', gap: '10px' }}>
                    <button className="btn-success" onClick={() => handleGroupAction('start')}>Запустить все</button>
                    <button className="btn-warning" onClick={() => handleGroupAction('stop')}>Остановить все</button>
                    <button className="btn-danger" onClick={() => handleGroupAction('delete')}>Удалить все</button>
                </div>
            </div>

            <div className="card">
                <h3>Добавить новый процесс</h3>
                <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                    <label>
                        Тип источника:
                        <select value={type} onChange={e => setType(e.target.value)} style={{ marginLeft: '10px' }}>
                            <option value="network">Сетевой интерфейс</option>
                            <option value="folder">Папка с файлами (PCAP)</option>
                            <option value="file">Одиночный файл (PCAP)</option>
                        </select>
                    </label>
                    <input type="text" placeholder="Название (Name)" value={name} onChange={e => setName(e.target.value)} required />

                    {type === 'network' && (
                        <div style={{ display: 'flex', gap: '10px' }}>
                            <input type="text" placeholder="IP адрес" value={ip} onChange={e => setIp(e.target.value)} required />
                            <input type="number" placeholder="UDP Порт" value={port} onChange={e => setPort(e.target.value)} required />
                        </div>
                    )}

                    {type === 'folder' && (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                            <input type="text" placeholder="Путь к папке" value={folderPath} onChange={e => setFolderPath(e.target.value)} required />
                            <label><input type="checkbox" checked={scanSubfolders} onChange={e => setScanSubfolders(e.target.checked)} /> Сканировать подпапки</label>
                            <label><input type="checkbox" checked={monitorNewFiles} onChange={e => setMonitorNewFiles(e.target.checked)} /> Мониторинг новых файлов</label>
                        </div>
                    )}

                    {type === 'file' && (
                        <input type="text" placeholder="Абсолютный путь к файлу" value={filePath} onChange={e => setFilePath(e.target.value)} required />
                    )}

                    <button type="submit" className="btn-primary" style={{ alignSelf: 'flex-start' }}>Добавить процесс</button>
                </form>
            </div>

            <div className="card">
                <h3>Список процессов</h3>
                {loading ? <p>Загрузка...</p> : (
                    <table className="data-table">
                        <thead>
                            <tr>
                                <th>ID</th>
                                <th>Название</th>
                                <th>Тип</th>
                                <th>Источник</th>
                                <th>Статус</th>
                                <th>Действия</th>
                            </tr>
                        </thead>
                        <tbody>
                            {processes.map(p => (
                                <tr key={p.id}>
                                    <td>{p.id.substring(0, 8)}</td>
                                    <td>{p.name}</td>
                                    <td>{p.type}</td>
                                    <td>{p.type === 'network' ? `${p.ip}:${p.udp_port}` : p.type === 'folder' ? p.folder_path : p.file_path}</td>
                                    <td>
                                        <span className={`status-badge ${p.status === 'running' ? 'running' : p.status === 'error' ? 'error' : 'stopped'}`}>
                                            {p.status}
                                        </span>
                                    </td>
                                    <td>
                                        {p.status !== 'running' && <button onClick={() => handleAction(p.id, 'start')} className="btn-success">Старт</button>}
                                        {p.status === 'running' && <button onClick={() => handleAction(p.id, 'stop')} className="btn-warning">Стоп</button>}
                                        <button onClick={() => handleDelete(p.id)} className="btn-danger" style={{ marginLeft: '5px' }}>Удалить</button>
                                    </td>
                                </tr>
                            ))}
                            {processes.length === 0 && <tr><td colSpan="6" style={{textAlign: 'center'}}>Нет добавленных процессов</td></tr>}
                        </tbody>
                    </table>
                )}
            </div>
        </div>
    );
}
