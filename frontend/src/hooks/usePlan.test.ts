import {beforeEach, describe, expect, it, vi} from 'vitest';
import {act, renderHook, waitFor} from '@testing-library/react';
import {toast} from 'react-toastify';
import {CheckPlanConflicts, CreateDistributionPlan, GetWorkingDays} from '@wailsjs/go/backend/App';
import type {DayConflict, Task, WorkDay} from '../types/backend';
import usePlan, {buildConflictWarning} from './usePlan';

vi.mock('@wailsjs/go/backend/App', () => ({
    CheckPlanConflicts: vi.fn(),
    CreateDistributionPlan: vi.fn(),
    GetWorkingDays: vi.fn(),
}));

vi.mock('react-toastify', () => ({
    toast: {success: vi.fn(), warning: vi.fn(), error: vi.fn(), info: vi.fn()},
}));

const tarefa = (taskId: number): Task => ({
    taskId,
    taskName: `Tarefa ${taskId}`,
    projectId: 1,
    projectName: 'Projeto',
    entries: [{minutes: 60, userId: 1, time: '09:00:00', description: 'Dev', isBillable: true}],
});

const plano = (date: string, taskId = 1): WorkDay => ({
    date,
    totalMin: 60,
    entries: [{taskId, entry: {minutes: 60, userId: 1, time: '09:00:00', description: 'Dev', isBillable: true}}],
});

const conflito: DayConflict = {
    date: '2024-05-01',
    existingMinutes: 120,
    existingEntries: 2,
    plannedMinutes: 60,
    sameTask: [{taskId: 1, taskName: 'Tarefa 1', existingMinutes: 60, plannedMinutes: 60}],
};

// Promise controlada pelo teste, para simular respostas fora de ordem.
const adiada = <T, >() => {
    let resolver!: (valor: T) => void;
    const promise = new Promise<T>((resolve) => {
        resolver = resolve;
    });
    return {promise, resolver};
};

const savedTasks = [tarefa(1), tarefa(2)];

