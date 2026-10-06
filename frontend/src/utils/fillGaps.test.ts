import {describe, expect, it} from 'vitest';
import {currentMonth, monthFromNavigationState, periodForMonth} from './fillGaps';

describe('periodForMonth', () => {
    it('mês atual vai do dia 1 até hoje', () => {
        expect(periodForMonth('2025-09', false, '2025-09-17')).toEqual({start: '2025-09-01', end: '2025-09-17'});
    });

    it('mês inteiro vai até o último dia, inclusive fevereiro', () => {
        expect(periodForMonth('2025-09', true, '2025-09-17')).toEqual({start: '2025-09-01', end: '2025-09-30'});
        expect(periodForMonth('2024-02', true, '2024-02-10')).toEqual({start: '2024-02-01', end: '2024-02-29'});
    });

    it('mês passado vai até o último dia', () => {
        expect(periodForMonth('2025-08', false, '2025-09-17')).toEqual({start: '2025-08-01', end: '2025-08-31'});
    });

    it('mês futuro só com mês inteiro', () => {
        expect(periodForMonth('2025-10', false, '2025-09-17')).toBeNull();
        expect(periodForMonth('2025-10', true, '2025-09-17')).toEqual({start: '2025-10-01', end: '2025-10-31'});
    });

    it('mês inválido devolve null', () => {
        expect(periodForMonth('', false, '2025-09-17')).toBeNull();
    });
});

describe('currentMonth', () => {
    it('formata no fuso local', () => {
        expect(currentMonth(new Date(2025, 8, 30, 23, 30))).toBe('2025-09');
    });
});

describe('monthFromNavigationState', () => {
    it('aceita {month} e {reminderDate}', () => {
        expect(monthFromNavigationState({month: '2026-09'})).toBe('2026-09');
        expect(monthFromNavigationState({reminderDate: '2026-09-30'})).toBe('2026-09');
    });

    it('ignora state ausente ou em outro formato', () => {
        expect(monthFromNavigationState(null)).toBeNull();
        expect(monthFromNavigationState('2026-09')).toBeNull();
        expect(monthFromNavigationState({month: '09/2026'})).toBeNull();
        expect(monthFromNavigationState({reminderDate: 42})).toBeNull();
    });
});
