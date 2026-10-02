import {beforeEach, describe, expect, it, vi} from 'vitest';
import {act, renderHook} from '@testing-library/react';
import {toast} from 'react-toastify';
import {DeleteMultipleTimeEntries, LogMultipleTimes} from '@wailsjs/go/backend/App';
import type {EntryTask, TimeLogResult, WorkDay} from '../types/backend';
import useBatchSubmit, {type UseBatchSubmitOptions} from './useBatchSubmit';

vi.mock('@wailsjs/go/backend/App', () => ({
    LogMultipleTimes: vi.fn(),
    DeleteMultipleTimeEntries: vi.fn(),
}));

vi.mock('react-toastify', () => ({
    toast: {success: vi.fn(), warning: vi.fn(), error: vi.fn(), info: vi.fn(() => 'toast-id'), dismiss: vi.fn()},
}));

const entrada = (taskId: number, time: string): EntryTask => ({
    taskId,
    entry: {minutes: 60, userId: 1, time, description: 'Dev', isBillable: true},
});

// Dia 01: tarefa 10 duas vezes e tarefa 20 uma vez. Dia 02: tarefa 10.
const plano: WorkDay[] = [
    {date: '2024-05-01', totalMin: 180, entries: [entrada(10, '09:00:00'), entrada(10, '13:00:00'), entrada(20, '15:00:00')]},
    {date: '2024-05-02', totalMin: 60, entries: [entrada(10, '09:00:00')]},
];

const ok = (date: string, taskId: number, entryId: number): TimeLogResult =>
    ({date, taskId, entryId, success: true, message: 'ok'});
const falha = (date: string, taskId: number): TimeLogResult =>
    ({date, taskId, entryId: 0, success: false, message: 'erro 500'});

const opcoes = (extra: Partial<UseBatchSubmitOptions> = {}): UseBatchSubmitOptions => ({
    workDays: plano,
    conflicts: [],
    isCheckingConflicts: false,
    conflictCheckFailed: false,
    checkConflicts: vi.fn(() => Promise.resolve()),
    refreshCalendar: vi.fn(),
    reloadCalendar: vi.fn(),
    onError: vi.fn(),
    ...extra,
});

