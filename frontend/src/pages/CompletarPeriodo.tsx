import {useCallback, useEffect, useState} from 'react';
import {toast} from 'react-toastify';
import {FiAlertCircle, FiLoader, FiPlay, FiTarget} from 'react-icons/fi';
import {ptBR} from 'date-fns/locale';
import {GetSavedTasks, GetTemplates, PlanFillGaps} from '@wailsjs/go/backend/App';
import TaskSelector from '../components/timeLog/TaskSelector';
import PlanPreview from '../components/timeLog/PlanPreview';
import ResultsPanel from '../components/timeLog/ResultsPanel';
import DaySummaryTable, {type DaySummary} from '../components/fillGaps/DaySummaryTable';
import usePlanReview from '../hooks/usePlanReview';
import useBatchSubmit from '../hooks/useBatchSubmit';
import {type Dados, paraBinding, type Task} from '../types/backend';
import type {backend} from '@wailsjs/go/models';
import {formatDateBR, todayYMD} from '../utils/dates';
import {errMsg} from '../utils/errors';
import {currentMonth, type Period, periodForMonth} from '../utils/fillGaps';
import {MINUTOS_POR_DIA_PADRAO} from '../utils/time';

const formatDate = (dateString: string) =>
    formatDateBR(dateString, 'dd/MM/yyyy (EEEE)', dateString || 'Data inválida', {locale: ptBR});

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white';
const labelClass = 'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';

type Origem = 'template' | 'tarefas';
type FillGapsRequest = Dados<backend.FillGapsRequest>;

const GRANULARIDADES = [5, 10, 15, 30, 60] as const;

