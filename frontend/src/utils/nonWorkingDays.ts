import type {NonWorkingDay, NonWorkingDayType} from '../types/backend';

// Rótulos dos tipos de dia não útil devolvidos por GetAllNonWorkingDays
// (api.NonWorkingDay.type). "holiday" é o feriado nacional e continua
// aparecendo só como "Feriado", como antes dos tipos novos.
export const NON_WORKING_DAY_LABEL: Record<NonWorkingDayType, string> = {
    weekend: 'Fim de semana',
    holiday: 'Feriado',
    state_holiday: 'Feriado estadual',
    municipal: 'Feriado municipal',
    bridge: 'Ponte',
    custom: 'Folga',
    vacation: 'Férias/ausência'
};

// Tipos desconhecidos (backend mais novo que o frontend) viram "Dia não útil".
export const nonWorkingDayLabel = (type: string): string =>
    (NON_WORKING_DAY_LABEL as Record<string, string | undefined>)[type] ?? 'Dia não útil';

// describeNonWorkingDay: "Feriado estadual: Revolução Constitucionalista".
export const describeNonWorkingDay = (day: Pick<NonWorkingDay, 'type' | 'name'>): string => {
    if (day.type === 'weekend') return NON_WORKING_DAY_LABEL.weekend;
    const label = nonWorkingDayLabel(day.type);
    if (day.type === 'vacation') {
        return day.name && day.name !== label ? `${label}: ${day.name}` : label;
    }
    return day.name ? `${label}: ${day.name}` : label;
};

// isVacation separa férias/ausências (cor própria no calendário) dos feriados.
export const isVacation = (day: Pick<NonWorkingDay, 'type'> | undefined): boolean => day?.type === 'vacation';
