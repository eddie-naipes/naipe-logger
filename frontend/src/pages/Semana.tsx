import {useCallback, useEffect, useMemo, useRef, useState} from 'react';
import {toast} from 'react-toastify';
import {FiAlertCircle, FiChevronLeft, FiChevronRight, FiCopy, FiGrid, FiLoader, FiRefreshCw} from 'react-icons/fi';
import {addDays} from 'date-fns';
import {ptBR} from 'date-fns/locale';
import {
    BuildCopyPreviousWeekPlan,
    DeleteMultipleTimeEntries,
    GetSavedTasks,
    GetWeekGrid,
    LogMultipleTimes
} from '@wailsjs/go/backend/App';
import WeekGridTable from '../components/week/WeekGridTable';
import AddTimeModal, {type AddTimeTarget} from '../components/week/AddTimeModal';
import CellEntriesModal from '../components/week/CellEntriesModal';
import EditEntryModal from '../components/timeEntries/EditEntryModal';
import PlanPreview from '../components/timeLog/PlanPreview';
import ResultsPanel from '../components/timeLog/ResultsPanel';
import useNonWorkingDays from '../hooks/useNonWorkingDays';
import useMinutosPorDia from '../hooks/useMinutosPorDia';
import usePlanReview from '../hooks/usePlanReview';
import useBatchSubmit from '../hooks/useBatchSubmit';
import {useOnTimeEntriesChanged, useTimeEntriesSignal} from '../contexts/TimeEntriesContext';
import {paraBinding, type Task, type TimeEntryReport} from '../types/backend';
import {formatDateBR, parseLocalDate, todayYMD, toYMD} from '../utils/dates';
import {errMsg} from '../utils/errors';
import {
    buildDeltaWorkDay,
    cellDelta,
    type EntryDefaults,
    entryDefaults,
    visibleDayIndexes,
    type WeekGrid,
    type WeekRow
} from '../utils/weekGrid';

const formatDate = (dateString: string) =>
    formatDateBR(dateString, 'dd/MM/yyyy (EEEE)', dateString || 'Data inválida', {locale: ptBR});

interface OpenCell {
    taskId: number;
    dayIndex: number;
    reduceBy: number;
}