// "Completar período": gera um plano que lança só o que falta em cada dia útil
// para atingir a jornada, a partir de um template ou de tarefas salvas. O envio
// reaproveita o fluxo do TimeLog (conflitos, reenviar falhas, desfazer).
const CompletarPeriodo = () => {
    const [savedTasks, setSavedTasks] = useState<Task[]>([]);
    const [templateNames, setTemplateNames] = useState<string[]>([]);
    const [origem, setOrigem] = useState<Origem>('template');
    const [templateName, setTemplateName] = useState('');
    const [selectedTasks, setSelectedTasks] = useState<number[]>([]);
    const [month, setMonth] = useState(() => currentMonth());
    const [wholeMonth, setWholeMonth] = useState(false);
    const [granularity, setGranularity] = useState(15);
    const [days, setDays] = useState<DaySummary[]>([]);
    const [minutesPerDay, setMinutesPerDay] = useState(MINUTOS_POR_DIA_PADRAO);
    // Último pedido calculado: o resumo é recalculado com ele após o envio,
    // mesmo que o formulário já tenha sido alterado.
    const [lastRequest, setLastRequest] = useState<FillGapsRequest | null>(null);
    const [isGenerating, setIsGenerating] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const review = usePlanReview();

    useEffect(() => {
        let cancelled = false;
        const load = async () => {
            try {
                const [tasks, templates] = await Promise.all([GetSavedTasks(), GetTemplates()]);
                if (cancelled) return;
                setSavedTasks(tasks ?? []);
                const nomes = Object.keys(templates ?? {}).sort((a, b) => a.localeCompare(b, 'pt-BR'));
                setTemplateNames(nomes);
                setTemplateName(prev => prev || nomes[0] || '');
                if (nomes.length === 0) setOrigem('tarefas');
            } catch (err) {
                if (cancelled) return;
                console.error('Erro ao carregar templates e tarefas:', err);
                setError('Erro ao carregar templates e tarefas: ' + errMsg(err));
            }
        };
        void load();
        return () => {
            cancelled = true;
        };
    }, []);

    const buildRequest = (period: Period): FillGapsRequest => ({
        start: period.start,
        end: period.end,
        templateName: origem === 'template' ? templateName : '',
        taskIds: origem === 'tarefas' ? selectedTasks : [],
        includeFuture: wholeMonth,
        granularity
    });

    const generate = async () => {
        const period = periodForMonth(month, wholeMonth, todayYMD());
        if (!period) {
            toast.warning('Mês futuro: marque "mês inteiro" para planejar dias que ainda não chegaram.');
            return;
        }
        if (origem === 'template' && !templateName) {
            toast.warning('Escolha um template.');
            return;
        }
        if (origem === 'tarefas' && selectedTasks.length === 0) {
            toast.warning('Selecione pelo menos uma tarefa salva.');
            return;
        }

        setIsGenerating(true);
        setError(null);
        try {
            const request = buildRequest(period);
            const result = await PlanFillGaps(paraBinding(request));
            setDays(result.days ?? []);
            setMinutesPerDay(result.minutesPerDay || MINUTOS_POR_DIA_PADRAO);
            setLastRequest(request);
            const plan = result.plan ?? [];
            if (plan.length === 0) {
                toast.info('Nada a completar: todos os dias do período já atingiram a jornada.');
            } else {
                toast.success(`Plano gerado para ${plan.length} dia(s).`);
            }
            await review.setPlan(plan);
        } catch (err) {
            console.error('Erro ao gerar plano de preenchimento:', err);
            toast.error('Erro ao gerar plano: ' + errMsg(err));
            setError('Erro ao gerar plano: ' + errMsg(err));
        } finally {
            setIsGenerating(false);
        }
    };

    // Depois do envio atualiza só o resumo: trocar o plano quebraria o
    // "reenviar só as falhas", que casa os resultados com o plano enviado.
    const refreshSummary = useCallback(() => {
        if (!lastRequest) return;
        void PlanFillGaps(paraBinding(lastRequest))
            .then(result => setDays(result.days ?? []))
            .catch((err: unknown) => console.error('Erro ao atualizar o resumo:', err));
    }, [lastRequest]);

    const batch = useBatchSubmit({
        workDays: review.workDays,
        conflicts: review.conflicts,
        isCheckingConflicts: review.isCheckingConflicts,
        conflictCheckFailed: review.conflictCheckFailed,
        checkConflicts: review.checkConflicts,
        refreshCalendar: refreshSummary,
        reloadCalendar: refreshSummary,
        onError: setError
    });

    const toggleTask = (taskId: number) => setSelectedTasks(prev => (
        prev.includes(taskId) ? prev.filter(id => id !== taskId) : [...prev, taskId]
    ));
    const toggleAll = () => setSelectedTasks(prev => (
        prev.length === savedTasks.length ? [] : savedTasks.map(t => t.taskId)
    ));

    return (
        <div>
            <div className="mb-6">
                <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">Completar Período</h1>
                <p className="text-gray-600 dark:text-gray-400">
                    Lança só o que falta em cada dia útil para fechar a jornada diária
                </p>
            </div>

            {error && (
                <div role="alert" className="mb-6 bg-red-50 border-l-4 border-red-500 p-4 dark:bg-red-900/20 dark:border-red-700">
                    <div className="flex items-start">
                        <FiAlertCircle className="mt-0.5 w-5 h-5 text-red-500 mr-2" aria-hidden="true"/>
                        <p className="text-sm text-red-700 dark:text-red-200">{error}</p>
                    </div>
                </div>
            )}

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
                {origem === 'tarefas' ? (
                    <TaskSelector
                        savedTasks={savedTasks}
                        selectedTasks={selectedTasks}
                        onToggle={toggleTask}
                        onToggleAll={toggleAll}
                    />
                ) : (
                    <div className="card lg:col-span-2">
                        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Template</h2>
                        <label htmlFor="fill-template" className={labelClass}>Template usado para completar</label>
                        <select
                            id="fill-template"
                            value={templateName}
                            onChange={e => setTemplateName(e.target.value)}
                            className={inputClass}
                        >
                            {templateNames.map(nome => <option key={nome} value={nome}>{nome}</option>)}
                        </select>
                        <p className="mt-3 text-sm text-gray-600 dark:text-gray-400">
                            As entradas do template são usadas na ordem em que aparecem, até completar a jornada;
                            a última é encurtada para não passar dela. Os dias da semana de cada tarefa são respeitados.
                        </p>
                    </div>
                )}

                <div className="card">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                        <FiTarget className="w-5 h-5 mr-2" aria-hidden="true"/>
                        Período
                    </h2>

                    <fieldset className="mb-4">
                        <legend className={labelClass}>Origem das entradas</legend>
                        <div className="flex gap-4 text-sm text-gray-700 dark:text-gray-300">
                            <label className="flex items-center gap-2">
                                <input
                                    type="radio"
                                    name="fill-origem"
                                    checked={origem === 'template'}
                                    onChange={() => setOrigem('template')}
                                    disabled={templateNames.length === 0}
                                />
                                Template
                            </label>
                            <label className="flex items-center gap-2">
                                <input
                                    type="radio"
                                    name="fill-origem"
                                    checked={origem === 'tarefas'}
                                    onChange={() => setOrigem('tarefas')}
                                />
                                Tarefas salvas
                            </label>
                        </div>
                    </fieldset>

                    <div className="mb-4">
                        <label htmlFor="fill-month" className={labelClass}>Mês</label>
                        <input
                            id="fill-month"
                            type="month"
                            value={month}
                            onChange={e => setMonth(e.target.value)}
                            className={inputClass}
                        />
                    </div>

                    <label className="flex items-start gap-2 mb-4 text-sm text-gray-700 dark:text-gray-300">
                        <input
                            type="checkbox"
                            className="mt-1"
                            checked={wholeMonth}
                            onChange={e => setWholeMonth(e.target.checked)}
                        />
                        <span>
                            Mês inteiro
                            <span className="block text-xs text-gray-500 dark:text-gray-400">
                                Inclui dias que ainda não chegaram. Sem isso, vai só até hoje.
                            </span>
                        </span>
                    </label>

                    <div className="mb-4">
                        <label htmlFor="fill-granularity" className={labelClass}>Granularidade mínima</label>
                        <select
                            id="fill-granularity"
                            value={granularity}
                            onChange={e => setGranularity(Number(e.target.value))}
                            className={inputClass}
                        >
                            {GRANULARIDADES.map(g => <option key={g} value={g}>{g} min</option>)}
                        </select>
                        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                            Dias em que falta menos que isso são considerados completos.
                        </p>
                    </div>

                    <button
                        type="button"
                        onClick={() => void generate()}
                        disabled={isGenerating}
                        className="btn-primary w-full flex items-center justify-center disabled:opacity-50"
                    >
                        {isGenerating
                            ? <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/>
                            : <FiPlay className="w-4 h-4 mr-2" aria-hidden="true"/>}
                        {isGenerating ? 'Calculando...' : 'Calcular o que falta'}
                    </button>
                </div>
            </div>

            <DaySummaryTable days={days} minutesPerDay={minutesPerDay} formatDate={formatDate}/>

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
                    onRefreshCalendar={refreshSummary}
                />
            )}
        </div>
    );
};

export default CompletarPeriodo;
