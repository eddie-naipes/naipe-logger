import {render, screen, waitFor, within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {beforeEach, describe, expect, it, vi} from 'vitest';
import {DownloadTimeReport, ExportTimeReportCSV, GetTimeReportSummary} from '@wailsjs/go/backend/App';
import {toast} from 'react-toastify';
import Reports from './Reports';
import {paraBinding, type TimeReport} from '../types/backend';

vi.mock('@wailsjs/go/backend/App', () => ({
    GetTimeReportSummary: vi.fn(),
    ExportTimeReportCSV: vi.fn(),
    DownloadTimeReport: vi.fn(),
    OpenDirectoryPath: vi.fn(),
}));

vi.mock('react-toastify', () => ({
    toast: {success: vi.fn(), warning: vi.fn(), error: vi.fn(), info: vi.fn()},
}));

const relatorio: TimeReport = {
    startDate: '2026-06-01',
    endDate: '2026-06-03',
    totalMinutes: 900,
    billableMinutes: 600,
    nonBillableMinutes: 300,
    entryCount: 3,
    workingDays: 3,
    minutesPerDay: 480,
    expectedMinutes: 1440,
    balanceMinutes: -540,
    daysWithEntries: 2,
    workingDaysWithoutEntries: 1,
    byProject: [
        {projectId: 1, projectName: 'Alfa', minutes: 600, billableMinutes: 600, entryCount: 2},
        {projectId: 2, projectName: 'Beta', minutes: 300, billableMinutes: 0, entryCount: 1},
    ],
    byTask: [
        {taskId: 10, taskName: 'Desenvolvimento', projectId: 1, projectName: 'Alfa', minutes: 600, billableMinutes: 600, entryCount: 2},
        {taskId: 20, taskName: 'Atendimento', projectId: 2, projectName: 'Beta', minutes: 300, billableMinutes: 0, entryCount: 1},
    ],
    byDay: [
        {date: '2026-06-01', minutes: 600, billableMinutes: 600, isWorkingDay: true, expectedMinutes: 480},
        {date: '2026-06-02', minutes: 300, billableMinutes: 0, isWorkingDay: true, expectedMinutes: 480},
        {date: '2026-06-03', minutes: 0, billableMinutes: 0, isWorkingDay: true, expectedMinutes: 480},
    ],
    byWeek: [{weekStart: '2026-06-01', weekEnd: '2026-06-03', minutes: 900, billableMinutes: 600, workingDays: 3, expectedMinutes: 1440}],
};

describe('Reports', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        vi.mocked(GetTimeReportSummary).mockResolvedValue(paraBinding(relatorio));
        vi.mocked(ExportTimeReportCSV).mockResolvedValue('C:\\Users\\eu\\TeamworkReports\\horas.csv');
        vi.mocked(DownloadTimeReport).mockResolvedValue('C:\\Users\\eu\\TeamworkReports\\r.pdf');
    });

    it('mostra totais, saldo e rankings do período', async () => {
        render(<Reports/>);

        expect(await screen.findByTestId('total-lancado')).toHaveTextContent('15h');
        expect(screen.getByText('67% cobrável')).toBeInTheDocument();
        expect(screen.getByText('−9h')).toBeInTheDocument();
        expect(screen.getByText('Faltam 9h')).toBeInTheDocument();

        const porProjeto = screen.getByRole('region', {name: 'Por projeto'});
        expect(within(porProjeto).getByText('Alfa')).toBeInTheDocument();
        expect(screen.getAllByRole('button', {name: /lançadas/}).length).toBe(3);
    });

    it('troca o atalho de período e consulta de novo', async () => {
        const user = userEvent.setup();
        render(<Reports/>);
        await screen.findByTestId('total-lancado');

        await user.click(screen.getByRole('button', {name: 'Mês passado'}));
        await waitFor(() => expect(GetTimeReportSummary).toHaveBeenCalledTimes(2));
        expect(screen.getByRole('button', {name: 'Mês passado'})).toHaveAttribute('aria-pressed', 'true');
    });

    it('valida o período personalizado sem chamar o backend', async () => {
        const user = userEvent.setup();
        render(<Reports/>);
        await screen.findByTestId('total-lancado');

        await user.click(screen.getByRole('button', {name: 'Personalizado'}));
        await user.clear(screen.getByLabelText('Início'));
        await user.type(screen.getByLabelText('Início'), '2026-12-31');

        expect(await screen.findByRole('alert')).toHaveTextContent('A data final não pode ser anterior à inicial.');
        expect(screen.getByRole('button', {name: 'CSV detalhado'})).toBeDisabled();
    });

    it('exporta o CSV detalhado e o resumido', async () => {
        const user = userEvent.setup();
        render(<Reports/>);
        await screen.findByTestId('total-lancado');

        await user.click(screen.getByRole('button', {name: 'CSV detalhado'}));
        await waitFor(() => expect(ExportTimeReportCSV).toHaveBeenCalledWith(expect.any(String), expect.any(String), true));
        await user.click(screen.getByRole('button', {name: 'CSV resumido'}));
        await waitFor(() => expect(ExportTimeReportCSV).toHaveBeenLastCalledWith(expect.any(String), expect.any(String), false));
        expect(toast.success).toHaveBeenCalledTimes(2);
    });

    it('mostra o erro da exportação', async () => {
        vi.mocked(DownloadTimeReport).mockRejectedValue('sem permissão');
        const user = userEvent.setup();
        render(<Reports/>);
        await screen.findByTestId('total-lancado');

        await user.click(screen.getByRole('button', {name: 'PDF'}));
        await waitFor(() => expect(toast.error).toHaveBeenCalledWith(expect.stringContaining('sem permissão')));
    });

    it('ordena a tabela por tarefa pelo cabeçalho', async () => {
        const user = userEvent.setup();
        render(<Reports/>);
        await screen.findByTestId('total-lancado');

        const tabela = screen.getByRole('region', {name: 'Detalhe por tarefa'});
        const primeiraTarefa = () => within(tabela).getAllByRole('row')[1];
        expect(primeiraTarefa()).toHaveTextContent('Desenvolvimento');

        await user.click(within(tabela).getByRole('button', {name: 'Tarefa'}));
        expect(primeiraTarefa()).toHaveTextContent('Atendimento');
        expect(within(tabela).getByRole('columnheader', {name: 'Tarefa'})).toHaveAttribute('aria-sort', 'ascending');
    });

    it('mostra o erro do backend ao carregar', async () => {
        vi.mocked(GetTimeReportSummary).mockRejectedValue('API não configurada');
        render(<Reports/>);
        expect(await screen.findByRole('alert')).toHaveTextContent('API não configurada');
    });
});
