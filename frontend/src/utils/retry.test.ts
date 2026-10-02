// @vitest-environment node
import {describe, expect, it} from 'vitest';
import type {DeleteTimeEntryResult, EntryTask, TimeLogResult, WorkDay} from '../types/backend';
import {buildRetryWorkDays, failedDeleteIds} from './retry';

const entrada = (taskId: number, time: string, description = `tarefa ${taskId} ${time}`): EntryTask => ({
    taskId,
    entry: {minutes: 60, userId: 1, time, description, isBillable: true}
});

const dia = (date: string, entries: EntryTask[]): WorkDay => ({
    date,
    entries,
    totalMin: entries.reduce((s, e) => s + e.entry.minutes, 0)
});

const resultado = (date: string, taskId: number, success: boolean, entryId = 0): TimeLogResult => ({
    date,
    taskId,
    success,
    entryId,
    message: success ? 'ok' : 'falhou'
});

describe('buildRetryWorkDays', () => {
    it('não reenvia nada quando tudo deu certo', () => {
        const plano = [dia('2024-05-01', [entrada(10, '09:00:00')])];
        expect(buildRetryWorkDays([resultado('2024-05-01', 10, true, 1)], plano)).toEqual([]);
    });

    it('reenvia só a entrada que falhou, casando por dia e tarefa', () => {
        const plano = [
            dia('2024-05-01', [entrada(10, '09:00:00'), entrada(20, '10:00:00')]),
            dia('2024-05-02', [entrada(10, '09:00:00'), entrada(20, '10:00:00')]),
        ];
        const resultados = [
            resultado('2024-05-01', 10, true, 1),
            resultado('2024-05-01', 20, true, 2),
            resultado('2024-05-02', 10, true, 3),
            resultado('2024-05-02', 20, false),
        ];

        const reenviar = buildRetryWorkDays(resultados, plano);

        expect(reenviar).toHaveLength(1);
        expect(reenviar[0]?.date).toBe('2024-05-02');
        expect(reenviar[0]?.entries.map(e => e.taskId)).toEqual([20]);
    });

    it('não depende da ordem dos resultados', () => {
        const plano = [
            dia('2024-05-01', [entrada(10, '09:00:00')]),
            dia('2024-05-02', [entrada(10, '09:00:00')]),
        ];
        // O resultado da falha chega antes do sucesso, fora da ordem do plano.
        const resultados = [resultado('2024-05-02', 10, false), resultado('2024-05-01', 10, true, 7)];

        const reenviar = buildRetryWorkDays(resultados, plano);
        expect(reenviar.map(d => d.date)).toEqual(['2024-05-02']);
    });

    it('com várias entradas da mesma tarefa no dia, reenvia no máximo a quantidade de falhas', () => {
        const plano = [dia('2024-05-01', [
            entrada(10, '09:00:00', 'manhã'),
            entrada(10, '13:00:00', 'tarde'),
            entrada(10, '16:00:00', 'fim do dia'),
        ])];
        // Uma das três falhou: só uma pode ser reenviada, senão duplicaria horas.
        const resultados = [
            resultado('2024-05-01', 10, true, 1),
            resultado('2024-05-01', 10, false),
            resultado('2024-05-01', 10, true, 3),
        ];

        const reenviar = buildRetryWorkDays(resultados, plano);
        expect(reenviar).toHaveLength(1);
        expect(reenviar[0]?.entries).toHaveLength(1);
    });

    it('com duas falhas da mesma tarefa no dia, reenvia duas', () => {
        const plano = [dia('2024-05-01', [
            entrada(10, '09:00:00'),
            entrada(10, '13:00:00'),
            entrada(10, '16:00:00'),
            entrada(20, '17:00:00'),
        ])];
        const resultados = [
            resultado('2024-05-01', 10, false),
            resultado('2024-05-01', 10, true, 2),
            resultado('2024-05-01', 10, false),
            resultado('2024-05-01', 20, true, 4),
        ];

        const reenviar = buildRetryWorkDays(resultados, plano);
        expect(reenviar[0]?.entries.map(e => e.taskId)).toEqual([10, 10]);
    });

    it('a mesma tarefa em outro dia não conta como falha', () => {
        const plano = [
            dia('2024-05-01', [entrada(10, '09:00:00'), entrada(10, '13:00:00')]),
            dia('2024-05-02', [entrada(10, '09:00:00'), entrada(10, '13:00:00')]),
        ];
        const resultados = [
            resultado('2024-05-01', 10, true, 1),
            resultado('2024-05-01', 10, true, 2),
            resultado('2024-05-02', 10, false),
            resultado('2024-05-02', 10, true, 4),
        ];

        const reenviar = buildRetryWorkDays(resultados, plano);
        expect(reenviar.map(d => [d.date, d.entries.length])).toEqual([['2024-05-02', 1]]);
    });

    it('ignora falhas sem correspondência no plano', () => {
        const plano = [dia('2024-05-01', [entrada(10, '09:00:00')])];
        expect(buildRetryWorkDays([resultado('2024-06-01', 99, false)], plano)).toEqual([]);
    });

    it('não muta o plano original', () => {
        const plano = [dia('2024-05-01', [entrada(10, '09:00:00'), entrada(20, '10:00:00')])];
        const copia = structuredClone(plano);
        buildRetryWorkDays([resultado('2024-05-01', 20, false)], plano);
        expect(plano).toEqual(copia);
    });
});

describe('failedDeleteIds', () => {
    const exclusao = (entryId: number, success: boolean): DeleteTimeEntryResult => ({
        entryId,
        success,
        message: success ? 'removida' : 'erro'
    });

    it('devolve só os IDs que falharam', () => {
        const ids = failedDeleteIds([exclusao(1, true), exclusao(2, false), exclusao(3, true), exclusao(4, false)]);
        expect(ids).toEqual([2, 4]);
    });

    it('não repete IDs', () => {
        expect(failedDeleteIds([exclusao(2, false), exclusao(2, false)])).toEqual([2]);
    });

    it('nunca reenvia um ID que já foi excluído no mesmo lote', () => {
        expect(failedDeleteIds([exclusao(5, false), exclusao(5, true)])).toEqual([]);
    });

    it('lista vazia quando nada falhou', () => {
        expect(failedDeleteIds([exclusao(1, true)])).toEqual([]);
        expect(failedDeleteIds([])).toEqual([]);
    });
});
