import React, { useState, useEffect } from 'react';
import { Processes } from './components/Processes';
import { Statistics } from './components/Statistics';
import { Database } from './components/Database';
import { Settings } from './components/Settings';
import { Logs } from './components/Logs';
import { Diagnose } from './components/Diagnose';
import { Help } from './components/Help';
import { About } from './components/About';
import { useTheme } from './hooks/useTheme';

function App() {
    const [route, setRoute] = useState(window.location.hash || '#processes');
    const { theme, toggleTheme } = useTheme();

    // Состояния навигационной панели (п.3 плана)
    const [isPinned, setIsPinned] = useState(() => localStorage.getItem('venera-nav-pinned') !== 'false');
    const [isHovered, setIsHovered] = useState(false);

    useEffect(() => {
        const onHashChange = () => setRoute(window.location.hash);
        window.addEventListener('hashchange', onHashChange);
        return () => window.removeEventListener('hashchange', onHashChange);
    }, []);

    const togglePin = () => {
        const nextState = !isPinned;
        setIsPinned(nextState);
        localStorage.setItem('venera-nav-pinned', String(nextState));
    };

    const isExpanded = isPinned || isHovered;

    const renderRoute = () => {
        switch (route) {
            case '#processes': return <Processes />;
            case '#statistics': return <Statistics />;
            case '#db': return <Database />;
            case '#settings': return <Settings />;
            case '#logs': return <Logs />;
            case '#diagnose': return <Diagnose />;
            case '#help': return <Help />;
            case '#about': return <About />;
            default: return <Processes />;
        }
    };

    const navLinkClass = (path) => {
        const isActive = route === path || (route === '' && path === '#processes');
        return isActive ? 'active' : '';
    };

    return (
        <div className="app-container">
            <nav 
                className={isExpanded ? 'expanded' : ''} 
                onMouseEnter={() => setIsHovered(true)} 
                onMouseLeave={() => setIsHovered(false)}
            >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <h2>{isExpanded ? 'Venera' : 'V'}</h2>
                    <button className={`pin-button ${isPinned ? 'pinned' : ''}`} onClick={togglePin} title="Закрепить панель">
                        📌
                    </button>
                </div>

                <a href="#processes" className={navLinkClass('#processes')}>⚙️ <span>Процессы</span></a>
                <a href="#statistics" className={navLinkClass('#statistics')}>📊 <span>Статистика</span></a>
                <a href="#db" className={navLinkClass('#db')}>🗄️ <span>База Данных</span></a>
                <a href="#settings" className={navLinkClass('#settings')}>🔧 <span>Настройки</span></a>
                <a href="#logs" className={navLinkClass('#logs')}>📝 <span>Логи</span></a>
                <a href="#diagnose" className={navLinkClass('#diagnose')}>🩺 <span>Диагностика</span></a>
                <a href="#help" className={navLinkClass('#help')}>❓ <span>Помощь</span></a>
                <a href="#about" className={navLinkClass('#about')}>ℹ️ <span>О программе</span></a>

                <button onClick={toggleTheme} className="theme-toggle" style={{ marginTop: 'auto', display: 'flex', gap: '10px', alignItems: 'center' }}>
                    {theme === 'light' ? '☀️' : '🌙'} <span>{theme === 'light' ? 'Светлая' : 'Темная'}</span>
                </button>
            </nav>
            <main>
                {renderRoute()}
            </main>
        </div>
    );
}

export default App;

