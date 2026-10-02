import {FiAlertCircle, FiCheck, FiLoader, FiRefreshCw} from 'react-icons/fi';
import type {DeleteTimeEntryResult} from '../../types/backend';

interface DeleteResultsProps {
    results: readonly DeleteTimeEntryResult[];
    deleting: boolean;
    onRetryFailed: () => void;
}

// Resultado da última exclusão em lote, com o botão de reenviar só as que
// falharam.
const DeleteResults = ({results, deleting, onRetryFailed}: DeleteResultsProps) => {
    if (!results || results.length === 0) return null;

    const failedCount = results.filter(r => !r.success).length;

    return (
        <div className="mt-6 bg-gray-50 dark:bg-gray-800 rounded-lg p-4" aria-live="polite">
            <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">
                Resultados da Deleção
            </h3>
            <ul className="space-y-2 max-h-32 overflow-y-auto">
                {results.map((result) => (
                    <li key={`${result.entryId}-${result.success}`} className={`flex items-center text-sm ${
                        result.success
                            ? 'text-green-700 dark:text-green-300'
                            : 'text-red-700 dark:text-red-300'
                    }`}>
                        {result.success ? (
                            <FiCheck className="w-4 h-4 mr-2" aria-label="Sucesso"/>
                        ) : (
                            <FiAlertCircle className="w-4 h-4 mr-2" aria-label="Falha"/>
                        )}
                        <span>Entrada {result.entryId}: {result.message}</span>
                    </li>
                ))}
            </ul>

            {failedCount > 0 && (
                <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-600">
                    <button
                        type="button"
                        onClick={onRetryFailed}
                        disabled={deleting}
                        className="flex items-center px-3 py-2 text-sm bg-amber-50 hover:bg-amber-100 dark:bg-amber-900/20 dark:hover:bg-amber-900/40 text-amber-700 dark:text-amber-300 border border-amber-200 dark:border-amber-800 rounded-lg disabled:opacity-50"
                    >
                        {deleting ? (
                            <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/>
                        ) : (
                            <FiRefreshCw className="w-4 h-4 mr-2" aria-hidden="true"/>
                        )}
                        Reenviar {failedCount} que falharam
                    </button>
                </div>
            )}
        </div>
    );
};

export default DeleteResults;
