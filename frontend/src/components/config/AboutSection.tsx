import {useEffect, useState} from 'react';
import {toast} from 'react-toastify';
import {FiCheckCircle, FiDownload, FiExternalLink, FiInfo, FiLoader, FiRefreshCw} from 'react-icons/fi';
import {GetAppSettings, GetAppVersion, SaveAppSettings} from '@wailsjs/go/backend/App';
import {useUpdateContext} from '../../contexts/UpdateContext';
import {progressPercent} from '../../hooks/useUpdate';
import {errMsg} from '../../utils/errors';

// Seção "Sobre / Atualizações" da página de configuração: versão atual,
// verificação manual (ignora o cache de 1h) e a preferência de verificar ao
// iniciar.
const AboutSection = () => {
    const updater = useUpdateContext();
    const [appVersion, setAppVersion] = useState('');
    const [checkOnStartup, setCheckOnStartup] = useState(true);
    const [isSavingPref, setIsSavingPref] = useState(false);
    const [upToDate, setUpToDate] = useState(false);

    useEffect(() => {
        const load = async () => {
            const [versao, settings] = await Promise.allSettled([GetAppVersion(), GetAppSettings()]);
            if (versao.status === 'fulfilled') setAppVersion(versao.value);
            if (settings.status === 'fulfilled') setCheckOnStartup(settings.value.checkUpdatesOnStartup);
        };
        void load();
    }, []);

    const handleCheck = async () => {
        setUpToDate(false);
        const info = await updater.check(true);
        if (!info) return;
        if (info.available) {
            toast.info(`Nova versão ${info.latestVersion} disponível.`);
        } else {
            setUpToDate(true);
            toast.success('Você está na versão mais recente.');
        }
    };

    const handleToggle = async (valor: boolean) => {
        setIsSavingPref(true);
        try {
            // Parte do objeto completo salvo para não zerar as outras preferências.
            const settings = await GetAppSettings();
            settings.checkUpdatesOnStartup = valor;
            await SaveAppSettings(settings);
            setCheckOnStartup(valor);
        } catch (error) {
            console.error('Erro ao salvar preferência de atualização:', error);
            toast.error('Erro ao salvar preferência: ' + errMsg(error));
        } finally {
            setIsSavingPref(false);
        }
    };

    const info = updater.info;
    const percent = progressPercent(updater.progress);

    return (
        <div className="card max-w-md mx-auto mt-6">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-1 flex items-center">
                <FiInfo className="w-5 h-5 mr-2" aria-hidden="true"/>
                Sobre / Atualizações
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                Versão atual: <strong>{appVersion || '—'}</strong>
            </p>

            <button
                type="button"
                onClick={() => void handleCheck()}
                disabled={updater.checking || updater.installing}
                className="w-full btn-secondary flex items-center justify-center disabled:opacity-50"
            >
                <FiRefreshCw className={`w-5 h-5 mr-2 ${updater.checking ? 'animate-spin' : ''}`} aria-hidden="true"/>
                {updater.checking ? 'Verificando...' : 'Verificar atualizações'}
            </button>

            <div aria-live="polite" className="mt-3 text-sm">
                {info?.available ? (
                    <div className="bg-blue-50 dark:bg-blue-900/20 p-3 rounded-lg border border-blue-200 dark:border-blue-800 text-blue-700 dark:text-blue-300">
                        <p>Nova versão <strong>{info.latestVersion}</strong> disponível.</p>
                        {info.canInstall ? (
                            <button
                                type="button"
                                onClick={() => void updater.install()}
                                disabled={updater.installing}
                                className="mt-2 btn-primary flex items-center px-3 py-1.5 disabled:opacity-60"
                            >
                                {updater.installing ? (
                                    <>
                                        <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/>
                                        {percent === null ? 'Baixando...' : `Baixando ${percent}%`}
                                    </>
                                ) : (
                                    <>
                                        <FiDownload className="w-4 h-4 mr-2" aria-hidden="true"/>
                                        Atualizar agora
                                    </>
                                )}
                            </button>
                        ) : (
                            <button
                                type="button"
                                onClick={() => void updater.openReleasePage()}
                                className="mt-2 btn-primary flex items-center px-3 py-1.5"
                            >
                                <FiExternalLink className="w-4 h-4 mr-2" aria-hidden="true"/>
                                Abrir página da versão
                            </button>
                        )}
                    </div>
                ) : upToDate && (
                    <p className="flex items-center text-green-700 dark:text-green-400">
                        <FiCheckCircle className="w-4 h-4 mr-2" aria-hidden="true"/>
                        Você está na versão mais recente.
                    </p>
                )}
            </div>

            <label className="mt-4 flex items-center justify-between gap-3 cursor-pointer">
                <span className="text-sm text-gray-700 dark:text-gray-300">Verificar atualizações ao iniciar</span>
                <input
                    type="checkbox"
                    role="switch"
                    checked={checkOnStartup}
                    disabled={isSavingPref}
                    onChange={(e) => void handleToggle(e.target.checked)}
                    className="w-5 h-5 text-primary-600 bg-gray-100 border-gray-300 rounded focus:ring-primary-500 dark:focus:ring-primary-600 dark:bg-gray-700 dark:border-gray-600 disabled:opacity-50"
                />
            </label>
        </div>
    );
};

export default AboutSection;
