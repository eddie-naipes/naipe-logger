import {useCallback, useRef, useState} from 'react';
import {toast} from 'react-toastify';
import {FiAlertCircle, FiInfo, FiRefreshCw} from 'react-icons/fi';
import {ptBR} from 'date-fns/locale';
import MonthlyTimeCalendar, {type MonthlyTimeCalendarHandle} from '../components/MonthlyTimeCalendar';
import TaskSelector from '../components/timeLog/TaskSelector';
import PeriodForm from '../components/timeLog/PeriodForm';
import PlanPreview from '../components/timeLog/PlanPreview';
import ResultsPanel from '../components/timeLog/ResultsPanel';
import useSavedTasks from '../hooks/useSavedTasks';
import useNonWorkingDays from '../hooks/useNonWorkingDays';
import usePlan from '../hooks/usePlan';
import useBatchSubmit from '../hooks/useBatchSubmit';
import {IsWorkDay} from '@wailsjs/go/backend/App';
import {formatDateBR, todayYMD} from '../utils/dates';
import {describeNonWorkingDay} from '../utils/nonWorkingDays';
import type {DateRange} from '../hooks/useTimeEntries';

const formatDate = (dateString: string) =>
    formatDateBR(dateString, 'dd/MM/yyyy (EEEE)', dateString || 'Data inválida', {locale: ptBR});

