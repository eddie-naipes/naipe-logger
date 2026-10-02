import React, {useMemo} from 'react';
import {FiAlertCircle, FiCheck, FiClock, FiLoader} from 'react-icons/fi';
import ConflictBanner from './ConflictBanner';

// Plano de lançamento gerado, com resumo, conflitos e o botão de executar.
const PlanPreview = ({
                         workDays,
                         savedTasks,
                         conflicts,
                         isCheckingConflicts,
                         conflictCheckFailed,
                         isSubmitting,
                         onSubmit,
                         formatDate
                     }) => {
    const totals = useMemo(() => {
        const totalMinutes = workDays.reduce((sum, day) => sum + (day.totalMin || 0), 0);
        const entries = workDays.reduce((sum, day) => sum + (day.entries ? day.entries.length : 0), 0);
        return {
            days: workDays.length,
            hours: Math.floor(totalMinutes / 60),
            minutes: totalMinutes % 60,
            entries
        };
    }, [workDays]);

    // Calculado uma vez por plano (antes rodava duas vezes a cada render).
    const taskDaysSummary = useMemo(() => {
        const taskDays = {};
        workDays.forEach(day => {
            day.entries?.forEach(entry => {
                if (!taskDays[entry.taskId]) taskDays[entry.taskId] = new Set();
                taskDays[entry.taskId].add(day.date);
            });
        });

        return Object.entries(taskDays).map(([taskId, daysSet]) => {
            const task = savedTasks.find(t => t.taskId === Number(taskId));
            return `${task?.taskName || `Tarefa ${taskId}`}: ${daysSet.size} dias`;
        }).join(' • ');
    }, [workDays, savedTasks]);

    if (!workDays || workDays.length === 0) return null;

    return (
        <div className="card mb-6">
            <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center">
                    <FiClock className="w-5 h-5 mr-2" aria-hidden="true"/>
                    Plano de Lançamento
                </h2>

                <div className="text-sm text-gray-600 dark:text-gray-400">
                    <span className="font-medium">{totals.days} dias</span> •{' '}
                    <span className="font-medium">{totals.hours}h {totals.minutes}min</span> •{' '}
                    <span className="font-medium">{totals.entries} entradas</span>
                </div>
            </div>

            <ConflictBanner
                conflicts={conflicts}
                isChecking={isCheckingConflicts}
                checkFailed={conflictCheckFailed}
                formatDate={formatDate}
            />

            {taskDaysSummary && (
                <div className="mb-4 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                    <p className="text-sm text-blue-700 dark:text-blue-300">
                        <strong>Distribuição por tarefa:</strong> {taskDaysSummary}
                    </p>
                </div>
            )}

            <div className="overflow-y-auto max-h-96">
                <div className="space-y-4">
                    {workDays.map((day) => (
                        <div key={day.date} className="border border-gray-200 rounded-lg p-4 dark:border-gray-700">
                            <h3 className="text-md font-medium text-gray-900 dark:text-white mb-2">
                                {formatDate(day.date)}
                            </h3>

                            {day.entries && day.entries.length > 0 ? (
                                <div className="space-y-2">
                                    {day.entries.map((entry, entryIndex) => (
                                        <div
                                            key={`${day.date}-${entry.taskId}-${entry.entry.time}-${entryIndex}`}
                                            className="bg-gray-50 rounded-md p-3 text-sm dark:bg-gray-800"
                                        >
                                            <div className="flex justify-between">
                                                <div>
                                                    <span className="font-medium">{entry.entry.description}</span>
                                                    <span className="text-gray-600 dark:text-gray-400 ml-2">
                                                        ({entry.entry.minutes} min • {entry.entry.time ? entry.entry.time.substring(0, 5) : '00:00'})
                                                    </span>
                                                </div>
                                                <div className="text-gray-700 dark:text-gray-300">
                                                    TaskID: {entry.taskId}
                                                </div>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            ) : (
                                <p className="text-sm text-gray-500 dark:text-gray-400 py-2">
                                    Nenhuma entrada para este dia.
                                </p>
                            )}

                            <div className="mt-2 text-right text-sm">
                                <span className="text-gray-600 dark:text-gray-400">Total do dia:</span>
                                <span className="font-medium ml-1">{day.totalMin || 0} minutos</span>
                                <span className="text-gray-600 dark:text-gray-400 ml-1">
                                    ({((day.totalMin || 0) / 60).toFixed(1)}h)
                                </span>
                            </div>
                        </div>
                    ))}
                </div>
            </div>

            <div className="mt-4">
                {isSubmitting && (
                    <p className="mb-4 text-sm text-gray-500 dark:text-gray-400 text-center" role="status">
                        Processando lançamentos...
                    </p>
                )}

                <button
                    type="button"
                    onClick={onSubmit}
                    disabled={isSubmitting || isCheckingConflicts}
                    className={`flex items-center justify-center disabled:opacity-50 ${conflicts.length > 0 ? 'btn-warning' : 'btn-success'}`}
                >
                    {isSubmitting ? (
                        <>
                            <FiLoader className="w-5 h-5 mr-2 animate-spin" aria-hidden="true"/>
                            Enviando...
                        </>
                    ) : conflicts.length > 0 ? (
                        <>
                            <FiAlertCircle className="w-5 h-5 mr-2" aria-hidden="true"/>
                            Executar mesmo com {conflicts.length} dia(s) já lançado(s)
                        </>
                    ) : (
                        <>
                            <FiCheck className="w-5 h-5 mr-2" aria-hidden="true"/>
                            Executar Lançamento
                        </>
                    )}
                </button>
            </div>
        </div>
    );
};

export default PlanPreview;
