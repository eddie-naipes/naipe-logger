import {describe, expect, it} from 'vitest';
import {
    buildDeltaWorkDay,
    cellDelta,
    computeTotals,
    dayStatus,
    entryDefaults,
    formatHHMM,
    parseDuration,
    visibleDayIndexes,
    type WeekRow
} from './weekGrid';

const linha = (taskId: number, minutos: number[]): WeekRow => ({
    taskId,
    taskName: `T${taskId}`,
    projectName: 'P',
    saved: false,
    total: 0,
    cells: minutos.map((m, i) => ({date: `2025-09-${15 + i}`, minutes: m, entries: []}))
});

describe('formatHHMM', () => {
    it('formata minutos como hh:mm', () => {
        expect(formatHHMM(450)).toBe('07:30');
        expect(formatHHMM(0)).toBe('00:00');
        expect(formatHHMM(605)).toBe('10:05');
    });
});

describe('parseDuration', () => {
    it.each([
        ['07:30', 450],
        ['7:30', 450],
        ['7h30', 450],
        ['7h', 420],
        ['45m', 45],
        ['90min', 90],
        ['1,5', 90],
        ['1.5', 90],
        ['8', 480],
        ['', 0],
        ['  ', 0]
    ])('"%s" vira %d min', (texto, esperado) => {
        expect(parseDuration(texto)).toBe(esperado);
    });

    it.each(['abc', '7:75', '7h60', '-1', '1:2:3'])('"%s" é inválido', texto => {
        expect(parseDuration(texto)).toBeNull();
    });
});

describe('cellDelta', () => {
    it('aumentar cria só a diferença', () => {
        expect(cellDelta(120, 180)).toEqual({kind: 'add', minutes: 60});
        expect(cellDelta(0, 30)).toEqual({kind: 'add', minutes: 30});
    });

    it('reduzir não apaga: só informa quanto sobra', () => {
        expect(cellDelta(180, 60)).toEqual({kind: 'reduce', minutes: 120});
    });

    it('sem mudança não faz nada', () => {
        expect(cellDelta(60, 60)).toEqual({kind: 'none'});
    });
});

describe('computeTotals', () => {
    it('soma por dia, por tarefa e no geral', () => {
        const rows = [linha(1, [60, 120, 0, 0, 0, 0, 0]), linha(2, [30, 0, 0, 0, 0, 0, 45])];
        const totals = computeTotals(rows);
        expect(totals.perDay).toEqual([90, 120, 0, 0, 0, 0, 45]);
        expect(totals.perRow).toEqual([180, 75]);
        expect(totals.total).toBe(255);
    });

    it('grade vazia dá zeros', () => {
        expect(computeTotals([])).toEqual({perDay: [0, 0, 0, 0, 0, 0, 0], perRow: [], total: 0});
    });
});

describe('visibleDayIndexes', () => {
    it('esconde sábado e domingo por padrão', () => {
        expect(visibleDayIndexes(false)).toEqual([0, 1, 2, 3, 4]);
        expect(visibleDayIndexes(true)).toHaveLength(7);
    });
});

describe('dayStatus', () => {
    it('classifica os dias', () => {
        expect(dayStatus('2025-09-15', 0, 480, true, '2025-09-20')).toBe('nonworking');
        expect(dayStatus('2025-09-25', 0, 480, false, '2025-09-20')).toBe('future');
        expect(dayStatus('2025-09-15', 300, 480, false, '2025-09-20')).toBe('below');
        expect(dayStatus('2025-09-20', 480, 480, false, '2025-09-20')).toBe('complete');
    });
});

describe('entryDefaults e buildDeltaWorkDay', () => {
    it('usa a primeira entrada da tarefa salva', () => {
        const d = entryDefaults('Tarefa', {
            taskId: 1, taskName: 'Tarefa', projectId: 1, projectName: 'P',
            entries: [{minutes: 60, userId: 0, time: '14:00:00', description: 'Dev', isBillable: false}]
        });
        expect(d).toEqual({description: 'Dev', isBillable: false, time: '14:00'});
    });

    it('sem tarefa salva usa o nome e billable', () => {
        expect(entryDefaults('Suporte')).toEqual({description: 'Suporte', isBillable: true, time: '09:00'});
    });

    it('monta um WorkDay com a diferença', () => {
        const wd = buildDeltaWorkDay('2025-09-15', 7, 45, {description: ' Dev ', isBillable: true, time: '10:30'});
        expect(wd).toEqual({
            date: '2025-09-15',
            totalMin: 45,
            entries: [{taskId: 7, entry: {minutes: 45, userId: 0, time: '10:30:00', description: 'Dev', isBillable: true}}]
        });
    });
});
