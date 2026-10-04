import {beforeEach, describe, expect, it, vi} from 'vitest';
import {render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {toast} from 'react-toastify';
import {GetNotificationStatus, GetReminderSettings, SaveReminderSettings, SendTestReminder} from '@wailsjs/go/backend/App';
import RemindersSection from './RemindersSection';

vi.mock('@wailsjs/go/backend/App', () => ({
    GetReminderSettings: vi.fn(),
    SaveReminderSettings: vi.fn(),
    GetNotificationStatus: vi.fn(),
    SendTestReminder: vi.fn(),
}));

vi.mock('react-toastify', () => ({
    toast: {success: vi.fn(), warning: vi.fn(), error: vi.fn(), info: vi.fn()},
}));

const padrao = {enabled: true, dailyTime: '18:00', workDaysOnly: true, monthEndEnabled: true, monthEndDays: 2};

describe('RemindersSection', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        vi.mocked(GetReminderSettings).mockResolvedValue(padrao);
        vi.mocked(GetNotificationStatus).mockResolvedValue({available: true});
        vi.mocked(SaveReminderSettings).mockResolvedValue();
        vi.mocked(SendTestReminder).mockResolvedValue();
    });

    it('salva o horário e as opções alteradas', async () => {
        render(<RemindersSection/>);
        const horario = await screen.findByLabelText('Horário do lembrete diário');
        await userEvent.clear(horario);
        await userEvent.type(horario, '17:30');
        await userEvent.click(screen.getByLabelText(/Lembrete de fim de mês/));
        await userEvent.click(screen.getByRole('button', {name: /Salvar lembretes/}));

        await waitFor(() => expect(SaveReminderSettings).toHaveBeenCalledWith(
            {...padrao, dailyTime: '17:30', monthEndEnabled: false}));
        expect(toast.success).toHaveBeenCalled();
    });

    it('testar lembrete envia a notificação só ao clicar', async () => {
        render(<RemindersSection/>);
        await screen.findByText('Lembretes');
        expect(SendTestReminder).not.toHaveBeenCalled();
        await userEvent.click(screen.getByRole('button', {name: /Testar lembrete/}));
        expect(SendTestReminder).toHaveBeenCalledTimes(1);
    });

    it('avisa quando as notificações não estão disponíveis', async () => {
        vi.mocked(GetNotificationStatus).mockResolvedValue({available: false, error: 'sem D-Bus'});
        render(<RemindersSection/>);
        expect(await screen.findByRole('alert')).toHaveTextContent('sem D-Bus');
    });
});
