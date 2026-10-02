import {useEffect, useState} from 'react';
import {toast} from 'react-toastify';
import {FiAlertTriangle, FiLoader, FiTrash2} from 'react-icons/fi';
import {CorruptedConfigBackups, GetLegacyInstall, RunLegacyUninstaller} from '@wailsjs/go/backend/App';
import Modal from './Modal';
import type {LegacyInstall} from '../types/backend';
import {errMsg} from '../utils/errors';

// Guarda o local da instalação antiga que o usuário pediu para não lembrar
// mais. Se aparecer outra instalação (outro local), o aviso volta.
export const LEGACY_IGNORADO_KEY = 'teamwork-logger:legacy-ignorado';

const lerIgnorado = (): string | null => {
    try {
        return window.localStorage.getItem(LEGACY_IGNORADO_KEY);
    } catch {
        return null;
    }
};

const gravarIgnorado = (local: string): void => {
    try {
        window.localStorage.setItem(LEGACY_IGNORADO_KEY, local);
    } catch {
        // Sem localStorage o aviso só volta na próxima abertura; não é crítico.
    }
};

// Avisos mostrados uma vez na inicialização, um de cada vez:
// 1. config.json/templates.json corrompidos foram recuperados com os padrões;
// 2. há uma instalação antiga (admin, Program Files) no Windows.
const StartupNotices = () => {
    const [backups, setBackups] = useState<string[]>([]);
    const [legacy, setLegacy] = useState<LegacyInstall | null>(null);
    const [corruptedClosed, setCorruptedClosed] = useState(false);
    const [legacyClosed, setLegacyClosed] = useState(false);
    const [dontAskAgain, setDontAskAgain] = useState(false);
    const [uninstalling, setUninstalling] = useState(false);

    useEffect(() => {
        CorruptedConfigBackups()
            .then(lista => setBackups(lista ?? []))
            .catch((error: unknown) => console.error('Erro ao verificar configuração corrompida:', error));

        GetLegacyInstall()
            .then(install => {
                if (install.found && lerIgnorado() !== install.installLocation) {
                    setLegacy(install);
                }
            })
            .catch((error: unknown) => console.error('Erro ao procurar instalação antiga:', error));
    }, []);

    const showCorrupted = backups.length > 0 && !corruptedClosed;
    const showLegacy = !showCorrupted && legacy !== null && !legacyClosed;

    const closeLegacy = () => {
        if (dontAskAgain && legacy) gravarIgnorado(legacy.installLocation);
        setLegacyClosed(true);
    };

    const handleUninstall = async () => {
        setUninstalling(true);
        try {
            await RunLegacyUninstaller();
            toast.info('Desinstalador da versão antiga iniciado. Siga as instruções na janela dele.');
            setLegacyClosed(true);
        } catch (error) {
            console.error('Erro ao executar desinstalador antigo:', error);
            toast.error('Não foi possível remover a versão antiga: ' + errMsg(error));
        } finally {
            setUninstalling(false);
        }
    };

    if (showCorrupted) {
        return (
            <Modal
                isOpen
                onClose={() => setCorruptedClosed(true)}
                title="Configuração recuperada"
                icon={<FiAlertTriangle className="w-5 h-5 mr-2 text-amber-500" aria-hidden="true"/>}
                footer={
                    <button type="button" className="btn-primary" onClick={() => setCorruptedClosed(true)}>
                        Entendi
                    </button>
                }
            >
                <p className="text-sm text-gray-700 dark:text-gray-300">
                    Um arquivo de configuração estava corrompido e não pôde ser lido. O aplicativo está usando a{' '}
                    <strong>configuração padrão</strong> — pode ser preciso reconectar ao Teamwork e cadastrar de
                    novo tarefas ou templates.
                </p>
                <p className="mt-3 text-sm text-gray-700 dark:text-gray-300">
                    O conteúdo original foi preservado para inspeção em:
                </p>
                <ul className="mt-2 space-y-1">
                    {backups.map(arquivo => (
                        <li
                            key={arquivo}
                            className="text-xs font-mono break-all bg-gray-50 dark:bg-gray-700 text-gray-700 dark:text-gray-200 p-2 rounded"
                        >
                            {arquivo}
                        </li>
                    ))}
                </ul>
            </Modal>
        );
    }

    if (showLegacy) {
        return (
            <Modal
                isOpen
                onClose={closeLegacy}
                closeDisabled={uninstalling}
                title="Versão antiga instalada"
                icon={<FiAlertTriangle className="w-5 h-5 mr-2 text-amber-500" aria-hidden="true"/>}
                footer={
                    <>
                        <button type="button" className="btn-secondary" onClick={closeLegacy} disabled={uninstalling}>
                            Lembrar depois
                        </button>
                        <button
                            type="button"
                            className="btn-primary flex items-center disabled:opacity-60"
                            onClick={() => void handleUninstall()}
                            disabled={uninstalling}
                        >
                            {uninstalling
                                ? <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/>
                                : <FiTrash2 className="w-4 h-4 mr-2" aria-hidden="true"/>}
                            Remover versão antiga
                        </button>
                    </>
                }
            >
                <p className="text-sm text-gray-700 dark:text-gray-300">
                    Há uma versão antiga do aplicativo{legacy.displayName && <> (<strong>{legacy.displayName}</strong>)</>}{' '}
                    instalada em:
                </p>
                <p className="mt-2 text-xs font-mono break-all bg-gray-50 dark:bg-gray-700 text-gray-700 dark:text-gray-200 p-2 rounded">
                    {legacy.installLocation || 'local não informado'}
                </p>
                <p className="mt-3 text-sm text-gray-700 dark:text-gray-300">
                    Ela não é mais atualizada e pode confundir os atalhos. Remover executa o desinstalador dela;
                    o <strong>Windows pedirá permissão de administrador</strong>. Suas configurações não são afetadas.
                </p>
                <label className="mt-4 flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300 cursor-pointer">
                    <input
                        type="checkbox"
                        checked={dontAskAgain}
                        onChange={(e) => setDontAskAgain(e.target.checked)}
                        className="w-4 h-4 text-primary-600 bg-gray-100 border-gray-300 rounded focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600"
                    />
                    Não perguntar novamente
                </label>
            </Modal>
        );
    }

    return null;
};

export default StartupNotices;
