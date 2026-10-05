import {useEffect} from 'react';
import {useNavigate} from 'react-router';
import {EventsOn} from '@wailsjs/runtime/runtime';

export const EVENT_REMINDER_OPEN = 'reminder:open';

// Payload de "reminder:open" (backend.ReminderOpen).
export interface ReminderOpenEvent {
    route: string;
    date?: string;
}

// Rotas que um lembrete pode abrir; qualquer outra é ignorada.
const ROTAS_PERMITIDAS = new Set(['/timelog', '/completar']);

// Ouve os cliques nas notificações de lembrete (o backend já trouxe a janela
// para frente) e navega para a tela indicada, levando a data no state.
const useReminderNavigation = (): void => {
    const navigate = useNavigate();

    useEffect(() => {
        const off = EventsOn(EVENT_REMINDER_OPEN, (ev: ReminderOpenEvent) => {
            if (!ROTAS_PERMITIDAS.has(ev.route)) return;
            void navigate(ev.route, {state: ev.date ? {reminderDate: ev.date} : null});
        });
        return off;
    }, [navigate]);
};

export default useReminderNavigation;
