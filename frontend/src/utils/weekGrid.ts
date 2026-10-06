// Regras puras da grade semanal (página Semana): formatação e leitura de
// durações, totais, delta de edição de célula e status de cada dia. Ficam fora
// do componente para serem testadas sem renderizar a grade.
import type {planning} from '@wailsjs/go/models';
import type {Dados, Task, WorkDay} from '../types/backend';

export type WeekGrid = Dados<planning.WeekGrid>;
export type WeekRow = Dados<planning.WeekRow>;
export type WeekCell = Dados<planning.WeekCell>;

// formatHHMM: 450 -> "07:30"; 0 -> "00:00".
export const formatHHMM = (totalMinutes: number): string => {
    const total = Math.max(0, Math.round(totalMinutes || 0));
    const h = Math.floor(total / 60);
    const m = total % 60;
    return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
};

// parseDuration lê o que o usuário digita numa célula e devolve minutos, ou
// null se não der para entender. Aceita "07:30", "7:30", "7h30", "7h", "45m",
// "1,5"/"1.5" e "8" (números soltos são horas — é como se pensa num
// timesheet). Vazio vale 0.
export const parseDuration = (raw: string): number | null => {
    const texto = raw.trim().toLowerCase().replace(/\s+/g, '');
    if (texto === '') return 0;

    let m = /^(\d{1,2}):(\d{1,2})$/.exec(texto);
    if (m) {
        const minutos = Number(m[2]);
        return minutos < 60 ? Number(m[1]) * 60 + minutos : null;
    }

    m = /^(\d{1,2})h(\d{1,2})?(?:m(?:in)?)?$/.exec(texto);
    if (m) {
        const minutos = m[2] ? Number(m[2]) : 0;
        return minutos < 60 ? Number(m[1]) * 60 + minutos : null;
    }

    m = /^(\d{1,4})m(?:in)?$/.exec(texto);
    if (m) return Number(m[1]);

    m = /^(\d{1,2})(?:[.,](\d{1,2}))?$/.exec(texto);
    if (m) return Math.round(Number(`${m[1]}.${m[2] ?? '0'}`) * 60);

    return null;
};

export type CellDelta =
    | {kind: 'none'}
    | {kind: 'add'; minutes: number}
    | {kind: 'reduce'; minutes: number};

// cellDelta decide o que fazer quando a célula muda de `atual` para `alvo`:
// aumentar cria um lançamento só com a diferença; reduzir nunca apaga sozinho —
// a interface abre a lista de lançamentos para o usuário escolher.
export const cellDelta = (atual: number, alvo: number): CellDelta => {
    if (alvo > atual) return {kind: 'add', minutes: alvo - atual};
    if (alvo < atual) return {kind: 'reduce', minutes: atual - alvo};
    return {kind: 'none'};
};

// Índices (0 = segunda ... 6 = domingo) das colunas exibidas.
export const visibleDayIndexes = (showWeekend: boolean): number[] =>
    showWeekend ? [0, 1, 2, 3, 4, 5, 6] : [0, 1, 2, 3, 4];

export interface GridTotals {
    perDay: number[];
    perRow: number[];
    total: number;
}

// computeTotals soma a grade por dia e por tarefa a partir das células — não
// confia em totais prontos, para continuar certo se a grade for recortada.
export const computeTotals = (rows: readonly WeekRow[], days = 7): GridTotals => {
    const perDay: number[] = Array.from({length: days}, () => 0);
    const perRow = rows.map(row => {
        let soma = 0;
        (row.cells ?? []).forEach((cell, i) => {
            const minutos = cell.minutes || 0;
            soma += minutos;
            if (i < days) perDay[i] = (perDay[i] ?? 0) + minutos;
        });
        return soma;
    });
    return {perDay, perRow, total: perRow.reduce((a, b) => a + b, 0)};
};

export type DayStatus = 'nonworking' | 'future' | 'below' | 'complete';

// dayStatus classifica uma coluna para o destaque visual. Dias futuros não são
// "abaixo da jornada": ainda não aconteceram.
export const dayStatus = (
    date: string,
    minutes: number,
    jornada: number,
    isNonWorking: boolean,
    today: string
): DayStatus => {
    if (isNonWorking) return 'nonworking';
    if (date > today) return 'future';
    return minutes < jornada ? 'below' : 'complete';
};

export interface EntryDefaults {
    description: string;
    isBillable: boolean;
    time: string;
}

// entryDefaults: descrição, billable e horário padrão de um novo lançamento
// na tarefa — da primeira entrada da tarefa salva, ou o nome da tarefa.
export const entryDefaults = (taskName: string, saved?: Task): EntryDefaults => {
    const primeira = saved?.entries?.[0];
    return {
        description: primeira?.description?.trim() || taskName || '',
        isBillable: primeira ? primeira.isBillable : true,
        time: primeira?.time ? primeira.time.substring(0, 5) : '09:00'
    };
};

// buildDeltaWorkDay monta o plano de um único lançamento (a diferença da
// célula) no formato aceito por LogMultipleTimes.
export const buildDeltaWorkDay = (
    date: string,
    taskId: number,
    minutes: number,
    defaults: EntryDefaults
): WorkDay => ({
    date,
    totalMin: minutes,
    entries: [{
        taskId,
        entry: {
            minutes,
            userId: 0,
            time: `${defaults.time}:00`,
            description: defaults.description.trim(),
            isBillable: defaults.isBillable
        }
    }]
});
