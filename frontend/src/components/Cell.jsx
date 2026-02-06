import { memo } from 'react';

const Cell = memo(function Cell({ value }) {
    let className = 'cell';
    if (value === 1) className += ' player1';
    else if (value === 2) className += ' player2';

    return <div className={className}></div>;
});

export default Cell;
