import {endOfMonth, format} from 'date-fns';
import {parseLocalDate, toYMD} from './dates';

export interface Period {
    start: string;
    end: string;
}

// currentMonth devolve 'YYYY-MM' do mês de hoje (valor de <input type="month">).
export const currentMonth = (today: Date = new Date()): string => format(today, 'yyyy-MM');

// monthFromNavigationState lê o mês ('YYYY-MM') que outra tela passou no
// state da navegação: {month} (ex.: o Fechamento do mês) ou {reminderDate}
// (clique num lembrete). Qualquer outro formato é ignorado.
export const monthFromNavigationState = (state: unknown): string | null => {
    if (typeof state !== 'object' || state === null) return null;
    const {month, reminderDate} = state as {month?: unknown; reminderDate?: unknown};
    if (typeof month === 'string' && /^\d{4}-\d{2}$/.test(month)) return month;
    if (typeof reminderDate === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(reminderDate)) return reminderDate.slice(0, 7);
    return null;
};

// periodForMonth calcula o período de "Completar período" para o mês escolhido.
// Sem "mês inteiro" o fim é hoje (mês corrente) ou o último dia (mês passado);
// um mês futuro só faz sentido com "mês inteiro" e devolve null sem ele.
export const periodForMonth = (month: string, wholeMonth: boolean, today: string): Period | null => {
    const inicio = parseLocalDate(`${month}-01`);
    if (!inicio) return null;

    const start = toYMD(inicio);
    const ultimo = toYMD(endOfMonth(inicio));

    if (wholeMonth) return {start, end: ultimo};
    if (start > today) return null;
    return {start, end: ultimo < today ? ultimo : today};
};