describe('useBatchSubmit', () => {
    beforeEach(() => {
        vi.spyOn(window, 'confirm').mockReturnValue(true);
        vi.spyOn(console, 'error').mockImplementation(() => {});
    });

    it('envia o plano e separa sucessos, falhas e o que pode ser desfeito', async () => {
        vi.mocked(LogMultipleTimes).mockResolvedValue([
            ok('2024-05-01', 10, 101),
            falha('2024-05-01', 10),
            ok('2024-05-01', 20, 0), // criada, mas sem ID: não dá para desfazer
            ok('2024-05-02', 10, 104),
        ] as never);
        const checkConflicts = vi.fn(() => Promise.resolve());
        const props = opcoes({checkConflicts});
        const {result} = renderHook(() => useBatchSubmit(props));

        await act(() => result.current.submitPlan());

        expect(LogMultipleTimes).toHaveBeenCalledWith(plano);
        expect(result.current.showResults).toBe(true);
        expect(result.current.results).toHaveLength(4);
        expect(result.current.failedEntries).toHaveLength(1);
        expect(result.current.undoableEntries.map(r => r.entryId)).toEqual([101, 104]);
        expect(result.current.notUndoableCount).toBe(1);
        // Houve sucessos: o plano é reavaliado para acusar duplicata num reenvio.
        expect(checkConflicts).toHaveBeenCalledWith(plano);
        expect(toast.warning).toHaveBeenCalledWith('3 lançamentos com sucesso e 1 falhas.');
    });

    it('não envia sem confirmação quando há conflitos', async () => {
        const confirmar = vi.spyOn(window, 'confirm').mockReturnValue(false);
        const props = opcoes({
            conflicts: [{date: '2024-05-01', existingMinutes: 60, existingEntries: 1, plannedMinutes: 180, sameTask: []}],
        });
        const {result} = renderHook(() => useBatchSubmit(props));

        await act(() => result.current.submitPlan());

        expect(confirmar).toHaveBeenCalled();
        expect(LogMultipleTimes).not.toHaveBeenCalled();
    });

    it('não envia enquanto a verificação de conflitos não termina', async () => {
        const {result} = renderHook(() => useBatchSubmit(opcoes({isCheckingConflicts: true})));
        await act(() => result.current.submitPlan());
        expect(LogMultipleTimes).not.toHaveBeenCalled();
    });

    it('reenvia só as falhas e nunca o que já deu certo', async () => {
        vi.mocked(LogMultipleTimes)
            .mockResolvedValueOnce([
                ok('2024-05-01', 10, 101),
                falha('2024-05-01', 10),
                falha('2024-05-01', 20),
                ok('2024-05-02', 10, 104),
            ] as never)
            .mockResolvedValueOnce([ok('2024-05-01', 10, 102), ok('2024-05-01', 20, 103)] as never);
        const {result} = renderHook(() => useBatchSubmit(opcoes()));

        await act(() => result.current.submitPlan());
        await act(() => result.current.retryFailed());

        const reenviado = vi.mocked(LogMultipleTimes).mock.calls[1]![0];
        // Do dia 01 vai uma entrada da tarefa 10 (das duas, só uma falhou) e a
        // da tarefa 20; o dia 02 deu certo e fica de fora.
        expect(reenviado).toHaveLength(1);
        expect(reenviado[0]!.date).toBe('2024-05-01');
        expect(reenviado[0]!.entries.map(e => e.taskId)).toEqual([10, 20]);

        expect(result.current.failedEntries).toEqual([]);
        // Os IDs anteriores continuam valendo para o desfazer.
        expect(result.current.undoableEntries.map(r => r.entryId).sort()).toEqual([101, 102, 103, 104]);
        expect(toast.success).toHaveBeenCalledWith('2 lançamento(s) reenviado(s) com sucesso!');
    });

    it('desfaz apagando só as entradas com ID e mantém as que não saíram', async () => {
        vi.mocked(LogMultipleTimes).mockResolvedValue([
            ok('2024-05-01', 10, 101),
            ok('2024-05-01', 10, 102),
            ok('2024-05-01', 20, 0),
        ] as never);
        vi.mocked(DeleteMultipleTimeEntries).mockResolvedValue([
            {entryId: 101, success: true, message: 'ok'},
            {entryId: 102, success: false, message: 'erro'},
        ] as never);
        const reloadCalendar = vi.fn();
        const props = opcoes({reloadCalendar});
        const {result} = renderHook(() => useBatchSubmit(props));

        await act(() => result.current.submitPlan());
        await act(() => result.current.undoBatch());

        expect(DeleteMultipleTimeEntries).toHaveBeenCalledWith([101, 102]);
        // A 101 saiu da lista; a 102 (falhou) e a sem ID continuam visíveis.
        expect(result.current.results.map(r => r.entryId)).toEqual([102, 0]);
        expect(result.current.showResults).toBe(true);
        expect(reloadCalendar).toHaveBeenCalled();
    });

    it('desfazer tudo com sucesso limpa o painel', async () => {
        vi.mocked(LogMultipleTimes).mockResolvedValue([ok('2024-05-01', 10, 101)] as never);
        vi.mocked(DeleteMultipleTimeEntries).mockResolvedValue([{entryId: 101, success: true, message: 'ok'}] as never);
        const {result} = renderHook(() => useBatchSubmit(opcoes()));

        await act(() => result.current.submitPlan());
        await act(() => result.current.undoBatch());

        expect(result.current.results).toEqual([]);
        expect(result.current.showResults).toBe(false);
    });

    it('informa o erro quando o envio falha por completo', async () => {
        vi.mocked(LogMultipleTimes).mockRejectedValue('sem conexão');
        const props = opcoes();
        const {result} = renderHook(() => useBatchSubmit(props));

        await act(() => result.current.submitPlan());

        expect(props.onError).toHaveBeenLastCalledWith('Erro ao lançar horas: sem conexão');
        expect(result.current.isSubmitting).toBe(false);
        expect(result.current.results).toEqual([]);
    });
});
