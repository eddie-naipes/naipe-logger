import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';
import {act, fireEvent, render, screen} from '@testing-library/react';
import {type ReactNode, useCallback, useMemo, useState} from 'react';
import {TimeEntriesContext, type TimeEntriesSignal, useOnTimeEntriesChanged, useTimeEntriesSignal} from './TimeEntriesContext';

// Provider igual ao do App.
const Provider = ({children}: {children: ReactNode}) => {
    const [version, setVersion] = useState(0);
    const notifyChanged = useCallback(() => setVersion(v => v + 1), []);
    const valor = useMemo<TimeEntriesSignal>(() => ({version, notifyChanged}), [version, notifyChanged]);
    return <TimeEntriesContext.Provider value={valor}>{children}</TimeEntriesContext.Provider>;
};
const Emissor = () => {
    const {notifyChanged} = useTimeEntriesSignal();
    return <button onClick={notifyChanged}>alterar lançamentos</button>;
};
const disparar = () => fireEvent.click(screen.getByRole('button', {name: 'alterar lançamentos'}));
const Ouvinte = ({onChange, delayMs}: {onChange: () => void; delayMs?: number}) => {
    useOnTimeEntriesChanged(onChange, delayMs);
    return null;
};

describe('sinal de lançamentos alterados', () => {
    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());

    it('não dispara na montagem e recarrega depois do atraso', () => {
        const onChange = vi.fn();
        render(<Provider><Emissor/><Ouvinte onChange={onChange} delayMs={1000}/></Provider>);

        act(() => { vi.advanceTimersByTime(5000); });
        expect(onChange).not.toHaveBeenCalled();

        act(() => { disparar(); });
        act(() => { vi.advanceTimersByTime(999); });
        expect(onChange).not.toHaveBeenCalled();
        act(() => { vi.advanceTimersByTime(1); });
        expect(onChange).toHaveBeenCalledTimes(1);
    });

    it('junta notificações seguidas numa única recarga', () => {
        const onChange = vi.fn();
        render(<Provider><Emissor/><Ouvinte onChange={onChange} delayMs={1000}/></Provider>);

        act(() => { disparar(); });
        act(() => { vi.advanceTimersByTime(500); });
        act(() => { disparar(); });
        act(() => { vi.advanceTimersByTime(1000); });

        expect(onChange).toHaveBeenCalledTimes(1);
    });

    it('fora do provider é inerte', () => {
        const onChange = vi.fn();
        render(<><Emissor/><Ouvinte onChange={onChange}/></>);

        act(() => { disparar(); });
        act(() => { vi.advanceTimersByTime(5000); });
        expect(onChange).not.toHaveBeenCalled();
    });
});
