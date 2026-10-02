import {describe, expect, it, vi} from 'vitest';
import {render, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {UpdateContext} from '../contexts/UpdateContext';
import type {UseUpdateResult} from '../hooks/useUpdate';
import UpdateBanner from './UpdateBanner';

const estado = (extra: Partial<UseUpdateResult> = {}): UseUpdateResult => ({
    info: {
        available: true,
        currentVersion: '1.0.0',
        latestVersion: '1.1.0',
        releaseNotes: '<img src=x onerror=alert(1)> **novidade**',
        releaseUrl: 'https://github.com/eddie-naipes/naipe-logger/releases/tag/v1.1.0',
        publishedAt: '',
        canInstall: true,
    },
    available: true,
    checking: false,
    installing: false,
    progress: null,
    dismissed: false,
    check: vi.fn(() => Promise.resolve(null)),
    install: vi.fn(() => Promise.resolve()),
    openReleasePage: vi.fn(() => Promise.resolve()),
    dismiss: vi.fn(),
    ...extra,
});

const renderizar = (valor: UseUpdateResult) => render(
    <UpdateContext.Provider value={valor}>
        <UpdateBanner/>
    </UpdateContext.Provider>
);

describe('UpdateBanner', () => {
    it('mostra as notas como texto puro, sem interpretar HTML', async () => {
        const {container} = renderizar(estado());
        await userEvent.click(screen.getByRole('button', {name: 'Ver novidades'}));

        expect(screen.getByText('<img src=x onerror=alert(1)> **novidade**')).toBeInTheDocument();
        expect(container.querySelector('img')).toBeNull();
    });

    it('com canInstall=false oferece só a página da versão', async () => {
        const valor = estado({info: {...estado().info!, canInstall: false}});
        renderizar(valor);

        expect(screen.queryByRole('button', {name: /Atualizar agora/})).not.toBeInTheDocument();
        await userEvent.click(screen.getByRole('button', {name: /Abrir página da versão/}));
        expect(valor.openReleasePage).toHaveBeenCalledTimes(1);
    });

    it('durante o download desabilita o botão e mostra o progresso', () => {
        renderizar(estado({installing: true, progress: {received: 256, total: 1024}}));

        expect(screen.getByRole('button', {name: /Baixando/})).toBeDisabled();
        expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '25');
    });

    it('some quando dispensado', () => {
        const {container} = renderizar(estado({dismissed: true}));
        expect(container).toBeEmptyDOMElement();
    });
});