const TimeLog = () => {
    const [dateRange, setDateRange] = useState<DateRange>(() => ({
        startDate: todayYMD(),
        endDate: todayYMD()
    }));
    const [error, setError] = useState<string | null>(null);
    const calendarRef = useRef<MonthlyTimeCalendarHandle>(null);

    const {
        savedTasks,
        selectedTasks,
        toggleTaskSelection,
        selectAllTasks,
        isLoading,
        error: savedTasksError,
        appliedTemplate
    } = useSavedTasks();

    const nonWorkingDays = useNonWorkingDays(dateRange.startDate, dateRange.endDate);

    const plan = usePlan(savedTasks, setError);

    // Recarrega o mês exibido no calendário (sem remontar nem voltar ao mês atual).
    const reloadCalendar = useCallback(() => {
        calendarRef.current?.refresh();
    }, []);

    const refreshCalendar = useCallback(() => {
        reloadCalendar();
        toast.success('Calendário atualizado com os novos lançamentos!');
    }, [reloadCalendar]);

    const batch = useBatchSubmit({
        workDays: plan.workDays,
        conflicts: plan.conflicts,
        isCheckingConflicts: plan.isCheckingConflicts,
        conflictCheckFailed: plan.conflictCheckFailed,
        checkConflicts: plan.checkConflicts,
        refreshCalendar,
        reloadCalendar,
        onError: setError
    });

    const handleDaySelection = async (formattedDate: string) => {
        const nonWorkingDay = nonWorkingDays[formattedDate];
        if (nonWorkingDay) {
            if (nonWorkingDay.type === 'holiday') {
                toast.warning(`${formattedDate} é um feriado: ${nonWorkingDay.name}. Não é possível lançar horas em feriados.`);
            } else if (nonWorkingDay.type === 'weekend') {
                toast.warning(`${formattedDate} é um fim de semana. Não é possível lançar horas em fins de semana.`);
            } else {
                toast.warning(`${formattedDate} não é dia útil (${describeNonWorkingDay(nonWorkingDay)}). Não é possível lançar horas.`);
            }
            return;
        }

        try {
            const isWorkDay = await IsWorkDay(formattedDate);
            if (!isWorkDay) {
                toast.warning(`${formattedDate} não é um dia útil. Não é possível lançar horas.`);
                return;
            }
        } catch (err) {
            console.error('Erro ao verificar dia útil:', err);
        }

        setDateRange({startDate: formattedDate, endDate: formattedDate});

        // Período e tarefas vão por parâmetro: o estado acima só estará
        // atualizado no próximo render.
        await plan.generatePlan({start: formattedDate, end: formattedDate, taskIds: selectedTasks});
    };

    const handleGenerate = () => {
        void plan.generatePlan({start: dateRange.startDate, end: dateRange.endDate, taskIds: selectedTasks});
    };

    if (isLoading) {
        return (
            <div className="flex justify-center items-center h-full" role="status" aria-label="Carregando tarefas">
                <div className="animate-spin-slow w-12 h-12 border-4 border-primary-600 border-t-transparent rounded-full"></div>
            </div>
        );
    }

    const shownError = error || savedTasksError;

    return (
        <div>
            <div className="mb-6 flex justify-between items-center">
                <div>
                    <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">Lançar Horas</h1>
                    <p className="text-gray-600 dark:text-gray-400">Registre horas trabalhadas em múltiplos dias</p>
                </div>
                <button
                    type="button"
                    onClick={refreshCalendar}
                    className="flex items-center px-3 py-2 text-sm bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 rounded-lg transition-colors"
                >
                    <FiRefreshCw className="w-4 h-4 mr-2" aria-hidden="true"/>
                    Atualizar Calendário
                </button>
            </div>

            <MonthlyTimeCalendar ref={calendarRef} onDayClick={(day) => void handleDaySelection(day)}/>

            {savedTasks.length > 0 && appliedTemplate && (
                <div className="mb-6 bg-blue-50 border-l-4 border-blue-500 p-4 dark:bg-blue-900/20 dark:border-blue-700">
                    <div className="flex items-start">
                        <FiInfo className="mt-0.5 w-5 h-5 text-blue-500 dark:text-blue-600 mr-2" aria-hidden="true"/>
                        <div>
                            <h3 className="text-sm font-medium text-blue-800 dark:text-blue-400">
                                Template &quot;{appliedTemplate}&quot; Aplicado
                            </h3>
                            <p className="mt-1 text-sm text-blue-700 dark:text-blue-200">
                                {savedTasks.length} tarefas foram carregadas do template. Configure o período e clique
                                em &quot;Gerar Plano&quot; para continuar.
                            </p>
                        </div>
                    </div>
                </div>
            )}

            {shownError && (
                <div role="alert" className="mb-6 bg-red-50 border-l-4 border-red-500 p-4 dark:bg-red-900/20 dark:border-red-700">
                    <div className="flex items-start">
                        <FiAlertCircle className="mt-0.5 w-5 h-5 text-red-500 dark:text-red-600 mr-2" aria-hidden="true"/>
                        <div>
                            <h3 className="text-sm font-medium text-red-800 dark:text-red-400">Erro</h3>
                            <p className="mt-1 text-sm text-red-700 dark:text-red-200">{shownError}</p>
                        </div>
                    </div>
                </div>
            )}

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
                <TaskSelector
                    savedTasks={savedTasks}
                    selectedTasks={selectedTasks}
                    onToggle={toggleTaskSelection}
                    onToggleAll={selectAllTasks}
                />

                <PeriodForm
                    dateRange={dateRange}
                    onChange={setDateRange}
                    nonWorkingDays={nonWorkingDays}
                    onGenerate={handleGenerate}
                    isGenerating={plan.isGenerating}
                    disabled={savedTasks.length === 0}
                />
            </div>

            <PlanPreview
                workDays={plan.workDays}
                savedTasks={savedTasks}
                conflicts={plan.conflicts}
                isCheckingConflicts={plan.isCheckingConflicts}
                conflictCheckFailed={plan.conflictCheckFailed}
                isSubmitting={batch.isSubmitting}
                onSubmit={() => void batch.submitPlan()}
                formatDate={formatDate}
            />

            {batch.showResults && (
                <ResultsPanel
                    results={batch.results}
                    failedEntries={batch.failedEntries}
                    undoableEntries={batch.undoableEntries}
                    notUndoableCount={batch.notUndoableCount}
                    isRetrying={batch.isRetrying}
                    isUndoing={batch.isUndoing}
                    onRetry={() => void batch.retryFailed()}
                    onUndo={() => void batch.undoBatch()}
                    onRefreshCalendar={refreshCalendar}
                />
            )}
        </div>
    );
};

export default TimeLog;
