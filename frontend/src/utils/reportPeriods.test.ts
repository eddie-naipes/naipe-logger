import {describe, expect, it} from 'vitest';
import {presetRange, rangeError} from './reportPeriods';

// 15/10/2026 é quinta-feira.
const hoje = new Date(2026, 9, 15);

describe('presetRange', () => {
    it('calcula os atalhos em datas locais', () => {
        expect(presetRange('thisMonth', hoje)).toEqual({startDate: '2026-10-01', endDate: '2026-10-31'});
        expect(presetRange('lastMonth', hoje)).toEqual({startDate: '2026-09-01', endDate: '2026-09-30'});
        expect(presetRange('thisWeek', hoje)).toEqual({startDate: '2026-10-12', endDate: '2026-10-18'});
        expect(presetRange('last30', hoje)).toEqual({startDate: '2026-09-16', endDate: '2026-10-15'});
        expect(presetRange('custom', hoje)).toBeNull();
    });

    it('mês passado em janeiro volta para dezembro do ano anterior', () => {
        expect(presetRange('lastMonth', new Date(2026, 0, 10))).toEqual({startDate: '2025-12-01', endDate: '2025-12-31'});
    });
});

describe('rangeError', () => {
    it('valida o período personalizado', () => {
        expect(rangeError({startDate: '', endDate: '2026-01-01'})).toMatch(/Selecione/);
        expect(rangeError({startDate: '2026-02-01', endDate: '2026-01-01'})).toMatch(/anterior/);
        expect(rangeError({startDate: '2025-01-01', endDate: '2026-01-02'})).toMatch(/366/);
        expect(rangeError({startDate: '2024-01-01', endDate: '2024-12-31'})).toBeNull();
    });
});
