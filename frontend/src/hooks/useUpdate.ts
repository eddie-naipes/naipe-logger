import {useCallback, useEffect, useMemo, useRef, useState} from 'react';
import {toast} from 'react-toastify';
import {
    CheckForUpdate,
    CheckForUpdateNow,
    DownloadAndInstallUpdate,
    OpenReleasePage
} from '@wailsjs/go/backend/App';
import {EventsOn} from '@wailsjs/runtime/runtime';
import type {UpdateInfo, UpdateProgress} from '../types/backend';
import {errMsg} from '../utils/errors';

// Nomes dos eventos emitidos por backend/app_update.go.
export const EVENT_UPDATE_AVAILABLE = 'update:available';
export const EVENT_UPDATE_PROGRESS = 'update:progress';

export interface UseUpdateOptions {
    // Verifica uma vez ao montar (ou assim que virar true): espelha a
    // preferência checkUpdatesOnStartup, que só é conhecida depois de carregar
    // as configurações.
    autoCheck: boolean;
}

export interface UseUpdateResult {
    info: UpdateInfo | null;
    available: boolean;
    checking: boolean;
    installing: boolean;
    progress: UpdateProgress | null;
    dismissed: boolean;
    // force ignora o cache de 1h do backend (botão "Verificar atualizações").
    check: (force?: boolean) => Promise<UpdateInfo | null>;
    install: () => Promise<void>;
    openReleasePage: () => Promise<void>;
    dismiss: () => void;
}

// Percentual do download (0–100), ou null quando o tamanho total é
// desconhecido.
export const progressPercent = (progress: UpdateProgress | null): number | null => {
    if (!progress || progress.total <= 0) return null;
    return Math.min(100, Math.round((progress.received / progress.total) * 100));
};

const useUpdate = ({autoCheck}: UseUpdateOptions): UseUpdateResult => {
    const [info, setInfo] = useState<UpdateInfo | null>(null);
    const [checking, setChecking] = useState(false);
    const [installing, setInstalling] = useState(false);
    const [progress, setProgress] = useState<UpdateProgress | null>(null);
    const [dismissed, setDismissed] = useState(false);

    // Refs: travam clique duplo no "Atualizar agora" e evitam uma segunda
    // checagem automática quando o evento do startup já trouxe o resultado.
    const installingRef = useRef(false);
    const autoCheckedRef = useRef(false);
    const mountedRef = useRef(true);

    useEffect(() => {
        mountedRef.current = true;
        const offAvailable = EventsOn(EVENT_UPDATE_AVAILABLE, (dados: UpdateInfo) => {
            autoCheckedRef.current = true;
            setInfo(dados);
        });
        const offProgress = EventsOn(EVENT_UPDATE_PROGRESS, (dados: UpdateProgress) => {
            setProgress(dados);
        });
        return () => {
            mountedRef.current = false;
            offAvailable();
            offProgress();
        };
    }, []);

    const check = useCallback(async (force = false): Promise<UpdateInfo | null> => {
        setChecking(true);
        try {
            const resultado = await (force ? CheckForUpdateNow() : CheckForUpdate());
            if (!mountedRef.current) return resultado;
            setInfo(resultado);
            // Uma checagem explícita volta a mostrar o aviso dispensado.
            if (force && resultado.available) setDismissed(false);
            return resultado;
        } catch (error) {
            console.error('Erro ao verificar atualizações:', error);
            if (force) toast.error('Não foi possível verificar atualizações: ' + errMsg(error));
            return null;
        } finally {
            if (mountedRef.current) setChecking(false);
        }
    }, []);

    useEffect(() => {
        if (!autoCheck || autoCheckedRef.current) return;
        autoCheckedRef.current = true;
        void check();
    }, [autoCheck, check]);

    const openReleasePage = useCallback(async (): Promise<void> => {
        try {
            await OpenReleasePage();
        } catch (error) {
            console.error('Erro ao abrir a página da versão:', error);
            toast.error('Não foi possível abrir a página da versão: ' + errMsg(error));
        }
    }, []);

    const install = useCallback(async (): Promise<void> => {
        if (installingRef.current) return;
        installingRef.current = true;
        setInstalling(true);
        setProgress(null);
        try {
            // Em caso de sucesso o backend executa o instalador e fecha o app.
            await DownloadAndInstallUpdate();
        } catch (error) {
            console.error('Erro ao instalar atualização:', error);
            toast.error('Não foi possível instalar a atualização: ' + errMsg(error));
            installingRef.current = false;
            if (mountedRef.current) {
                setInstalling(false);
                setProgress(null);
            }
        }
    }, []);

    const dismiss = useCallback(() => setDismissed(true), []);

    // Memoizado: o resultado vai para um contexto e não deve forçar render
    // dos consumidores a cada render do App.
    return useMemo(() => ({
        info,
        available: info?.available ?? false,
        checking,
        installing,
        progress,
        dismissed,
        check,
        install,
        openReleasePage,
        dismiss,
    }), [info, checking, installing, progress, dismissed, check, install, openReleasePage, dismiss]);
};

export default useUpdate;
