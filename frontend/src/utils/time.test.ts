// @vitest-environment node
import {describe, expect, it} from 'vitest';
import {
    formatHoursMinutes,
    formatWorkingDays,
    hoursAndMinutesToMinutes,
    minutesToHoursAndMinutes,
    sumEntryMinutes
} from './time';

describe('formatWorkingDays', () => {
    it.each([
        [undefined, 'Todos os dias'],
        [[], 'Todos os dias'],
        [[0, 1, 2, 3, 4, 5, 6], 'Todos os dias'],
        [[5, 4, 3, 2, 1], 'Dias úteis'],
        [[3, 1], 'Seg, Qua'],
        [[6, 0], 'Dom, Sáb'],
    ])('%j -> %s', (dias, esperado) => {
        expect(formatWorkingDays(dias)).toBe(esperado);
    });

    it('não muta o array recebido', () => {
        const dias = [5, 1, 3];
        formatWorkingDays(dias);
        expect(dias).toEqual([5, 1, 3]);
    });

    it('cinco dias que não são os úteis são listados', () => {
        expect(formatWorkingDays([0, 1, 2, 3, 4])).toBe('Dom, Seg, Ter, Qua, Qui');
    });
});

describe('minutesToHoursAndMinutes', () => {
    it.each([
        [0, {hours: 0, minutes: 0}],
        [59, {hours: 0, minutes: 59}],
        [60, {hours: 1, minutes: 0}],
        [450, {hours: 7, minutes: 30}],
        ['90', {hours: 1, minutes: 30}],
        [-30, {hours: 0, minutes: 0}],
        [null, {hours: 0, minutes: 0}],
        [89.6, {hours: 1, minutes: 30}],
    ])('%j minutos', (minutos, esperado) => {
        expect(minutesToHoursAndMinutes(minutos)).toEqual(esperado);
    });
});

describe('hoursAndMinutesToMinutes', () => {
    it('soma horas e minutos', () => {
        expect(hoursAndMinutesToMinutes(7, 30)).toBe(450);
        expect(hoursAndMinutesToMinutes('2', '15')).toBe(135);
    });

    it('trata valores inválidos como zero', () => {
        expect(hoursAndMinutesToMinutes('abc', 10)).toBe(10);
        expect(hoursAndMinutesToMinutes(1, '')).toBe(60);
    });
});

describe('formatHoursMinutes', () => {
    it.each([
        [480, '8h'],
        [450, '7h30'],
        [65, '1h05'],
        [0, '0h'],
    ])('%i -> %s', (minutos, esperado) => {
        expect(formatHoursMinutes(minutos)).toBe(esperado);
    });
});

describe('sumEntryMinutes', () => {
    it('soma os minutos das entradas', () => {
        expect(sumEntryMinutes([{minutes: 60}, {minutes: 30}, {}])).toBe(90);
    });

    it('aceita lista ausente', () => {
        expect(sumEntryMinutes(undefined)).toBe(0);
        expect(sumEntryMinutes(null)).toBe(0);
    });
});
