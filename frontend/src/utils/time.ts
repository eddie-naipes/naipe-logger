// Utilitários de jornada: conversão minutos↔horas e dias da semana das tarefas.

export interface DiaSemana {
    id: number;
    nome: string;
    abrev: string;
}

export const DIAS_SEMANA: readonly DiaSemana[] = [
    {id: 1, nome: 'Segunda', abrev: 'Seg'},
    {id: 2, nome: 'Terça', abrev: 'Ter'},
    {id: 3, nome: 'Quarta', abrev: 'Qua'},
    {id: 4, nome: 'Quinta', abrev: 'Qui'},
    {id: 5, nome: 'Sexta', abrev: 'Sex'},
    {id: 6, nome: 'Sábado', abrev: 'Sáb'},
    {id: 0, nome: 'Domingo', abrev: 'Dom'}
];

// Abreviações indexadas por getDay() (0 = domingo).
export const DIAS_ABREV = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb'] as const;

export const DIAS_UTEIS: readonly number[] = [1, 2, 3, 4, 5];

// formatWorkingDays descreve os dias da semana de uma tarefa. Não muta o array
// recebido (que normalmente vem do estado do React).
export const formatWorkingDays = (workingDays: readonly number[] | null | undefined): string => {
    if (!workingDays || workingDays.length === 0) return 'Todos os dias';
    if (workingDays.length === 7) return 'Todos os dias';
    if (workingDays.length === 5 && DIAS_UTEIS.every(day => workingDays.includes(day))) {
        return 'Dias úteis';
    }

    return [...workingDays]
        .sort((a, b) => a - b)
        .map(day => DIAS_ABREV[day] ?? `Dia ${day}`)
        .join(', ');
};

export interface HorasMinutos {
    hours: number;
    minutes: number;
}

export const minutesToHoursAndMinutes = (totalMinutes: number | string | null | undefined): HorasMinutos => {
    const total = Math.max(0, Math.round(Number(totalMinutes) || 0));
    return {hours: Math.floor(total / 60), minutes: total % 60};
};

export const hoursAndMinutesToMinutes = (hours: number | string, minutes: number | string): number =>
    ((Number(hours) || 0) * 60) + (Number(minutes) || 0);

// formatHoursMinutes: 450 -> "7h30" ; 480 -> "8h".
export const formatHoursMinutes = (totalMinutes: number): string => {
    const {hours, minutes} = minutesToHoursAndMinutes(totalMinutes);
    return minutes === 0 ? `${hours}h` : `${hours}h${String(minutes).padStart(2, '0')}`;
};

export const sumEntryMinutes = (entries: readonly {minutes?: number}[] | null | undefined): number =>
    (entries ?? []).reduce((sum, e) => sum + (e.minutes || 0), 0);

// Jornada padrão usada enquanto a configuração não carrega (8h).
export const MINUTOS_POR_DIA_PADRAO = 480;
