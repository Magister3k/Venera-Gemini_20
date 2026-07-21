import React, { useState } from 'react';

export function Diagnose() {
    const [report, setReport] = useState(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState(null);

    const handleRunDiagnose = async () => {
        setLoading(true);
        setError(null);
        try {
            const res = await fetch('/api/diagnose/run', { method: 'POST' });
            if (res.ok) {
                const data = await res.json();
                setReport(data);
            } else {
                const errData = await res.json();
                setError(errData.error);
            }
        } catch (e) {
            setError(e.message);
        } finally {
            setLoading(false);
        }
    };

    const handleDownloadPDF = () => {
        window.open('/api/diagnose/pdf', '_blank');
    };

    const handleDownloadArchive = () => {
        window.open('/api/diagnose/archive', '_blank');
    };

    const StatusIcon = ({ ok }) => (
        <span style={{ color: ok ? 'green' : 'red', fontWeight: 'bold', marginRight: '10px' }}>
            {ok ? '✓ Успешно' : '❌ Ошибка'}
        </span>
    );

    return (
        <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <h2>Диагностика системы</h2>
                <div style={{ display: 'flex', gap: '10px' }}>
                    <button onClick={handleRunDiagnose} className="btn-primary" disabled={loading}>
                        {loading ? 'Выполнение...' : 'Запустить диагностику'}
                    </button>
                    {report && (
                        <>
                            <button onClick={handleDownloadPDF} className="btn-secondary">Скачать PDF отчет</button>
                            <button onClick={handleDownloadArchive} className="btn-success">Скачать GZ архив</button>
                        </>
                    )}
                </div>
            </div>

            {error && <div className="card" style={{ color: 'red', fontWeight: 'bold' }}>Ошибка запуска диагностики: {error}</div>}

            {report && (
                <div className="card" style={{ marginTop: '20px' }}>
                    <h3>Результаты проверок</h3>
                    <ul style={{ listStyle: 'none', padding: 0, lineHeight: '1.8' }}>
                        <li><StatusIcon ok={true} /> <strong>Версия приложения:</strong> {report.AppVersion}</li>
                        <li><StatusIcon ok={true} /> <strong>Режим работы:</strong> {report.AppMode}</li>
                        <li><StatusIcon ok={report.ConfigExists} /> <strong>Наличие файла конфигурации config.toml</strong></li>
                        <li><StatusIcon ok={report.ManifestRegistered} /> <strong>ETW Манифест зарегистрирован (Версия: {report.ManifestVersion})</strong></li>
                        <li><StatusIcon ok={report.ServiceInstalled} /> <strong>Служба Windows VeneraSrv:</strong> {report.ServiceStatus}</li>
                        <li><StatusIcon ok={report.PGConnected} /> <strong>Подключение к СУБД PostgreSQL</strong></li>
                        <li><StatusIcon ok={report.DragonflyImageExist} /> <strong>Подключение к СУБД DragonflyDB (кэш)</strong></li>
                        <li><StatusIcon ok={report.TsharkExists} /> <strong>Наличие утилиты Tshark</strong></li>
                        <li><StatusIcon ok={report.PodmanExists} /> <strong>Наличие утилиты Podman</strong></li>
                    </ul>

                    <h3>Ресурсы системы</h3>
                    <ul style={{ listStyle: 'none', padding: 0, lineHeight: '1.8' }}>
                        <li><strong>Свободная ОЗУ:</strong> {report.FreeRAMPercent.toFixed(2)}% ({Math.floor(report.FreeRAMBytes / 1024 / 1024)} MB)</li>
                        <li><strong>Свободно на диске (БД):</strong> {Math.floor(report.PGDiskFreeBytes / 1024 / 1024)} MB</li>
                    </ul>

                    {report.EventLogErrors && report.EventLogErrors.length > 0 && (
                        <div style={{ marginTop: '20px' }}>
                            <h3 style={{ color: '#ff4c4c' }}>Последние ошибки Windows Event Log</h3>
                            <pre style={{ backgroundColor: '#1e1e1e', color: '#ffc107', padding: '10px', borderRadius: '4px', overflowX: 'auto' }}>
                                {report.EventLogErrors.join('\n')}
                            </pre>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
