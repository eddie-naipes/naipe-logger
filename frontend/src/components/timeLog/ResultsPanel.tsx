import {FiAlertCircle, FiCheck, FiCheckCircle, FiLoader, FiRefreshCw, FiTrash2} from 'react-icons/fi';
import type {KeyedTimeLogResult} from '../../hooks/useBatchSubmit';
import type {TimeLogResult} from '../../types/backend';

interface ResultsPanelProps {
    results: readonly KeyedTimeLogResult[];
    failedEntries: readonly TimeLogResult[];
    undoableEntries: readonly TimeLogResult[];
    notUndoableCount: number;
    isRetrying: boolean;
    isUndoing: boolean;
    onRetry: () => void;
    onUndo: () => void;
    onRefreshCalendar: () => void;
}

// Resultado do lote enviado: lista, resumo, reenviar falhas e desfazer.
const ResultsPanel = ({
                          results,
                          failedEntries,
                          undoableEntries,
                          notUndoableCount,
                          isRetrying,
                          isUndoing,
                          onRetry,
                          onUndo,
                          onRefreshCalendar
                      }: ResultsPanelProps) => {
    if (!results || results.length === 0) return null;

    const successCount = results.length - failedEntries.length;

    return (
        <div className="card" aria-live="polite">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                <FiCheckCircle className="w-5 h-5 mr-2" aria-hidden="true"/>
                Resultados do Lançamento
            </h2>

            <div className="overflow-y-auto max-h-64">
                <ul className="space-y-2">
                    {results.map((result) => (
                        <li
                            key={result._key}
                            className={`p-3 rounded-md ${
                                result.success
                                    ? 'bg-green-50 dark:bg-green-900/20'
                                    : 'bg-red-50 dark:bg-red-900/20'
                            }`}
                        >
                            <div className="flex items-start">
                                <div
                                    className={`flex-shrink-0 w-5 h-5 rounded-full flex items-center justify-center ${
                                        result.success
                                            ? 'bg-green-100 text-green-600 dark:bg-green-800 dark:text-green-200'
                                            : 'bg-red-100 text-red-600 dark:bg-red-800 dark:text-red-200'
                                    }`}
                                    aria-hidden="true"
                                >
                                    {result.success ? <FiCheck size={12}/> : <FiAlertCircle size={12}/>}
                                </div>
                                <div className="ml-3">
                                    <p className={`text-sm ${
                                        result.success
                                            ? 'text-green-800 dark:text-green-200'
                                            : 'text-red-800 dark:text-red-200'
                                    }`}>
                                        <span className="font-medium">
                                            {result.success ? 'Sucesso' : 'Falha'}:
                                        </span> {result.message || 'Sem mensagem'}
                                    </p>
                                    <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                                        Tarefa: {result.taskId} • Data: {result.date || 'N/A'}
                                    </p>
                                </div>
                            </div>
                        </li>
                    ))}
                </ul>
            </div>

            <div className="mt-4 bg-gray-50 p-3 rounded-md dark:bg-gray-800">
                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">Resumo</h3>
                <div className="grid grid-cols-2 gap-2">
                    <div className="bg-green-50 p-2 rounded dark:bg-green-900/20">
                        <p className="text-xs text-green-800 dark:text-green-200">
                            <span className="font-medium">Sucessos:</span> {successCount}
                        </p>
                    </div>
                    <div className="bg-red-50 p-2 rounded dark:bg-red-900/20">
                        <p className="text-xs text-red-800 dark:text-red-200">
                            <span className="font-medium">Falhas:</span> {failedEntries.length}
                        </p>
                    </div>
                </div>

                {failedEntries.length > 0 && (
                    <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-600">
                        <button
                            type="button"
                            onClick={onRetry}
                            disabled={isRetrying || isUndoing}
                            className="w-full flex items-center justify-center px-3 py-2 text-sm bg-amber-50 hover:bg-amber-100 dark:bg-amber-900/20 dark:hover:bg-amber-900/40 text-amber-700 dark:text-amber-300 border border-amber-200 dark:border-amber-800 rounded-lg transition-colors disabled:opacity-50"
                        >
                            {isRetrying ? (
                                <>
                                    <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/>
                                    Reenviando...
                                </>
                            ) : (
                                <>
                                    <FiRefreshCw className="w-4 h-4 mr-2" aria-hidden="true"/>
                                    Reenviar {failedEntries.length} que falharam
                                </>
                            )}
                        </button>
                    </div>
                )}

                {undoableEntries.length > 0 && (
                    <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-600">
                        <button
                            type="button"
                            onClick={onUndo}
                            disabled={isUndoing || isRetrying}
                            className="w-full flex items-center justify-center px-3 py-2 text-sm bg-red-50 hover:bg-red-100 dark:bg-red-900/20 dark:hover:bg-red-900/40 text-red-700 dark:text-red-300 border border-red-200 dark:border-red-800 rounded-lg transition-colors disabled:opacity-50"
                        >
                            {isUndoing ? (
                                <>
                                    <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/>
                                    Desfazendo...
                                </>
                            ) : (
                                <>
                                    <FiTrash2 className="w-4 h-4 mr-2" aria-hidden="true"/>
                                    Desfazer {undoableEntries.length} lançamento(s)
                                </>
                            )}
                        </button>

                        {notUndoableCount > 0 && (
                            <p className="mt-2 text-xs text-amber-700 dark:text-amber-400">
                                {notUndoableCount} lançamento(s) não podem ser desfeitos automaticamente
                                porque o Teamwork não devolveu o identificador da entrada. Remova-os
                                pelo Gerenciador de Apontamentos.
                            </p>
                        )}
                    </div>
                )}

                <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-600">
                    <button
                        type="button"
                        onClick={onRefreshCalendar}
                        className="w-full flex items-center justify-center px-3 py-2 text-sm bg-primary-100 hover:bg-primary-200 dark:bg-primary-900/20 dark:hover:bg-primary-900/40 text-primary-700 dark:text-primary-300 rounded-lg transition-colors"
                    >
                        <FiRefreshCw className="w-4 h-4 mr-2" aria-hidden="true"/>
                        Atualizar Calendário
                    </button>
                </div>
            </div>
        </div>
    );
};

export default ResultsPanel;
