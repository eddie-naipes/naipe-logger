import {beforeEach, describe, expect, it, vi} from 'vitest';
import {render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {toast} from 'react-toastify';
import {GetAutostart, SetAutostart} from '@wailsjs/go/backend/App';
import AutostartSection from './AutostartSection';

vi.mock('@wailsjs/go/backend/App', () => ({
    GetAutostart: vi.fn(),
    SetAutostart: vi.fn(),
}));

vi.mock('react-toastify', () => ({
    toast: {success: vi.fn(), warning: vi.fn(), error: vi.fn(), info: vi.fn()},
}));

describe('AutostartSection', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('liga o início com o sistema', async () => {
        vi.mocked(GetAutostart).mockResolvedValue({enabled: false, supported: true, reason: ''});
        vi.mocked(SetAutostart).mockResolvedValue({enabled: true, supported: true, reason: ''});
        render(<AutostartSection/>);

        const toggle = await screen.findByLabelText('Iniciar com o sistema (minimizado)');
        await userEvent.click(toggle);

        await waitFor(() => expect(SetAutostart).toHaveBeenCalledWith(true));
        expect(toast.success).toHaveBeenCalled();
        expect(screen.getByLabelText('Iniciar com o sistema (minimizado)')).toBeChecked();
        expect(screen.getByText(/lembretes e o cronômetro só funcionam com o app aberto/)).toBeInTheDocument();
    });

    it('desabilita a opção e explica o motivo na versão de desenvolvimento', async () => {
        vi.mocked(GetAutostart).mockResolvedValue({
            enabled: false,
            supported: false,
            reason: 'Indisponível na versão de desenvolvimento (wails dev)'
        });
        render(<AutostartSection/>);

        expect(await screen.findByLabelText('Iniciar com o sistema (minimizado)')).toBeDisabled();
        expect(screen.getByText(/versão de desenvolvimento/)).toBeInTheDocument();
    });
});
