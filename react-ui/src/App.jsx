import React, { useState, useEffect } from 'react';
import { Statistics } from './components/Statistics';
import { Placeholder } from './components/Placeholder';
import { useTheme } from './hooks/useTheme';

// Роутер на основе хэшей (SPA) п.10.2 ТЗ
function App() {
    const [route, setRoute] = useState(window.location.hash || '#processes');
    const { theme, toggleTheme } = useTheme();

    useEffect(() => {
        const onHashChange = () => setRoute(window.location.hash);
        window.addEventListener('hashchange', onHashChange);
        return () => window.removeEventListener('hashchange', onHashChange);
    }, []);

    const renderRoute = () => {
        switch (route) {
            case '#processes':
                return <Placeholder title="Процессы (п.10.4 ТЗ)" />;
            case '#statistics':
                return <Statistics />;
            case '#db':
                return <Placeholder title="База Данных (п.10.6 ТЗ)" />;
            case '#settings':
                return <Placeholder title="Настройки (п.10.7 ТЗ)" />;
            case '#logs':
                return <Placeholder title="Логи (п.10.8 ТЗ)" />;
            case '#diagnose':
                return <Placeholder title="Диагностика (п.10.9 ТЗ)" />;
            default:
                return <Placeholder title="Процессы (п.10.4 ТЗ)" />;
        }
    };

    return (
        <div className="app-container">
            <nav>
                <h2>Venera</h2>
                <a href="#processes" className={route === '#processes' || route === '' ? 'active' : ''}>Процессы</a>
                <a href="#statistics" className={route === '#statistics' ? 'active' : ''}>Статистика</a>
                <a href="#db" className={route === '#db' ? 'active' : ''}>База Данных</a>
                <a href="#settings" className={route === '#settings' ? 'active' : ''}>Настройки</a>
                <a href="#logs" className={route === '#logs' ? 'active' : ''}>Логи</a>
                <a href="#diagnose" className={route === '#diagnose' ? 'active' : ''}>Диагностика</a>

                <button onClick={toggleTheme} className="theme-toggle">
                    Тема: {theme === 'light' ? 'Светлая' : 'Темная'}
                </button>
            </nav>
            <main>
                {renderRoute()}
            </main>
        </div>
    );
}

export default App;
