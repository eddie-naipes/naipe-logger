import {beforeEach, describe, expect, it, vi} from 'vitest';
import {render, screen, waitFor, within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {DeleteMultipleTimeEntries, GetTimeEntriesForPeriodV2} from '@wailsjs/go/backend/App';
import type {TimeEntryReport} from '../types/backend';
import TimeEntryManager from './TimeEntryManager';

vi.mock('@wailsjs/go/backend/App', () => ({
    DeleteMultipleTimeEntries: vi.fn(),
    GetTimeEntriesForPeriodV2: vi.fn(),
    UpdateTimeEntry: vi.fn(),
}));

vi.mock('react-toastify', () => ({
    toast: {success: vi.fn(), warning: vi.fn(), error: vi.fn(), info: vi.fn()},
}));

const entrada = (id: number, taskName: string): TimeEntryReport => ({
    id,
    projectId: 1,
    projectName: 'Projeto',
    taskId: id * 10,
    taskName,
    tasklistId: 0,
    tasklistName: '',
    userId: 1,
    userFirstName: 'Ana',
    userLastName: 'Silva',
    date: '2024-05-01',
    hours: 1,
    minutes: 60,
    description: `Trabalho ${id}`,
    isBillable: true,
    isBilled: false,
    startTime: '09:00:00',
    endTime: '10:00:00',
});

describe('TimeEntryManager: reenviar exclusões que falharam', () => {
    beforeEach(() => {
        vi.spyOn(window, 'confirm').mockReturnValue(true);
        vi.spyOn(console, 'error').mockImplementation(() => {});
    });

    it('reenvia só os IDs que falharam e avisa quem abriu com a contagem real', async () => {
        // Antes da exclusão há três entradas; depois da primeira tentativa
        // sobra só a que falhou (2); depois do reenvio, nenhuma.
        vi.mocked(GetTimeEntriesForPeriodV2)
            .mockResolvedValueOnce([entrada(1, 'Alfa'), entrada(2, 'Beta'), entrada(3, 'Gama')] as never)
            .mockResolvedValueOnce([entrada(2, 'Beta')] as never)
            .mockResolvedValue([] as never);
        vi.mocked(DeleteMultipleTimeEntries)
            .mockResolvedValueOnce([
                {entryId: 1, success: true, message: 'removida'},
                {entryId: 2, success: false, message: 'erro 429'},
                {entryId: 3, success: true, message: 'removida'},
            ] as never)
            .mockResolvedValueOnce([{entryId: 2, success: true, message: 'removida'}] as never);

        const onEntriesChanged = vi.fn();
        render(<TimeEntryManager isOpen onClose={vi.fn()} onEntriesChanged={onEntriesChanged}/>);
        const dialog = await screen.findByRole('dialog', {name: 'Gerenciar Apontamentos de Horas'});

        await userEvent.click(await within(dialog).findByRole('checkbox', {name: 'Selecionar todas as entradas ativas visíveis'}));
        await userEvent.click(within(dialog).getByRole('button', {name: 'Deletar (3)'}));

        expect(DeleteMultipleTimeEntries).toHaveBeenNthCalledWith(1, [1, 2, 3]);
        expect(onEntriesChanged).toHaveBeenLastCalledWith({type: 'delete', succeeded: 2, failed: 1});

        const reenviar = await within(dialog).findByRole('button', {name: 'Reenviar 1 que falharam'});
        await userEvent.click(reenviar);

        expect(DeleteMultipleTimeEntries).toHaveBeenNthCalledWith(2, [2]);
        expect(onEntriesChanged).toHaveBeenLastCalledWith({type: 'delete', succeeded: 1, failed: 0});
        // Sem falhas, o painel de resultados some.
        await waitFor(() => expect(within(dialog).queryByRole('button', {name: /Reenviar/})).not.toBeInTheDocument());
    });

    it('não exclui entradas selecionadas que o filtro esconde', async () => {
        vi.mocked(GetTimeEntriesForPeriodV2).mockResolvedValue([entrada(1, 'Alfa'), entrada(2, 'Beta')] as never);
        vi.mocked(DeleteMultipleTimeEntries).mockResolvedValue([{entryId: 1, success: true, message: 'ok'}] as never);

        render(<TimeEntryManager isOpen onClose={vi.fn()}/>);
        const dialog = await screen.findByRole('dialog');

        await userEvent.click(await within(dialog).findByRole('checkbox', {name: 'Selecionar todas as entradas ativas visíveis'}));
        await userEvent.type(within(dialog).getByRole('textbox', {name: 'Filtrar por tarefa'}), 'Alfa');

        expect(within(dialog).getByRole('status')).toHaveTextContent('1 entrada(s) selecionada(s) estão ocultas');
        await userEvent.click(within(dialog).getByRole('button', {name: 'Deletar (1)'}));

        expect(DeleteMultipleTimeEntries).toHaveBeenCalledWith([1]);
    });
});
