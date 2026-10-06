import {render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {beforeEach, describe, expect, it, vi} from 'vitest';
import {GetBrazilianStates, GetWorkCalendarSettings, SaveWorkCalendarSettings} from '@wailsjs/go/backend/App';
import {toast} from 'react-toastify';
import WorkCalendarSettings from './WorkCalendarSettings';
import {validateAbsence, validateCustomHoliday} from './calendarValidation';
import type {BrazilianState, CalendarSettings} from '../../types/backend';
import {paraBinding} from '../../types/backend';

vi.mock('@wailsjs/go/backend/App', () => ({
    GetBrazilianStates: vi.fn(),
    GetWorkCalendarSettings: vi.fn(),
    SaveWorkCalendarSettings: vi.fn(),
}));

vi.mock('react-toastify', () => ({
    toast: {success: vi.fn(), warning: vi.fn(), error: vi.fn(), info: vi.fn()},
}));

const estados: BrazilianState[] = [
    {uf: 'MG', name: 'Minas Gerais', holidays: []},
    {uf: 'SP', name: 'São Paulo', holidays: [{monthDay: '07-09', name: 'Revolução Constitucionalista de 1932', source: 'Lei estadual SP 9.497/1997'}]},
];

const vazio: CalendarSettings = {uf: '', disabledStateHolidays: [], customHolidays: [], absences: []};

describe('WorkCalendarSettings', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        vi.mocked(GetBrazilianStates).mockResolvedValue(paraBinding(estados));
        vi.mocked(GetWorkCalendarSettings).mockResolvedValue(paraBinding(vazio));
        vi.mocked(SaveWorkCalendarSettings).mockImplementation(s => Promise.resolve(s));
    });

    it('mostra os feriados da UF escolhida e salva com o feriado desmarcado', async () => {
        const user = userEvent.setup();
        const onSaved = vi.fn();
        render(<WorkCalendarSettings onSaved={onSaved}/>);

        await user.selectOptions(await screen.findByLabelText('Estado (feriados estaduais)'), 'SP');
        const checkbox = screen.getByRole('checkbox', {name: /Revolução Constitucionalista/});
        expect(checkbox).toBeChecked();
        await user.click(checkbox);

        await user.click(screen.getByRole('button', {name: 'Salvar calendário'}));

        await waitFor(() => expect(SaveWorkCalendarSettings).toHaveBeenCalledWith(
            expect.objectContaining({uf: 'SP', disabledStateHolidays: ['07-09']})
        ));
        expect(onSaved).toHaveBeenCalled();
        expect(toast.success).toHaveBeenCalled();
    });

    it('avisa quando a UF não tem feriado estadual fixo', async () => {
        const user = userEvent.setup();
        render(<WorkCalendarSettings/>);
        await user.selectOptions(await screen.findByLabelText('Estado (feriados estaduais)'), 'MG');
        expect(screen.getByText(/Não há feriado estadual de data fixa/)).toBeInTheDocument();
    });

    it('valida e adiciona feriado municipal recorrente', async () => {
        const user = userEvent.setup();
        render(<WorkCalendarSettings/>);
        await screen.findByLabelText('Estado (feriados estaduais)');

        await user.click(screen.getByRole('button', {name: 'Adicionar feriado'}));
        expect(screen.getByRole('alert')).toHaveTextContent('Informe uma data válida.');

        await user.type(screen.getByLabelText('Data'), '2026-01-25');
        await user.click(screen.getByRole('button', {name: 'Adicionar feriado'}));
        expect(screen.getByRole('alert')).toHaveTextContent('Informe o nome do feriado.');

        await user.type(screen.getByLabelText('Nome'), 'Aniversário de São Paulo');
        await user.click(screen.getByLabelText('Repete todo ano'));
        await user.click(screen.getByRole('button', {name: 'Adicionar feriado'}));

        expect(screen.queryByRole('alert')).not.toBeInTheDocument();
        expect(screen.getByRole('list', {name: 'Feriados personalizados'})).toHaveTextContent('25/01 — Aniversário de São Paulo');
        expect(screen.getByRole('button', {name: 'Salvar calendário'})).toBeEnabled();
    });

    it('recusa férias com fim antes do início', async () => {
        const user = userEvent.setup();
        render(<WorkCalendarSettings/>);
        await screen.findByLabelText('Estado (feriados estaduais)');

        await user.type(screen.getByLabelText('Início'), '2026-07-20');
        await user.type(screen.getByLabelText('Fim'), '2026-07-10');
        await user.click(screen.getByRole('button', {name: 'Adicionar ausência'}));
        expect(screen.getByRole('alert')).toHaveTextContent('A data final não pode ser anterior à inicial.');
    });

    it('mostra o erro do backend ao salvar', async () => {
        vi.mocked(SaveWorkCalendarSettings).mockRejectedValue('UF desconhecida: "XX"');
        const user = userEvent.setup();
        render(<WorkCalendarSettings/>);
        await user.selectOptions(await screen.findByLabelText('Estado (feriados estaduais)'), 'SP');
        await user.click(screen.getByRole('button', {name: 'Salvar calendário'}));
        await waitFor(() => expect(toast.error).toHaveBeenCalledWith(expect.stringContaining('UF desconhecida')));
    });
});

describe('validações do calendário', () => {
    it('detecta feriado repetido considerando a recorrência', () => {
        const existentes = [{date: '2025-01-25', name: 'Cidade', type: 'municipal', recurring: true}];
        expect(validateCustomHoliday({date: '2027-01-25', name: 'Outro', type: 'ponte', recurring: false}, existentes))
            .toMatch(/Já existe/);
        expect(validateCustomHoliday({date: '2027-01-26', name: 'Outro', type: 'ponte', recurring: false}, existentes))
            .toBeNull();
        expect(validateCustomHoliday({date: '2026-02-30', name: 'x', type: 'outro', recurring: false}, []))
            .toMatch(/data válida/);
    });

    it('aceita férias atravessando o ano e recusa períodos longos demais', () => {
        expect(validateAbsence({start: '2026-12-21', end: '2027-01-08', description: ''})).toBeNull();
        expect(validateAbsence({start: '2026-01-01', end: '2029-01-01', description: ''})).toMatch(/2 anos/);
    });
});
