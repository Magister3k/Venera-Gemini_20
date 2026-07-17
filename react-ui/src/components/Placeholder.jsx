import React from 'react';

// Компонент-заглушка для разделов (п.10.3 ТЗ)
export function Placeholder({ title }) {
    return (
        <div>
            <h2>{title}</h2>
            <p>Этот раздел находится в разработке.</p>
        </div>
    );
}
