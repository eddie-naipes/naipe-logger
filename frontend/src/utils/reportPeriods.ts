import {differenceInCalendarDays, endOfMonth, endOfWeek, startOfMonth, startOfWeek, subDays, subMonths} from 'date-fns';
import {parseLocalDate, toYMD} from './dates';

// Atalhos de período da página de Relatórios. Tudo em datas locais
// 'YYYY-MM-DD' (toYMD), nunca via toISOString.

export type ReportPreset = 'thisMonth' | 'lastMonth' | 'thisWeek' | 'last30' | 'custom';

export interface ReportRange {
    startDate: string;
    endDate: string;
}

export const REPORT_PRESETS: readonly {value: ReportPreset; label: string}[] = [
    {value: 'thisMonth', label: 'Este mês'},
    {value: 'lastMonth', label: 'Mês passado'},
    {value: 'thisWeek', label: 'Esta semana'},
    {value: 'last30', label: 'Últimos 30 dias'},
    {value: 'custom', label: 'Personalizado'}
];

// Mesmo limite do backend (reports.MaxPeriodDays).
export const MAX_REPORT_DAYS = 366;

// presetRange devolve o intervalo do atalho; 'custom' devolve null (o
// usuário escolhe as datas). A semana começa na segunda, como no backend.
export const presetRange = (preset: ReportPreset, hoje: Date = new Date()): ReportRange | null => {
    switch (preset) {
        case 'thisMonth':
            return {startDate: toYMD(startOfMonth(hoje)), endDate: toYMD(endOfMonth(hoje))};
        case 'lastMonth': {
            const mes = subMonths(hoje, 1);
            return {startDate: toYMD(startOfMonth(mes)), endDate: toYMD(endOfMonth(mes))};
        }
        case 'thisWeek':
            return {
                startDate: toYMD(startOfWeek(hoje, {weekStartsOn: 1})),
                endDate: toYMD(endOfWeek(hoje, {weekStartsOn: 1}))
            };
        case 'last30':
            return {startDate: toYMD(subDays(hoje, 29)), endDate: toYMD(hoje)};
        case 'custom':
            return null;
    }
};

// rangeError valida um intervalo personalizado; null quando está ok.
export const rangeError = (range: ReportRange): string | null => {
    if (!range.startDate || !range.endDate) return 'Selecione as datas inicial e final.';
    if (range.startDate > range.endDate) return 'A data final não pode ser anterior à inicial.';
    const inicio = parseLocalDate(range.startDate);
    const fim = parseLocalDate(range.endDate);
    if (!inicio || !fim) return 'Datas inválidas.';
    const dias = differenceInCalendarDays(fim, inicio) + 1;
    if (dias > MAX_REPORT_DAYS) return `O período deve ter no máximo ${MAX_REPORT_DAYS} dias.`;
    return null;
};
