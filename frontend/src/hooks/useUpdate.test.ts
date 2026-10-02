import {beforeEach, describe, expect, it, vi} from 'vitest';
import {act, renderHook, waitFor} from '@testing-library/react';
import {toast} from 'react-toastify';
import {
    CheckForUpdate,
    CheckForUpdateNow,
    DownloadAndInstallUpdate,
    OpenReleasePage
} from '@wailsjs/go/backend/App';
import {EventsOn} from '@wailsjs/runtime/runtime';
import type {UpdateInfo} from '../types/backend';
import useUpdate, {EVENT_UPDATE_AVAILABLE, EVENT_UPDATE_PROGRESS, progressPercent} from './useUpdate';

vi.mock('@wailsjs/go/backend/App', () => ({
    CheckForUpdate: vi.fn(),
    CheckForUpdateNow: vi.fn(),
    DownloadAndInstallUpdate: vi.fn(),
    OpenReleasePage: vi.fn(),
}));

vi.mock('@wailsjs/runtime/runtime', () => ({
    EventsOn: vi.fn(),
}));

vi.mock('react-toastify', () => ({
    toast: {success: vi.fn(), warning: vi.fn(), error: vi.fn(), info: vi.fn()},
}));

const versao = (extra: Partial<UpdateInfo> = {}): UpdateInfo => ({
    available: true,
    currentVersion: '1.0.0',
    latestVersion: '1.1.0',
    releaseNotes: 'Correções',
    releaseUrl: 'https://github.com/eddie-naipes/naipe-logger/releases/tag/v1.1.0',
    publishedAt: '2026-09-30T12:00:00Z',
    canInstall: true,
    ...extra,
});

// Ouvintes registrados via EventsOn, e os cancelamentos devolvidos a cada um.
let ouvintes: Map<string, (dados: unknown) => void>;
let cancelamentos: Map<string, ReturnType<typeof vi.fn>>;

const emitir = (evento: string, dados: unknown) => {
    const ouvinte = ouvintes.get(evento);
    if (!ouvinte) throw new Error(`ninguém ouvindo ${evento}`);
    act(() => ouvinte(dados));
};

describe('useUpdate', () => {
    beforeEach(() => {
        ouvintes = new Map();
        cancelamentos = new Map();
        vi.mocked(EventsOn).mockImplementation((evento, callback) => {
            ouvintes.set(evento, callback as (dados: unknown) => void);
            const off = vi.fn(() => ouvintes.delete(evento));
            cancelamentos.set(evento, off);
            return off;
        });
        vi.spyOn(console, 'error').mockImplementation(() => {});
    });

    it('verifica ao montar e expõe a versão disponível', async () => {
        vi.mocked(CheckForUpdate).mockResolvedValue(versao());
        const {result} = renderHook(() => useUpdate({autoCheck: true}));

        await waitFor(() => expect(result.current.available).toBe(true));
        expect(CheckForUpdate).toHaveBeenCalledTimes(1);
        expect(result.current.info?.latestVersion).toBe('1.1.0');
        expect(result.current.checking).toBe(false);
    });

    it('sem atualização não marca available', async () => {
        vi.mocked(CheckForUpdate).mockResolvedValue(versao({available: false, latestVersion: '1.0.0'}));
        const {result} = renderHook(() => useUpdate({autoCheck: true}));

        await waitFor(() => expect(result.current.info).not.toBeNull());
        expect(result.current.available).toBe(false);
    });

    it('não verifica sozinho com a preferência desligada', () => {
        renderHook(() => useUpdate({autoCheck: false}));
        expect(CheckForUpdate).not.toHaveBeenCalled();
    });

    it('evento update:available dispensa a checagem automática', () => {
        const {result, rerender} = renderHook(({auto}) => useUpdate({autoCheck: auto}), {
            initialProps: {auto: false},
        });
        emitir(EVENT_UPDATE_AVAILABLE, versao());
        rerender({auto: true});

        expect(result.current.available).toBe(true);
        expect(CheckForUpdate).not.toHaveBeenCalled();
    });

    it('checagem forçada usa CheckForUpdateNow e reexibe o aviso dispensado', async () => {
        vi.mocked(CheckForUpdateNow).mockResolvedValue(versao());
        const {result} = renderHook(() => useUpdate({autoCheck: false}));
        act(() => result.current.dismiss());
        expect(result.current.dismissed).toBe(true);

        await act(() => result.current.check(true));

        expect(CheckForUpdateNow).toHaveBeenCalledTimes(1);
        expect(CheckForUpdate).not.toHaveBeenCalled();
        expect(result.current.dismissed).toBe(false);
    });

    it('acompanha o progresso do download e ignora clique duplo', async () => {
        let concluir!: () => void;
        vi.mocked(DownloadAndInstallUpdate).mockReturnValue(new Promise<void>((resolve) => {
            concluir = resolve;
        }));
        const {result} = renderHook(() => useUpdate({autoCheck: false}));

        let instalacao!: Promise<void>;
        act(() => {
            instalacao = result.current.install();
            void result.current.install();
        });
        expect(DownloadAndInstallUpdate).toHaveBeenCalledTimes(1);
        expect(result.current.installing).toBe(true);

        emitir(EVENT_UPDATE_PROGRESS, {received: 512, total: 1024});
        expect(result.current.progress).toEqual({received: 512, total: 1024});
        expect(progressPercent(result.current.progress)).toBe(50);

        await act(async () => {
            concluir();
            await instalacao;
        });
        // Sucesso: o backend fecha o app; o botão continua travado até lá.
        expect(result.current.installing).toBe(true);
    });

    it('falha no download libera o botão e avisa', async () => {
        vi.mocked(DownloadAndInstallUpdate).mockRejectedValue('checksum não confere');
        const {result} = renderHook(() => useUpdate({autoCheck: false}));

        await act(() => result.current.install());

        expect(result.current.installing).toBe(false);
        expect(toast.error).toHaveBeenCalledWith('Não foi possível instalar a atualização: checksum não confere');
    });

    it('com canInstall=false oferece a página da release', async () => {
        vi.mocked(CheckForUpdate).mockResolvedValue(versao({canInstall: false}));
        const {result} = renderHook(() => useUpdate({autoCheck: true}));
        await waitFor(() => expect(result.current.available).toBe(true));
        expect(result.current.info?.canInstall).toBe(false);

        await act(() => result.current.openReleasePage());

        expect(OpenReleasePage).toHaveBeenCalledTimes(1);
        expect(DownloadAndInstallUpdate).not.toHaveBeenCalled();
    });

    it('cancela a assinatura dos eventos ao desmontar', () => {
        const {unmount} = renderHook(() => useUpdate({autoCheck: false}));
        expect(ouvintes.has(EVENT_UPDATE_AVAILABLE)).toBe(true);
        expect(ouvintes.has(EVENT_UPDATE_PROGRESS)).toBe(true);

        unmount();

        expect(cancelamentos.get(EVENT_UPDATE_AVAILABLE)).toHaveBeenCalledTimes(1);
        expect(cancelamentos.get(EVENT_UPDATE_PROGRESS)).toHaveBeenCalledTimes(1);
        expect(ouvintes.size).toBe(0);
    });
});

describe('progressPercent', () => {
    it('devolve null quando o tamanho é desconhecido', () => {
        expect(progressPercent(null)).toBeNull();
        expect(progressPercent({received: 10, total: 0})).toBeNull();
        expect(progressPercent({received: 10, total: -1})).toBeNull();
    });

    it('limita a 100%', () => {
        expect(progressPercent({received: 2048, total: 1024})).toBe(100);
    });
});
