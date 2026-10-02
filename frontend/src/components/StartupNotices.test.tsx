import {beforeEach, describe, expect, it, vi} from 'vitest';
import {render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {toast} from 'react-toastify';
import {CorruptedConfigBackups, GetLegacyInstall, RunLegacyUninstaller} from '@wailsjs/go/backend/App';
import type {LegacyInstall} from '../types/backend';
import StartupNotices, {LEGACY_IGNORADO_KEY} from './StartupNotices';

vi.mock('@wailsjs/go/backend/App', () => ({
    CorruptedConfigBackups: vi.fn(),
    GetLegacyInstall: vi.fn(),
    RunLegacyUninstaller: vi.fn(),
}));

vi.mock('react-toastify', () => ({
    toast: {success: vi.fn(), warning: vi.fn(), error: vi.fn(), info: vi.fn()},
}));

const semLegado: LegacyInstall = {found: false, displayName: '', installLocation: '', uninstallString: ''};
const legado: LegacyInstall = {
    found: true,
    displayName: 'Teamwork Logger',
    installLocation: 'C:\\Program Files\\Teamwork Logger',
    uninstallString: 'C:\\Program Files\\Teamwork Logger\\uninst.exe',
};

const BACKUP = 'C:\\Users\\eu\\.teamwork-logger\\config.json.corrompido-20261002-101500';

describe('StartupNotices', () => {
    beforeEach(() => {
        window.localStorage.clear();
        vi.mocked(CorruptedConfigBackups).mockResolvedValue([]);
        vi.mocked(GetLegacyInstall).mockResolvedValue(semLegado);
    });

    it('não mostra nada quando está tudo certo', async () => {
        render(<StartupNotices/>);
        await waitFor(() => expect(GetLegacyInstall).toHaveBeenCalled());
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });

    it('avisa da configuração corrompida e lista os backups', async () => {
        vi.mocked(CorruptedConfigBackups).mockResolvedValue([BACKUP]);
        render(<StartupNotices/>);

        const dialogo = await screen.findByRole('dialog', {name: 'Configuração recuperada'});
        expect(dialogo).toHaveTextContent('configuração padrão');
        expect(dialogo).toHaveTextContent(BACKUP);

        await userEvent.click(screen.getByRole('button', {name: 'Entendi'}));
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });

    it('trata lista nula de backups como vazia', async () => {
        vi.mocked(CorruptedConfigBackups).mockResolvedValue(null as unknown as string[]);
        render(<StartupNotices/>);
        await waitFor(() => expect(CorruptedConfigBackups).toHaveBeenCalled());
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });

    it('mostra a instalação antiga depois do aviso de configuração', async () => {
        vi.mocked(CorruptedConfigBackups).mockResolvedValue([BACKUP]);
        vi.mocked(GetLegacyInstall).mockResolvedValue(legado);
        render(<StartupNotices/>);

        await userEvent.click(await screen.findByRole('button', {name: 'Entendi'}));

        const dialogo = await screen.findByRole('dialog', {name: 'Versão antiga instalada'});
        expect(dialogo).toHaveTextContent(legado.installLocation);
        expect(dialogo).toHaveTextContent('administrador');
    });

    it('remove a versão antiga pelo desinstalador', async () => {
        vi.mocked(GetLegacyInstall).mockResolvedValue(legado);
        vi.mocked(RunLegacyUninstaller).mockResolvedValue();
        render(<StartupNotices/>);

        await userEvent.click(await screen.findByRole('button', {name: 'Remover versão antiga'}));

        expect(RunLegacyUninstaller).toHaveBeenCalledTimes(1);
        expect(toast.info).toHaveBeenCalled();
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });

    it('"não perguntar novamente" esconde o aviso nas próximas aberturas', async () => {
        vi.mocked(GetLegacyInstall).mockResolvedValue(legado);
        const {unmount} = render(<StartupNotices/>);

        await userEvent.click(await screen.findByRole('checkbox', {name: 'Não perguntar novamente'}));
        await userEvent.click(screen.getByRole('button', {name: 'Lembrar depois'}));
        expect(window.localStorage.getItem(LEGACY_IGNORADO_KEY)).toBe(legado.installLocation);
        unmount();

        render(<StartupNotices/>);
        await waitFor(() => expect(GetLegacyInstall).toHaveBeenCalledTimes(2));
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
});
