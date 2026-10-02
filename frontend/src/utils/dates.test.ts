// @vitest-environment node
import {afterEach, describe, expect, it, vi} from 'vitest';
import {ptBR} from 'date-fns/locale';
import {formatDateBR, parseLocalDate, todayYMD, toYMD, utcTimestampToYMD} from './dates';

describe('fuso dos testes', () => {
    it('roda em America/Sao_Paulo (UTC-3)', () => {
        // Sem isso os testes abaixo passariam em UTC sem provar nada. O fuso
        // vem de test.env.TZ no vite.config.ts.
        expect(Intl.DateTimeFormat().resolvedOptions().timeZone).toBe('America/Sao_Paulo');
        expect(new Date(2024, 4, 1).getTimezoneOffset()).toBe(180);
    });
});

describe('parseLocalDate', () => {
    it('lê YYYY-MM-DD como meia-noite local, não UTC', () => {
        const data = parseLocalDate('2024-05-01');
        expect(data).not.toBeNull();
        expect(data?.getFullYear()).toBe(2024);
        expect(data?.getMonth()).toBe(4);
        expect(data?.getDate()).toBe(1);
        expect(data?.getHours()).toBe(0);
    });

    it('é diferente de new Date(texto), que cai no dia anterior no Brasil', () => {
        expect(new Date('2024-05-01').getDate()).toBe(30);
        expect(parseLocalDate('2024-05-01')?.getDate()).toBe(1);
    });

    it('aceita ISO com horário', () => {
        const data = parseLocalDate('2024-05-01T10:30:00');
        expect(data?.getHours()).toBe(10);
        expect(data?.getMinutes()).toBe(30);
    });

    it('devolve a própria Date quando válida', () => {
        const data = new Date(2024, 0, 15);
        expect(parseLocalDate(data)).toBe(data);
    });

    it.each([null, undefined, '', 'não é data', '2024-13-45'])('devolve null para %j', (valor) => {
        expect(parseLocalDate(valor)).toBeNull();
    });

    it('devolve null para Date inválida', () => {
        expect(parseLocalDate(new Date(Number.NaN))).toBeNull();
    });
});

describe('toYMD', () => {
    it('formata pela data local mesmo tarde da noite', () => {
        // 23h30 em São Paulo já é dia 2 em UTC; toISOString erraria o dia.
        const noite = new Date(2024, 4, 1, 23, 30);
        expect(noite.toISOString().slice(0, 10)).toBe('2024-05-02');
        expect(toYMD(noite)).toBe('2024-05-01');
    });

    it('preenche mês e dia com zero', () => {
        expect(toYMD(new Date(2024, 0, 5))).toBe('2024-01-05');
    });
});

describe('todayYMD', () => {
    afterEach(() => {
        vi.useRealTimers();
    });

    it('usa o dia local, não o dia UTC', () => {
        vi.useFakeTimers();
        // 01/05 às 23h30 em São Paulo = 02/05 às 02h30 UTC.
        vi.setSystemTime(new Date('2024-05-02T02:30:00Z'));
        expect(todayYMD()).toBe('2024-05-01');
    });
});

describe('utcTimestampToYMD', () => {
    it('lê o timestamp da API de calendário em UTC', () => {
        const meiaNoiteUTC = Date.UTC(2024, 4, 1);
        // Em horário local seria 30/04 às 21h.
        expect(new Date(meiaNoiteUTC).getDate()).toBe(30);
        expect(utcTimestampToYMD(meiaNoiteUTC)).toBe('2024-05-01');
    });

    it('preenche mês e dia com zero', () => {
        expect(utcTimestampToYMD(Date.UTC(2023, 8, 3))).toBe('2023-09-03');
    });

    it('devolve null para timestamp inválido', () => {
        expect(utcTimestampToYMD(Number.NaN)).toBeNull();
    });
});

describe('formatDateBR', () => {
    it('formata no padrão brasileiro por padrão', () => {
        expect(formatDateBR('2024-05-01')).toBe('01/05/2024');
    });

    it('aceita padrão e locale', () => {
        expect(formatDateBR('2024-05-01', "EEEE, dd 'de' MMMM", '—', {locale: ptBR})).toBe('quarta-feira, 01 de maio');
    });

    it('devolve o fallback para datas ausentes ou inválidas', () => {
        expect(formatDateBR(undefined)).toBe('—');
        expect(formatDateBR('lixo', 'dd/MM/yyyy', 'Data inválida')).toBe('Data inválida');
    });
});