// "Semana": timesheet tarefa × dia. Aumentar uma célula lança a diferença;
// reduzir abre os lançamentos da célula para editar/apagar um a um. "Copiar
// semana anterior" monta um plano revisável e envia pelo fluxo do TimeLog.
const Semana = () => {
    const [weekDate, setWeekDate] = useState(() => todayYMD());
    const [grid, setGrid] = useState<WeekGrid | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [showWeekend, setShowWeekend] = useState(false);
    const [savedTasks, setSavedTasks] = useState<Task[]>([]);

    const [addTarget, setAddTarget] = useState<AddTimeTarget | null>(null);
    const [addDefaults, setAddDefaults] = useState<EntryDefaults | null>(null);
    const [isAdding, setIsAdding] = useState(false);
    const [openCell, setOpenCell] = useState<OpenCell | null>(null);
    const [editing, setEditing] = useState<TimeEntryReport | null>(null);
    const [deletingId, setDeletingId] = useState<number | null>(null);
    const [isCopying, setIsCopying] = useState(false);

    const jornada = useMinutosPorDia();
    const {notifyChanged} = useTimeEntriesSignal();
    const today = todayYMD();

    const days = grid?.days ?? [];
    const nonWorkingDays = useNonWorkingDays(days[0] ?? weekDate, days[6] ?? weekDate);

    // Só a carga mais recente grava a grade (navegação rápida entre semanas).
    const loadRef = useRef(0);
    const loadGrid = useCallback(async (date: string): Promise<void> => {
        const requestId = ++loadRef.current;
        setIsLoading(true);
        setError(null);
        try {
            const data = await GetWeekGrid(date);
            if (requestId !== loadRef.current) return;
            setGrid(data);
        } catch (err) {
            if (requestId !== loadRef.current) return;
            console.error('Erro ao carregar a semana:', err);
            setError('Erro ao carregar a semana: ' + errMsg(err));
        } finally {
            if (requestId === loadRef.current) setIsLoading(false);
        }
    }, []);

    useEffect(() => {
        void loadGrid(weekDate);
    }, [weekDate, loadGrid]);

    useEffect(() => {
        GetSavedTasks()
            .then(tasks => setSavedTasks(tasks ?? []))
            .catch((err: unknown) => console.error('Erro ao carregar tarefas salvas:', err));
    }, []);

    const reloadGrid = useCallback(() => {
        void loadGrid(weekDate);
    }, [loadGrid, weekDate]);

    // Mudanças feitas em outras telas (gerenciador da Sidebar) refletem aqui.
    useOnTimeEntriesChanged(reloadGrid);

    const review = usePlanReview();
    const batch = useBatchSubmit({
        workDays: review.workDays,
        conflicts: review.conflicts,
        isCheckingConflicts: review.isCheckingConflicts,
        conflictCheckFailed: review.conflictCheckFailed,
        checkConflicts: review.checkConflicts,
        refreshCalendar: reloadGrid,
        reloadCalendar: reloadGrid,
        onError: setError
    });

    const goToWeek = (offsetDays: number) => {
        const base = parseLocalDate(grid?.weekStart ?? weekDate);
        if (!base) return;
        void review.setPlan([]);
        setWeekDate(toYMD(addDays(base, offsetDays)));
    };

    const goToToday = () => {
        void review.setPlan([]);
        setWeekDate(todayYMD());
    };

    const handleCellCommit = (row: WeekRow, dayIndex: number, target: number) => {
        const cell = row.cells?.[dayIndex];
        if (!cell) return;
        const delta = cellDelta(cell.minutes, target);
        if (delta.kind === 'add') {
            const naoUtil = nonWorkingDays[cell.date];
            setAddDefaults(entryDefaults(row.taskName, savedTasks.find(t => t.taskId === row.taskId)));
            setAddTarget({
                taskId: row.taskId,
                taskName: row.taskName || `Tarefa ${row.taskId}`,
                date: cell.date,
                currentMinutes: cell.minutes,
                addMinutes: delta.minutes,
                nonWorkingLabel: naoUtil ? (naoUtil.type === 'holiday' ? `feriado: ${naoUtil.name}` : 'fim de semana') : undefined
            });
        } else if (delta.kind === 'reduce') {
            setOpenCell({taskId: row.taskId, dayIndex, reduceBy: delta.minutes});
        }
    };

    const confirmAdd = async (values: EntryDefaults) => {
        if (!addTarget) return;
        setIsAdding(true);
        try {
            const plano = buildDeltaWorkDay(addTarget.date, addTarget.taskId, addTarget.addMinutes, values);
            const results = await LogMultipleTimes(paraBinding([plano]));
            const result = results?.[0];
            if (result?.success) {
                toast.success('Tempo lançado.');
                // A grade recarrega pelo sinal global, com o atraso que o
                // Teamwork precisa para refletir o lançamento.
                notifyChanged();
                setAddTarget(null);
            } else {
                toast.error('Erro ao lançar: ' + (result?.message || 'sem resposta do Teamwork'));
            }
        } catch (err) {
            console.error('Erro ao lançar tempo pela grade:', err);
            toast.error('Erro ao lançar: ' + errMsg(err));
        } finally {
            setIsAdding(false);
        }
    };

    const deleteEntry = async (entry: TimeEntryReport) => {
        if (!window.confirm(
            `Apagar o lançamento de ${entry.minutes} min ("${entry.description || 'sem descrição'}")?\n\n`
            + 'Esta ação não pode ser desfeita.'
        )) return;

        setDeletingId(entry.id);
        try {
            const results = await DeleteMultipleTimeEntries([entry.id]);
            const result = results?.[0];
            if (result?.success) {
                toast.success('Lançamento apagado.');
                notifyChanged();
            } else {
                toast.error('Erro ao apagar: ' + (result?.message || 'sem resposta do Teamwork'));
            }
        } catch (err) {
            console.error('Erro ao apagar lançamento:', err);
            toast.error('Erro ao apagar: ' + errMsg(err));
        } finally {
            setDeletingId(null);
        }
    };

    const copyPreviousWeek = async () => {
        if (!grid) return;
        setIsCopying(true);
        try {
            const copy = await BuildCopyPreviousWeekPlan(grid.weekStart);
            const plan = copy.plan ?? [];
            const pulados = copy.skippedDays ?? [];
            if (pulados.length > 0) {
                toast.info(`Dias não úteis pulados: ${pulados.map(d => formatDateBR(d, 'dd/MM')).join(', ')}.`);
            }
            if (plan.length === 0) {
                toast.info('A semana anterior não tem lançamentos para copiar.');
            } else {
                toast.success(`Plano de cópia com ${plan.length} dia(s). Revise antes de enviar.`);
            }
            await review.setPlan(plan);
        } catch (err) {
            console.error('Erro ao montar a cópia da semana anterior:', err);
            toast.error('Erro ao copiar a semana anterior: ' + errMsg(err));
        } finally {
            setIsCopying(false);
        }
    };

    // A lista da célula aberta vem sempre da grade atual: após editar/apagar,
    // a grade recarrega e a lista acompanha.
    const openCellTarget = useMemo(() => {
        if (!openCell || !grid) return null;
        const row = (grid.rows ?? []).find(r => r.taskId === openCell.taskId);
        const cell = row?.cells?.[openCell.dayIndex];
        if (!row || !cell) return null;
        return {
            taskName: row.taskName || `Tarefa ${row.taskId}`,
            date: cell.date,
            entries: cell.entries ?? [],
            reduceBy: openCell.reduceBy
        };
    }, [openCell, grid]);

    const visibleDays = visibleDayIndexes(showWeekend);
    const titulo = grid
        ? `${formatDateBR(grid.weekStart, 'dd/MM')} a ${formatDateBR(days[6] ?? grid.weekStart, 'dd/MM/yyyy')}`
        : '';

    return (
        <div>
            <div className="mb-6 flex flex-wrap justify-between items-center gap-3">
                <div>
                    <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">Semana</h1>
                    <p className="text-gray-600 dark:text-gray-400">Horas por tarefa e dia — edite direto na grade</p>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                    <button type="button" onClick={() => goToWeek(-7)} aria-label="Semana anterior" className="btn-secondary p-2">
                        <FiChevronLeft className="w-4 h-4" aria-hidden="true"/>
                    </button>
                    <span className="min-w-[10rem] text-center text-sm font-medium text-gray-800 dark:text-gray-200" aria-live="polite">
                        {titulo}
                    </span>
                    <button type="button" onClick={() => goToWeek(7)} aria-label="Próxima semana" className="btn-secondary p-2">
                        <FiChevronRight className="w-4 h-4" aria-hidden="true"/>
                    </button>
                    <button type="button" onClick={goToToday} className="btn-secondary text-sm">Hoje</button>
                    <button type="button" onClick={reloadGrid} aria-label="Recarregar semana" className="btn-secondary p-2">
                        <FiRefreshCw className={isLoading ? 'w-4 h-4 animate-spin' : 'w-4 h-4'} aria-hidden="true"/>
                    </button>
                </div>
            </div>

            {error && (
                <div role="alert" className="mb-6 bg-red-50 border-l-4 border-red-500 p-4 dark:bg-red-900/20 dark:border-red-700">
                    <div className="flex items-start">
                        <FiAlertCircle className="mt-0.5 w-5 h-5 text-red-500 mr-2" aria-hidden="true"/>
                        <p className="text-sm text-red-700 dark:text-red-200">{error}</p>
                    </div>
                </div>
            )}

            <div className="card mb-6">
                <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center">
                        <FiGrid className="w-5 h-5 mr-2" aria-hidden="true"/>
                        Grade semanal
                    </h2>
                    <div className="flex flex-wrap items-center gap-4">
                        <label className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                            <input type="checkbox" checked={showWeekend} onChange={e => setShowWeekend(e.target.checked)}/>
                            Mostrar sáb/dom
                        </label>
                        <button
                            type="button"
                            onClick={() => void copyPreviousWeek()}
                            disabled={!grid || isCopying}
                            className="btn-primary text-sm flex items-center disabled:opacity-50"
                        >
                            {isCopying
                                ? <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/>
                                : <FiCopy className="w-4 h-4 mr-2" aria-hidden="true"/>}
                            Copiar semana anterior
                        </button>
                    </div>
                </div>

                {!grid && isLoading ? (
                    <div className="flex justify-center py-10" role="status" aria-label="Carregando semana">
                        <div className="animate-spin-slow w-10 h-10 border-4 border-primary-600 border-t-transparent rounded-full"/>
                    </div>
                ) : grid && (grid.rows ?? []).length === 0 ? (
                    <p className="text-sm text-gray-600 dark:text-gray-400 py-6 text-center">
                        Nenhum lançamento nesta semana e nenhuma tarefa salva. Salve tarefas na tela &quot;Tarefas&quot; para
                        lançar pela grade.
                    </p>
                ) : grid && (
                    <>
                        <WeekGridTable
                            grid={grid}
                            visibleDays={visibleDays}
                            nonWorkingDays={nonWorkingDays}
                            jornada={jornada}
                            today={today}
                            disabled={isLoading || isAdding || deletingId !== null}
                            onCellCommit={handleCellCommit}
                            onOpenCell={(row, i) => setOpenCell({taskId: row.taskId, dayIndex: i, reduceBy: 0})}
                        />
                        <p className="mt-3 text-xs text-gray-500 dark:text-gray-400">
                            Digite o novo total da célula (ex.: 2:30, 2h30, 1,5). Aumentar lança só a diferença;
                            reduzir abre os lançamentos para você editar ou apagar. Dias abaixo da jornada ficam em
                            destaque; fins de semana e feriados, em cinza.
                        </p>
                    </>
                )}
            </div>

            <PlanPreview
                workDays={review.workDays}
                savedTasks={savedTasks}
                conflicts={review.conflicts}
                isCheckingConflicts={review.isCheckingConflicts}
                conflictCheckFailed={review.conflictCheckFailed}
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
                    onRefreshCalendar={reloadGrid}
                />
            )}

            {addTarget && addDefaults && (
                <AddTimeModal
                    key={`${addTarget.taskId}-${addTarget.date}-${addTarget.addMinutes}`}
                    target={addTarget}
                    defaults={addDefaults}
                    saving={isAdding}
                    onClose={() => setAddTarget(null)}
                    onConfirm={values => void confirmAdd(values)}
                />
            )}

            {openCellTarget && (
                <CellEntriesModal
                    target={openCellTarget}
                    deletingId={deletingId}
                    onClose={() => setOpenCell(null)}
                    onEdit={setEditing}
                    onDelete={entry => void deleteEntry(entry)}
                />
            )}

            <EditEntryModal
                key={editing?.id ?? 0}
                entry={editing}
                onClose={() => setEditing(null)}
                onSaved={() => {
                    setEditing(null);
                    notifyChanged();
                }}
            />
        </div>
    );
};

export default Semana;
