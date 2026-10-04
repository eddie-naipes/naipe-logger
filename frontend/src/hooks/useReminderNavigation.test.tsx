import {beforeEach, describe, expect, it, vi} from 'vitest';
import {act, render, screen} from '@testing-library/react';
import {MemoryRouter, Route, Routes, useLocation} from 'react-router';
import {EventsOn} from '@wailsjs/runtime/runtime';
import useReminderNavigation, {EVENT_REMINDER_OPEN} from './useReminderNavigation';

vi.mock('@wailsjs/runtime/runtime', () => ({
    EventsOn: vi.fn(),
}));

let ouvinte: ((dados: unknown) => void) | undefined;

const Rota = () => {
    const location = useLocation();
    return <p data-testid="rota">{location.pathname}|{JSON.stringify(location.state)}</p>;
};

const Raiz = () => {
    useReminderNavigation();
    return (
        <Routes>
            <Route path="*" element={<Rota/>}/>
        </Routes>
    );
};

describe('useReminderNavigation', () => {
    beforeEach(() => {
        vi.mocked(EventsOn).mockImplementation((evento, callback) => {
            if (evento === EVENT_REMINDER_OPEN) ouvinte = callback as (dados: unknown) => void;
            return () => {
                ouvinte = undefined;
            };
        });
    });

    it('navega para a rota do lembrete levando a data', () => {
        render(<MemoryRouter initialEntries={['/']}><Raiz/></MemoryRouter>);
        act(() => ouvinte?.({route: '/completar', date: '2026-09-30'}));
        expect(screen.getByTestId('rota').textContent).toBe('/completar|{"reminderDate":"2026-09-30"}');
    });

    it('ignora rotas fora da lista', () => {
        render(<MemoryRouter initialEntries={['/']}><Raiz/></MemoryRouter>);
        act(() => ouvinte?.({route: '/config'}));
        expect(screen.getByTestId('rota').textContent).toBe('/|null');
    });
});
