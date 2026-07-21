import React, { useState, useEffect } from 'react';

export function Settings() {
    const [config, setConfig] = useState(null);
    const [loading, setLoading] = useState(true);
    const [saveStatus, setSaveStatus] = useState('');
    const [pgTestStatus, setPgTestStatus] = useState(null);
    const [dfTestStatus, setDfTestStatus] = useState(null);

    const fetchConfig = async () => {
        try {
            const res = await fetch('/api/config');
            const data = await res.json();
            setConfig(data);
        } catch (e) {
            console.error(e);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchConfig();
    }, []);

    const handleChange = (section, field, value) => {
        setConfig(prev => ({
            ...prev,
            [section]: {
                ...prev[section],
                [field]: value
            }
        }));
    };

    const handleSave = async () => {
        setSaveStatus('Сохранение...');
        try {
            const res = await fetch('/api/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(config)
            });
            if (res.ok) {
                setSaveStatus('Конфигурация успешно сохранена!');
                setTimeout(() => setSaveStatus(''), 3000);
            } else {
                const data = await res.json();
                setSaveStatus(`Ошибка: ${data.error}`);
            }
        } catch (e) {
            setSaveStatus(`Ошибка сети: ${e.message}`);
        }
    };

    const handleTestPostgres = async () => {
        setPgTestStatus('Проверка...');
        try {
            const res = await fetch('/api/config/test-postgres', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(config.PostgreSQL)
            });
            if (res.ok) setPgTestStatus('Успешно (OK)');
            else {
                const d = await res.json();
                setPgTestStatus(`Ошибка: ${d.error}`);
            }
        } catch (e) { setPgTestStatus(`Сбой: ${e.message}`); }
    };

    const handleTestDragonfly = async () => {
        setDfTestStatus('Проверка...');
        try {
            const res = await fetch('/api/config/test-dragonfly', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(config.DragonflyDB)
            });
            if (res.ok) setDfTestStatus('Успешно (OK)');
            else {
                const d = await res.json();
                setDfTestStatus(`Ошибка: ${d.error}`);
            }
        } catch (e) { setDfTestStatus(`Сбой: ${e.message}`); }
    };

    if (loading || !config) return <div>Загрузка конфигурации...</div>;

    return (
        <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <h2>Настройки системы</h2>
                <button onClick={handleSave} className="btn-primary" style={{ padding: '10px 20px' }}>💾 Сохранить изменения</button>
            </div>
            {saveStatus && <p style={{ color: saveStatus.includes('Ошибка') ? 'red' : 'green', fontWeight: 'bold' }}>{saveStatus}</p>}

            <div className="card">
                <h3>Общие (Generic)</h3>
                <div className="grid-2col">
                    <label>Режим работы (Mode): 
                        <select value={config.Generic.mode} onChange={e => handleChange('Generic', 'mode', e.target.value)}>
                            <option value="tray">Системный трей (Tray)</option>
                            <option value="service">Служба (Service)</option>
                        </select>
                    </label>
                    <label>Автостарт процессов:
                        <input type="checkbox" checked={config.Generic.auto_start} onChange={e => handleChange('Generic', 'auto_start', e.target.checked)} />
                    </label>
                    <label>Отображать консоль (show_console_on_startup):
                        <input type="checkbox" checked={config.Generic.show_console_on_startup} onChange={e => handleChange('Generic', 'show_console_on_startup', e.target.checked)} />
                    </label>
                    <label>Максимальное количество процессов:
                        <input type="number" min="1" max="20" value={config.Generic.max_processes} onChange={e => handleChange('Generic', 'max_processes', parseInt(e.target.value))} />
                    </label>
                    <label>Web Server Port:
                        <input type="number" value={config.Generic.web_server_port} onChange={e => handleChange('Generic', 'web_server_port', parseInt(e.target.value))} />
                    </label>
                    <label>Ротация логов (дней):
                        <input type="number" value={config.Generic.log_rotation_days} onChange={e => handleChange('Generic', 'log_rotation_days', parseInt(e.target.value))} />
                    </label>
                </div>
            </div>

            <div className="card">
                <h3>PostgreSQL</h3>
                <div className="grid-2col">
                    <label>Хост: <input type="text" value={config.PostgreSQL.host} onChange={e => handleChange('PostgreSQL', 'host', e.target.value)} /></label>
                    <label>Порт: <input type="number" value={config.PostgreSQL.port} onChange={e => handleChange('PostgreSQL', 'port', parseInt(e.target.value))} /></label>
                    <label>Пользователь: <input type="text" value={config.PostgreSQL.user} onChange={e => handleChange('PostgreSQL', 'user', e.target.value)} /></label>
                    <label>Пароль: <input type="password" value={config.PostgreSQL.password} onChange={e => handleChange('PostgreSQL', 'password', e.target.value)} /></label>
                    <label>База Данных: <input type="text" value={config.PostgreSQL.database} onChange={e => handleChange('PostgreSQL', 'database', e.target.value)} /></label>
                    <label>SSL Mode: 
                        <select value={config.PostgreSQL.ssl_mode} onChange={e => handleChange('PostgreSQL', 'ssl_mode', e.target.value)}>
                            <option value="disable">Disable</option>
                            <option value="require">Require</option>
                        </select>
                    </label>
                </div>
                <div style={{ marginTop: '10px' }}>
                    <button onClick={handleTestPostgres} className="btn-secondary">Проверить подключение</button>
                    <span style={{ marginLeft: '10px', fontWeight: 'bold', color: pgTestStatus?.includes('Успеш') ? 'green' : 'red' }}>{pgTestStatus}</span>
                </div>
            </div>

            <div className="card">
                <h3>DragonflyDB (Кэш)</h3>
                <div className="grid-2col">
                    <label>Хост: <input type="text" value={config.DragonflyDB.host} onChange={e => handleChange('DragonflyDB', 'host', e.target.value)} /></label>
                    <label>Порт: <input type="number" value={config.DragonflyDB.port} onChange={e => handleChange('DragonflyDB', 'port', parseInt(e.target.value))} /></label>
                    <label>Пароль: <input type="password" value={config.DragonflyDB.password} onChange={e => handleChange('DragonflyDB', 'password', e.target.value)} /></label>
                    <label>Размер пакета (Batch Size): <input type="number" value={config.DragonflyDB.batch_size} onChange={e => handleChange('DragonflyDB', 'batch_size', parseInt(e.target.value))} /></label>
                </div>
                <div style={{ marginTop: '10px' }}>
                    <button onClick={handleTestDragonfly} className="btn-secondary">Проверить подключение</button>
                    <span style={{ marginLeft: '10px', fontWeight: 'bold', color: dfTestStatus?.includes('Успеш') ? 'green' : 'red' }}>{dfTestStatus}</span>
                </div>
            </div>

            <div className="card">
                <h3>Пути и Списки (Paths)</h3>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                    <label>Путь к Tshark: <input type="text" style={{width:'100%'}} value={config.Paths.tshark_exe} onChange={e => handleChange('Paths', 'tshark_exe', e.target.value)} /></label>
                    <label>Путь к Podman: <input type="text" style={{width:'100%'}} value={config.Paths.podman_exe} onChange={e => handleChange('Paths', 'podman_exe', e.target.value)} /></label>
                    <label>Список фильтров (.flt): <input type="text" style={{width:'100%'}} value={config.Paths.filter_list} onChange={e => handleChange('Paths', 'filter_list', e.target.value)} /></label>
                    <label>Список контроля (.ctr): <input type="text" style={{width:'100%'}} value={config.Paths.control_list} onChange={e => handleChange('Paths', 'control_list', e.target.value)} /></label>
                    <label>Правила алертов (.alr): <input type="text" style={{width:'100%'}} value={config.Paths.alerts_list} onChange={e => handleChange('Paths', 'alerts_list', e.target.value)} /></label>
                    <label>Директория бэкапов кэша: <input type="text" style={{width:'100%'}} value={config.Paths.db_backup_dir} onChange={e => handleChange('Paths', 'db_backup_dir', e.target.value)} /></label>
                </div>
            </div>

            <div className="card">
                <h3>Системная защита (System)</h3>
                <div className="grid-2col">
                    <label>Критический порог RAM (%): <input type="number" step="0.1" value={config.System.ram_critical_threshold} onChange={e => handleChange('System', 'ram_critical_threshold', parseFloat(e.target.value))} /></label>
                    <label>Предупреждение о диске (%): <input type="number" step="0.1" value={config.System.disk_warning_threshold} onChange={e => handleChange('System', 'disk_warning_threshold', parseFloat(e.target.value))} /></label>
                    <label>Критический порог диска (%): <input type="number" step="0.1" value={config.System.disk_critical_threshold} onChange={e => handleChange('System', 'disk_critical_threshold', parseFloat(e.target.value))} /></label>
                </div>
            </div>
        </div>
    );
}
