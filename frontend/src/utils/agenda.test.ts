import {describe, expect, it} from 'vitest';
import {
    type AgendaItem,
    casarImportados,
    itensDoPlano,
    montarWorkDays,
    periodoAgenda,
    tarefasDoPlano
} from './agenda';

const item = (parcial: Partial<AgendaItem>): AgendaItem => ({
    key: 'k',
    source: 'Trabalho',
    title: 'Daily',
    date: '2026-10-05',
    startTime: '09:00',
    endTime: '09:15',
    minutes: 15,
    status: 'mapped',
    reason: '',
    ruleIndex: 0,
    task: {taskId: 10, taskName: 'Cerimônias', projectId: 1, projectName: 'Projeto X'},
    description: 'Daily',
    billable: true,
    ...parcial
});

describe('periodoAgenda', () => {
    // 2026-10-07 é uma quarta-feira.
    const hoje = new Date(2026, 9, 7, 15, 0);

    it('calcula hoje, ontem e esta semana (segunda até hoje)', () => {
        expect(periodoAgenda('hoje', hoje)).toEqual({start: '2026-10-07', end: '2026-10-07'});
        expect(periodoAgenda('ontem', hoje)).toEqual({start: '2026-10-06', end: '2026-10-06'});
        expect(periodoAgenda('semana', hoje)).toEqual({start: '2026-10-05', end: '2026-10-07'});
    });
});

describe('itensDoPlano', () => {
    it('leva só selecionados com tarefa, aplicando a tarefa escolhida na tela', () => {
        const items = [
            item({key: 'b', startTime: '10:00'}),
            item({key: 'a', startTime: '09:00'}),
            item({key: 'sem-regra', status: 'unmapped', task: {taskId: 0, taskName: '', projectId: 0, projectName: ''}}),
            item({key: 'sem-tarefa', status: 'unmapped', task: {taskId: 0, taskName: '', projectId: 0, projectName: ''}}),
            item({key: 'ignorado', status: 'ignored'}),
            item({key: 'lancado', status: 'imported'}),
            item({key: 'nao-selecionado'})
        ];
        const selecionados = new Set(['a', 'b', 'sem-regra', 'sem-tarefa', 'ignorado', 'lancado']);
        const escolhidas = new Map([['sem-regra', {taskId: 20, taskName: 'Suporte', projectId: 2, projectName: 'Y'}]]);

        const plano = itensDoPlano(items, selecionados, escolhidas);
        expect(plano.map(i => i.key)).toEqual(['a', 'sem-regra', 'b']);
        expect(plano[1]?.task.taskId).toBe(20);
    });
});

describe('montarWorkDays', () => {
    it('agrupa por dia com o horário real de cada evento', () => {
        const dias = montarWorkDays([
            item({key: 'a'}),
            item({key: 'b', startTime: '14:30', minutes: 60, description: 'Planejamento', billable: false}),
            item({key: 'c', date: '2026-10-06'})
        ]);
        expect(dias).toHaveLength(2);
        expect(dias[0]?.totalMin).toBe(75);
        expect(dias[0]?.entries[1]).toEqual({
            taskId: 10,
            entry: {minutes: 60, userId: 0, time: '14:30:00', description: 'Planejamento', isBillable: false, date: '2026-10-05'}
        });
    });

    it('lista as tarefas do plano sem repetir', () => {
        expect(tarefasDoPlano([item({key: 'a'}), item({key: 'b'})])).toHaveLength(1);
    });
});

describe('casarImportados', () => {
    it('casa sucessos por data e tarefa, na ordem do plano, e ignora falhas', () => {
        const plano = [
            item({key: 'a', startTime: '09:00'}),
            item({key: 'b', startTime: '15:00'}),
            item({key: 'c', date: '2026-10-06'})
        ];
        const registros = casarImportados([
            {success: true, date: '2026-10-05', taskId: 10, entryId: 101},
            {success: true, date: '2026-10-05', taskId: 10, entryId: 102},
            {success: false, date: '2026-10-06', taskId: 10, entryId: 0}
        ], plano);
        expect(registros.map(r => [r.key, r.entryId])).toEqual([['a', 101], ['b', 102]]);
    });
});
