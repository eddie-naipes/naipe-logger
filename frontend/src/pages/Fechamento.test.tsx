import {render, screen, waitFor, within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {MemoryRouter} from 'react-router';
import {beforeEach, describe, expect, it, vi} from 'vitest';
import {DeleteMultipleTimeEntries, IgnoreAuditIssue, RunMonthAudit} from '@wailsjs/go/backend/App';
import Fechamento from './Fechamento';
import {paraBinding, type AuditResult, type TimeEntryReport} from '../types/backend';

vi.mock('@wailsjs/go/backend/App', () => ({
    RunMonthAudit: vi.fn(),
    DeleteMultipleTimeEntries: vi.fn(),
    IgnoreAuditIssue: vi.fn(),
    UnignoreAuditIssue: vi.fn(),
    GetAuditSettings: vi.fn(),
    SaveAuditSettings: vi.fn(),
    UpdateTimeEntry: vi.fn(),
    GetGitIntegration: vi.fn().mockResolvedValue({enabled: false, repositories: [], authorEmail: ''}),
}));

vi.mock('react-toastify', () => ({
    toast: {success: vi.fn(), warning: vi.fn(), error: vi.fn(), info: vi.fn()},
}));

// Lançamentos no formato de GetTimeEntriesForPeriodV2 (fixture real
// backend/api/testdata/time_v2.json, anonimizada).
const lancamento = (id: number, extra: Partial<TimeEntryReport> = {}): TimeEntryReport => ({
    id,
    projectId: 1007,
    projectName: 'Nome ficticio 5',
    taskId: 1043,
    taskName: 'Nome ficticio 26',
    tasklistId: 1047,
    tasklistName: 'Nome ficticio 27',
    userId: 1004,
    userFirstName: 'Nome ficticio 1',
    userLastName: 'Nome ficticio 2',
    date: '2026-09-30',
    hours: 1,
    minutes: 60,
    description: 'Descricao ficticia 5',
    isBillable: true,
    isBilled: false,
    startTime: '10:15:00',
    endTime: '',
    ...extra,
});

const resultado: AuditResult = {
    issues: [
        {
            key: 'incomplete_day:2026-09-29',
            type: 'incomplete_day',
            severity: 'error',
            date: '2026-09-29',
            entryIds: [],
            entries: [],
            message: 'Nenhuma hora lançada (jornada de 8h)',
            action: 'complete_period',
            minutes: 0,
            missingMinutes: 480,
            ignored: false,
        },
        {
            key: 'duplicate:1046,1047,1048',
            type: 'duplicate',
            severity: 'error',
            date: '2026-09-30',
            entryIds: [1046, 1047, 1048],
            entries: [lancamento(1046), lancamento(1047), lancamento(1048)],
            message: '3 lançamentos iguais de 1h em "Nome ficticio 26"',
            action: 'delete_duplicates',
            deleteEntryIds: [1047, 1048],
            keepEntryId: 1046,
            minutes: 60,
            ignored: false,
        },
        {
            key: 'no_task:1050',
            type: 'no_task',
            severity: 'warning',
            date: '2026-09-30',
            entryIds: [1050],
            entries: [lancamento(1050, {taskId: 0, taskName: '', description: 'Atendimento'})],
            message: 'Lançamento sem tarefa (só no projeto)',
            action: 'edit',
            minutes: 60,
            ignored: false,
        },
        {
            key: 'generic_description:1051',
            type: 'generic_description',
            severity: 'warning',
            date: '2026-09-30',
            entryIds: [1051],
            entries: [lancamento(1051, {description: 'Reunião'})],
            message: 'Descrição genérica: "Reunião"',
            action: 'edit',
            minutes: 60,
            ignored: true,
        },
    ],
    counts: [
        {type: 'incomplete_day', severity: 'error', count: 1},
        {type: 'duplicate', severity: 'error', count: 1},
        {type: 'no_task', severity: 'warning', count: 1},
    ],
    summary: {
        year: 2026,
        month: 9,
        loggedMinutes: 9600,
        expectedMinutes: 10560,
        expectedToDateMinutes: 10560,
        loggedToDateMinutes: 9600,
        workingDays: 22,
        workingDaysToDate: 22,
        minutesPerDay: 480,
        dailyLimitMinutes: 600,
        entryCount: 40,
        errorCount: 2,
        warningCount: 1,
        ignoredCount: 1,
        ready: false,
    },
};

const vazio: AuditResult = {
    issues: [],
    counts: [],
    summary: {...resultado.summary, errorCount: 0, warningCount: 0, ignoredCount: 0, ready: true},
};

const renderPagina = () => render(
    <MemoryRouter initialEntries={[{pathname: '/fechamento', state: {month: '2026-09'}}]}>
        <Fechamento/>
    </MemoryRouter>
);

describe('Fechamento', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        vi.mocked(RunMonthAudit).mockResolvedValue(paraBinding(resultado));
        vi.mocked(IgnoreAuditIssue).mockResolvedValue();
    });

    it('audita o mês recebido e agrupa os problemas por tipo', async () => {
        renderPagina();

        expect(await screen.findByRole('heading', {name: /Dias incompletos/})).toBeInTheDocument();
        expect(RunMonthAudit).toHaveBeenCalledWith(2026, 9);
        expect(screen.getByRole('heading', {name: /Possíveis duplicatas/})).toBeInTheDocument();
        expect(screen.getByRole('heading', {name: /Lançamentos sem tarefa/})).toBeInTheDocument();
        expect(screen.getByText('2 erros a corrigir')).toBeInTheDocument();
        // Ignorados ficam escondidos até pedir.
        expect(screen.queryByRole('heading', {name: /Descrições genéricas/})).not.toBeInTheDocument();

        await userEvent.click(screen.getByLabelText(/Mostrar ignorados/));
        expect(screen.getByRole('heading', {name: /Descrições genéricas/})).toBeInTheDocument();
        expect(screen.getByRole('button', {name: /Reexibir/})).toBeInTheDocument();
    });

    it('apaga só as cópias excedentes da duplicata, após confirmar', async () => {
        vi.mocked(DeleteMultipleTimeEntries).mockResolvedValue([
            {entryId: 1047, success: true, message: ''},
            {entryId: 1048, success: true, message: ''},
        ]);
        renderPagina();

        const item = await screen.findByTestId('issue-duplicate:1046,1047,1048');
        await userEvent.click(within(item).getByRole('button', {name: /Apagar 2 cópias/}));

        const dialogo = await screen.findByRole('dialog', {name: /Apagar 2 lançamentos/});
        expect(within(within(dialogo).getByRole('list', {name: 'Lançamentos a apagar'})).getAllByRole('listitem')).toHaveLength(2);
        expect(within(dialogo).getByRole('list', {name: 'Lançamento mantido'})).toBeInTheDocument();
        expect(DeleteMultipleTimeEntries).not.toHaveBeenCalled();

        await userEvent.click(within(dialogo).getByRole('button', {name: 'Apagar'}));
        await waitFor(() => expect(DeleteMultipleTimeEntries).toHaveBeenCalledWith([1047, 1048]));
        await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    });

    it('ignora um problema pela chave e audita de novo', async () => {
        renderPagina();

        const item = await screen.findByTestId('issue-no_task:1050');
        await userEvent.click(within(item).getByRole('button', {name: /Ignorar/}));

        await waitFor(() => expect(IgnoreAuditIssue).toHaveBeenCalledWith('no_task:1050'));
        await waitFor(() => expect(RunMonthAudit).toHaveBeenCalledTimes(2));
    });

    it('comemora quando o mês está sem problemas', async () => {
        vi.mocked(RunMonthAudit).mockResolvedValue(paraBinding(vazio));
        renderPagina();

        expect(await screen.findByText('Tudo certo!')).toBeInTheDocument();
        expect(screen.getByText('Pronto para entregar')).toBeInTheDocument();
    });
});
