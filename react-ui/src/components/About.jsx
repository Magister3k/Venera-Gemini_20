import React, { useEffect, useState } from 'react';

export function About() {
    const [info, setInfo] = useState(null);

    useEffect(() => {
        fetch('/api/about')
            .then(res => res.json())
            .then(data => setInfo(data))
            .catch(err => console.error("Ошибка загрузки информации о программе", err));
    }, []);

    if (!info) return <div>Загрузка...</div>;

    return (
        <div style={{ maxWidth: '600px', margin: '50px auto', textAlign: 'center' }}>
            <div className="card" style={{ padding: '40px' }}>
                <div style={{ fontSize: '48px', marginBottom: '20px' }}>🪐</div>
                <h1 style={{ marginBottom: '5px' }}>{info.app_name}</h1>
                <p style={{ color: '#666', marginTop: 0 }}>Версия: {info.version}</p>
                
                <hr style={{ margin: '30px 0', border: 'none', borderTop: '1px solid var(--border-color)' }} />
                
                <div style={{ textAlign: 'left', lineHeight: '1.8' }}>
                    <p><strong>Разработчик:</strong> {info.developer}</p>
                    <p><strong>Дата сборки:</strong> {info.build_time}</p>
                    <p><strong>Лицензия:</strong> {info.license}</p>
                </div>

                <p style={{ marginTop: '40px', fontSize: '12px', color: '#888' }}>
                    {info.copyright}
                </p>
            </div>
        </div>
    );
}