describe('usePlan', () => {
    beforeEach(() => {
        vi.mocked(GetWorkingDays).mockResolvedValue(['2024-05-01']);
        vi.mocked(CreateDistributionPlan).mockImplementation((dias) =>
            Promise.resolve(dias.map(d => plano(d)) as never));
        vi.mocked(CheckPlanConflicts).mockResolvedValue([]);
    });

    it('gera o plano só com as tarefas selecionadas e verifica conflitos', async () => {
        const {result} = renderHook(() => usePlan(savedTasks));

        await act(() => result.current.generatePlan({start: '2024-05-01', end: '2024-05-01', taskIds: [2]}));

        expect(GetWorkingDays).toHaveBeenCalledWith('2024-05-01', '2024-05-01');
        const [, tarefasEnviadas] = vi.mocked(CreateDistributionPlan).mock.calls[0]!;
        expect(tarefasEnviadas.map(t => t.taskId)).toEqual([2]);
        expect(result.current.workDays).toEqual([plano('2024-05-01')]);
        expect(CheckPlanConflicts).toHaveBeenCalledWith([plano('2024-05-01')]);
        expect(result.current.isGenerating).toBe(false);
        expect(result.current.conflicts).toEqual([]);
    });

    it('sem tarefas selecionadas não chama o backend', async () => {
        const {result} = renderHook(() => usePlan(savedTasks));
        await act(() => result.current.generatePlan({start: '2024-05-01', end: '2024-05-01', taskIds: []}));

        expect(GetWorkingDays).not.toHaveBeenCalled();
        expect(toast.warning).toHaveBeenCalledWith('Selecione pelo menos uma tarefa para lançar horas.');
    });

    it('rejeita período invertido', async () => {
        const {result} = renderHook(() => usePlan(savedTasks));
        await act(() => result.current.generatePlan({start: '2024-05-10', end: '2024-05-01', taskIds: [1]}));

        expect(GetWorkingDays).not.toHaveBeenCalled();
        expect(toast.warning).toHaveBeenCalledWith('A data inicial deve ser anterior ou igual à data final.');
    });

    it('expõe os conflitos encontrados', async () => {
        vi.mocked(CheckPlanConflicts).mockResolvedValue([conflito] as never);
        const {result} = renderHook(() => usePlan(savedTasks));

        await act(() => result.current.generatePlan({start: '2024-05-01', end: '2024-05-01', taskIds: [1]}));

        expect(result.current.conflicts).toEqual([conflito]);
        expect(toast.warning).toHaveBeenCalledWith('1 dia(s) do plano já possuem lançamentos. Revise antes de enviar.');
    });

    it('marca a verificação como falha quando CheckPlanConflicts dá erro', async () => {
        vi.spyOn(console, 'error').mockImplementation(() => {});
        vi.mocked(CheckPlanConflicts).mockRejectedValue('sem conexão');
        const {result} = renderHook(() => usePlan(savedTasks));

        await act(() => result.current.generatePlan({start: '2024-05-01', end: '2024-05-01', taskIds: [1]}));

        expect(result.current.conflictCheckFailed).toBe(true);
        expect(result.current.isCheckingConflicts).toBe(false);
    });

    it('repassa o erro de geração para onError', async () => {
        vi.spyOn(console, 'error').mockImplementation(() => {});
        vi.mocked(GetWorkingDays).mockRejectedValue('token expirado');
        const onError = vi.fn();
        const {result} = renderHook(() => usePlan(savedTasks, onError));

        await act(() => result.current.generatePlan({start: '2024-05-01', end: '2024-05-01', taskIds: [1]}));

        expect(onError).toHaveBeenLastCalledWith('Erro ao gerar plano: token expirado');
        expect(result.current.workDays).toEqual([]);
        expect(result.current.isGenerating).toBe(false);
    });

    it('descarta a resposta de uma geração antiga que chega depois da nova', async () => {
        const lenta = adiada<string[]>();
        vi.mocked(GetWorkingDays)
            .mockReturnValueOnce(lenta.promise)
            .mockResolvedValueOnce(['2024-05-02']);

        const {result} = renderHook(() => usePlan(savedTasks));

        let primeira!: Promise<void>;
        act(() => {
            primeira = result.current.generatePlan({start: '2024-05-01', end: '2024-05-01', taskIds: [1]});
        });
        await act(() => result.current.generatePlan({start: '2024-05-02', end: '2024-05-02', taskIds: [1]}));
        expect(result.current.workDays.map(d => d.date)).toEqual(['2024-05-02']);

        // A primeira geração termina por último e não pode sobrescrever o plano.
        await act(async () => {
            lenta.resolver(['2024-05-01']);
            await primeira;
        });
        expect(result.current.workDays.map(d => d.date)).toEqual(['2024-05-02']);
        expect(CreateDistributionPlan).toHaveBeenCalledTimes(1);
    });

    it('usa as tarefas mais recentes (sem closure velha)', async () => {
        const {result, rerender} = renderHook(({tarefas}) => usePlan(tarefas), {
            initialProps: {tarefas: [tarefa(1)]},
        });
        rerender({tarefas: [tarefa(1), tarefa(3)]});

        await act(() => result.current.generatePlan({start: '2024-05-01', end: '2024-05-01', taskIds: [3]}));

        await waitFor(() => expect(CreateDistributionPlan).toHaveBeenCalled());
        const [, tarefasEnviadas] = vi.mocked(CreateDistributionPlan).mock.calls[0]!;
        expect(tarefasEnviadas.map(t => t.taskId)).toEqual([3]);
    });
});

describe('buildConflictWarning', () => {
    it('lista os dias, as horas existentes e as tarefas repetidas', () => {
        const aviso = buildConflictWarning([conflito]);
        expect(aviso).toContain('1 dia(s) do plano já possuem tempo lançado');
        expect(aviso).toContain('• 2024-05-01: já tem 2.0h em 2 entrada(s) — 1 na(s) MESMA(S) tarefa(s) do plano');
        expect(aviso).toContain('DUPLICAR');
    });
});
