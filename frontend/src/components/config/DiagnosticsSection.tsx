import {useEffect, useState} from 'react';
import {toast} from 'react-toastify';
import {FiFolder, FiTool} from 'react-icons/fi';
import {GetLogsPath, OpenLogsFolder} from '@wailsjs/go/backend/App';
import {errMsg} from '../../utils/errors';

// Seção "Diagnóstico": mostra onde fica o app.log e abre a pasta para anexar a
// um pedido de suporte. GetLogsPath devolve "" quando o log não abriu.
const DiagnosticsSection = () => {
    const [logsPath, setLogsPath] = useState<string | null>(null);

    useEffect(() => {
        GetLogsPath()
            .then(setLogsPath)
            .catch((error: unknown) => {
                console.error('Erro ao obter pasta de logs:', error);
                setLogsPath('');
            });
    }, []);

    const handleOpen = async () => {
        try {
            await OpenLogsFolder();
        } catch (error) {
            console.error('Erro ao abrir pasta de logs:', error);
            toast.error('Não foi possível abrir a pasta de logs: ' + errMsg(error));
        }
    };

    return (
        <div className="card max-w-md mx-auto mt-6">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-1 flex items-center">
                <FiTool className="w-5 h-5 mr-2" aria-hidden="true"/>
                Diagnóstico
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                Os logs ajudam a investigar problemas. O token de API nunca é gravado neles.
            </p>
            {logsPath ? (
                <p className="text-xs font-mono break-all bg-gray-50 dark:bg-gray-700 text-gray-700 dark:text-gray-200 p-2 rounded mb-4">
                    {logsPath}
                </p>
            ) : logsPath === '' && (
                <p className="text-sm text-amber-700 dark:text-amber-300 mb-4">
                    O arquivo de log não está disponível nesta execução.
                </p>
            )}
            <button
                type="button"
                onClick={() => void handleOpen()}
                disabled={!logsPath}
                className="w-full btn-secondary flex items-center justify-center disabled:opacity-50"
            >
                <FiFolder className="w-5 h-5 mr-2" aria-hidden="true"/>
                Abrir pasta de logs
            </button>
        </div>
    );
};

export default DiagnosticsSection;
