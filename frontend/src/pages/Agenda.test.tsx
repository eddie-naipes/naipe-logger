import {render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {beforeEach, describe, expect, it, vi} from 'vitest';
import {
    CheckPlanConflicts,
    DeleteMultipleTimeEntries,
    GetAgendaFile,
    GetAgendaSettings,
    GetSavedTasks,
    ListAgendaCalendars,
    LogMultipleTimes,
    MarkAgendaImported,
    PlanAgenda,
    UnmarkAgendaImported
} from '@wailsjs/go/backend/App';
import Agenda from './Agenda';
import type {AgendaItem} from '../utils/agenda';

vi.mock('@wailsjs/go/backend/App', () => ({
    GetAgendaSettings: vi.fn(),
    SaveAgendaSettings: vi.fn(),
    GetSavedTasks: vi.fn(),
    PlanAgenda: vi.fn(),
    MarkAgendaImported: vi.fn(),
    UnmarkAgendaImported: vi.fn(),
    ListAgendaCalendars: vi.fn(),
    GetAgendaFile: vi.fn(),
    AddAgendaCalendar: vi.fn(),
    RemoveAgendaCalendar: vi.fn(),
    ImportAgendaFile: vi.fn(),
    ClearAgendaFile: vi.fn(),
    CheckPlanConflicts: vi.fn(),
    LogMultipleTimes: vi.fn(),
    DeleteMultipleTimeEntries: vi.fn(),
}));

vi.mock('react-toastify', () => ({
    toast: {success: vi.fn(), warning: vi.fn(), error: vi.fn(), info: vi.fn(), dismiss: vi.fn()},
}));

const tarefa = {taskId: 10, taskName: 'Cerimônias', projectId: 1, projectName: 'Projeto X'};
const semTarefa = {taskId: 0, taskName: '', projectId: 0, projectName: ''};

const item = (parcial: Partial<AgendaItem>): AgendaItem => ({
    key: 'k', source: 'Trabalho', title: 'Daily', date: '2026-10-05', startTime: '09:00', endTime: '09:15',
    minutes: 15, status: 'mapped', reason: 'Regra 1: daily', ruleIndex: 0, task: tarefa, description: 'Daily',
    billable: true, ...parcial
});

describe('Agenda', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        vi.spyOn(window, 'confirm').mockReturnValue(true);
        vi.mocked(GetAgendaSettings).mockResolvedValue({
            calendars: [], rules: [], defaultTask: semTarefa, ignoreWords: [], minMinutes: 5,
            rounding: 'exact', userEmail: '', billable: true, includeTransparent: false
        } as never);
        vi.mocked(GetSavedTasks).mockResolvedValue([{...tarefa, entries: []}] as never);
        vi.mocked(ListAgendaCalendars).mockResolvedValue([]);
        vi.mocked(GetAgendaFile).mockResolvedValue(null as never);
        vi.mocked(CheckPlanConflicts).mockResolvedValue([]);
        vi.mocked(MarkAgendaImported).mockResolvedValue();
        vi.mocked(UnmarkAgendaImported).mockResolvedValue();
        vi.mocked(PlanAgenda).mockResolvedValue({
            items: [
                item({key: 'daily|1'}),
                item({key: 'cliente', title: 'Reunião com cliente', startTime: '10:00', endTime: '11:00', minutes: 60,
                    status: 'unmapped', reason: 'Nenhuma regra', ruleIndex: -1, task: semTarefa, description: 'Reunião com cliente'}),
                item({key: 'almoco', title: 'Almoço', status: 'ignored', reason: 'Contém a palavra ignorada', task: semTarefa}),
            ],
            warnings: [],
            sources: ['Trabalho']
        } as never);
    });

    it('lança os eventos mapeados com o horário real, marca como importados e desmarca ao desfazer', async () => {
        vi.mocked(LogMultipleTimes).mockResolvedValue([
            {success: true, message: 'ok', date: '2026-10-05', taskId: 10, entryId: 501},
        ]);
        vi.mocked(DeleteMultipleTimeEntries).mockResolvedValue([{success: true, entryId: 501, message: ''}] as never);

        render(<Agenda/>);
        await userEvent.click(await screen.findByRole('button', {name: /Buscar eventos/}));

        expect(await screen.findByText('Reunião com cliente')).toBeInTheDocument();
        expect(screen.getByText('Contém a palavra ignorada')).toBeInTheDocument();
        expect(screen.getByLabelText('Incluir Almoço')).toBeDisabled();
        // Sem tarefa escolhida, o evento sem regra não pode entrar.
        expect(screen.getByLabelText('Incluir Reunião com cliente')).toBeDisabled();

        await userEvent.click(screen.getByRole('button', {name: /Gerar prévia \(1\)/}));
        await waitFor(() => expect(CheckPlanConflicts).toHaveBeenCalled());

        await userEvent.click(await screen.findByRole('button', {name: /Executar Lançamento/}));
        await waitFor(() => expect(LogMultipleTimes).toHaveBeenCalledWith([{
            date: '2026-10-05',
            totalMin: 15,
            entries: [{taskId: 10, entry: {minutes: 15, userId: 0, time: '09:00:00', description: 'Daily', isBillable: true, date: '2026-10-05'}}]
        }]));
        await waitFor(() => expect(MarkAgendaImported).toHaveBeenCalledWith([
            {key: 'daily|1', date: '2026-10-05', taskId: 10, entryId: 501, importedAt: ''}
        ]));
        expect(await screen.findByText('Já lançado')).toBeInTheDocument();

        await userEvent.click(screen.getByRole('button', {name: /Desfazer/}));
        await waitFor(() => expect(DeleteMultipleTimeEntries).toHaveBeenCalledWith([501]));
        await waitFor(() => expect(UnmarkAgendaImported).toHaveBeenCalledWith(['daily|1']));
    });

    it('permite escolher a tarefa de um evento sem regra', async () => {
        render(<Agenda/>);
        await userEvent.click(await screen.findByRole('button', {name: /Buscar eventos/}));
        await userEvent.selectOptions(await screen.findByLabelText('Tarefa para Reunião com cliente'), '10');
        expect(screen.getByLabelText('Incluir Reunião com cliente')).toBeChecked();
        expect(screen.getByRole('button', {name: /Gerar prévia \(2\)/})).toBeEnabled();
    });
});
