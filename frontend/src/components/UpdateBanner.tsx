import {useState} from 'react';
import {FiDownload, FiExternalLink, FiGift, FiLoader, FiX} from 'react-icons/fi';
import {useUpdateContext} from '../contexts/UpdateContext';
import {progressPercent} from '../hooks/useUpdate';

const formatMB = (bytes: number): string => (bytes / (1024 * 1024)).toFixed(1);

// Aviso discreto no topo do conteúdo quando há versão nova. As notas da
// release são exibidas como texto puro (o React escapa o conteúdo): nada de
// HTML/markdown vindo do GitHub é interpretado.
const UpdateBanner = () => {
    const {info, available, dismissed, installing, progress, install, openReleasePage, dismiss} = useUpdateContext();
    const [showNotes, setShowNotes] = useState(false);

    if (!info || !available || dismissed) return null;

    const percent = progressPercent(progress);
    const notes = info.releaseNotes.trim();

    return (
        <section
            aria-label="Atualização disponível"
            className="border-b border-primary-200 bg-primary-50 px-4 py-3 text-sm dark:border-primary-900 dark:bg-primary-900/20"
        >
            <div className="flex flex-wrap items-center gap-3">
                <FiGift className="h-5 w-5 shrink-0 text-primary-600 dark:text-primary-400" aria-hidden="true"/>
                <p className="flex-1 min-w-48 text-gray-800 dark:text-gray-200">
                    Nova versão <strong>{info.latestVersion}</strong> disponível
                    {info.currentVersion && <> (você usa a {info.currentVersion})</>}.
                    {notes && (
                        <button
                            type="button"
                            onClick={() => setShowNotes(v => !v)}
                            aria-expanded={showNotes}
                            className="ml-2 font-medium text-primary-700 underline hover:text-primary-800 dark:text-primary-400 dark:hover:text-primary-300"
                        >
                            {showNotes ? 'Ocultar novidades' : 'Ver novidades'}
                        </button>
                    )}
                </p>

                <div className="flex items-center gap-2">
                    {info.canInstall ? (
                        <button
                            type="button"
                            onClick={() => void install()}
                            disabled={installing}
                            className="btn-primary flex items-center px-3 py-1.5 disabled:opacity-60"
                        >
                            {installing ? (
                                <>
                                    <FiLoader className="mr-2 h-4 w-4 animate-spin" aria-hidden="true"/>
                                    Baixando...
                                </>
                            ) : (
                                <>
                                    <FiDownload className="mr-2 h-4 w-4" aria-hidden="true"/>
                                    Atualizar agora
                                </>
                            )}
                        </button>
                    ) : (
                        <button
                            type="button"
                            onClick={() => void openReleasePage()}
                            className="btn-primary flex items-center px-3 py-1.5"
                        >
                            <FiExternalLink className="mr-2 h-4 w-4" aria-hidden="true"/>
                            Abrir página da versão
                        </button>
                    )}
                    <button
                        type="button"
                        onClick={dismiss}
                        disabled={installing}
                        aria-label="Dispensar aviso de atualização nesta sessão"
                        title="Dispensar nesta sessão"
                        className="rounded-full p-1 text-gray-500 hover:bg-primary-100 hover:text-gray-700 disabled:opacity-50 dark:text-gray-400 dark:hover:bg-primary-900/40 dark:hover:text-gray-200"
                    >
                        <FiX className="h-5 w-5" aria-hidden="true"/>
                    </button>
                </div>
            </div>

            {installing && (
                <div className="mt-3">
                    <div
                        role="progressbar"
                        aria-label="Download da atualização"
                        aria-valuemin={0}
                        aria-valuemax={100}
                        aria-valuenow={percent ?? undefined}
                        className="h-2 w-full overflow-hidden rounded-full bg-primary-100 dark:bg-gray-700"
                    >
                        <div
                            className={percent === null
                                ? 'h-2 w-1/3 animate-pulse rounded-full bg-primary-600'
                                : 'h-2 rounded-full bg-primary-600 transition-all'}
                            style={percent === null ? undefined : {width: `${percent}%`}}
                        />
                    </div>
                    <p className="mt-1 text-xs text-gray-600 dark:text-gray-400">
                        {progress
                            ? percent === null
                                ? `${formatMB(progress.received)} MB baixados`
                                : `${percent}% — ${formatMB(progress.received)} de ${formatMB(progress.total)} MB`
                            : 'Iniciando download...'}
                        {' '}O aplicativo será fechado para o instalador rodar.
                    </p>
                </div>
            )}

            {showNotes && notes && (
                <pre className="mt-3 max-h-48 overflow-y-auto whitespace-pre-wrap wrap-break-word rounded-md border border-primary-100 bg-white p-3 font-sans text-xs text-gray-700 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300">
                    {notes}
                </pre>
            )}
        </section>
    );
};

export default UpdateBanner;
