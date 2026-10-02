import React from 'react';
import {FiAlertCircle, FiLoader} from 'react-icons/fi';

// Estado da verificação de lançamentos já existentes nos dias do plano.
const ConflictBanner = ({conflicts, isChecking, checkFailed, formatDate}) => (
    <>
        {isChecking && (
            <div className="mb-4 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg flex items-center" role="status">
                <FiLoader className="w-4 h-4 mr-2 animate-spin text-gray-500" aria-hidden="true"/>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                    Verificando lançamentos já existentes...
                </p>
            </div>
        )}

        {checkFailed && (
            <div className="mb-4 p-3 bg-gray-50 dark:bg-gray-800 border-l-4 border-gray-400 rounded-lg" role="alert">
                <p className="text-sm text-gray-700 dark:text-gray-300">
                    Não foi possível verificar lançamentos existentes. Confira manualmente
                    antes de enviar — horas duplicadas não podem ser desfeitas automaticamente.
                </p>
            </div>
        )}

        {conflicts.length > 0 && (
            <div className="mb-4 p-4 bg-amber-50 dark:bg-amber-900/20 border-l-4 border-amber-500 rounded-lg" role="alert">
                <div className="flex items-start">
                    <FiAlertCircle className="w-5 h-5 mr-3 mt-0.5 text-amber-500 flex-shrink-0" aria-hidden="true"/>
                    <div className="flex-1">
                        <h3 className="text-sm font-semibold text-amber-800 dark:text-amber-300">
                            {conflicts.length} dia(s) já possuem lançamentos
                        </h3>
                        <p className="mt-1 text-sm text-amber-700 dark:text-amber-400">
                            Enviar o plano vai <strong>somar</strong> as horas abaixo, não substituí-las.
                            Não há como desfazer automaticamente.
                        </p>
                        <ul className="mt-3 space-y-1.5">
                            {conflicts.map(conflict => (
                                <li key={conflict.date} className="text-sm text-amber-800 dark:text-amber-300">
                                    <span className="font-medium">{formatDate(conflict.date)}</span>
                                    {' — já tem '}
                                    <span className="font-medium">
                                        {(conflict.existingMinutes / 60).toFixed(1)}h
                                    </span>
                                    {` em ${conflict.existingEntries} entrada(s); plano adiciona `}
                                    <span className="font-medium">
                                        {(conflict.plannedMinutes / 60).toFixed(1)}h
                                    </span>
                                    {conflict.sameTask && conflict.sameTask.length > 0 && (
                                        <ul className="ml-4 mt-1 space-y-0.5">
                                            {conflict.sameTask.map(task => (
                                                <li key={task.taskId} className="text-xs text-amber-900 dark:text-amber-200">
                                                    ⚠ mesma tarefa
                                                    {task.taskName ? ` "${task.taskName}"` : ` #${task.taskId}`}
                                                    : {(task.existingMinutes / 60).toFixed(1)}h já lançadas
                                                </li>
                                            ))}
                                        </ul>
                                    )}
                                </li>
                            ))}
                        </ul>
                    </div>
                </div>
            </div>
        )}
    </>
);

export default ConflictBanner;
