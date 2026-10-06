import {useCallback, useEffect, useMemo, useRef, useState} from 'react';
import {toast} from 'react-toastify';
import {FiAlertCircle, FiCalendar, FiEye, FiLoader, FiSearch} from 'react-icons/fi';
import {ptBR} from 'date-fns/locale';
import {
    GetAgendaSettings,
    GetSavedTasks,
    MarkAgendaImported,
    PlanAgenda,
    SaveAgendaSettings,
    UnmarkAgendaImported
} from '@wailsjs/go/backend/App';
import type {agenda, config} from '@wailsjs/go/models';
import CalendarsCard from '../components/agenda/CalendarsCard';
import RulesCard, {type AgendaRule, type AgendaSettings} from '../components/agenda/RulesCard';
import EventsTable from '../components/agenda/EventsTable';
import CreateRuleModal from '../components/agenda/CreateRuleModal';
import PlanPreview from '../components/timeLog/PlanPreview';
import ResultsPanel from '../components/timeLog/ResultsPanel';
import usePlanReview from '../hooks/usePlanReview';
import useBatchSubmit from '../hooks/useBatchSubmit';
import {paraBinding, type Task} from '../types/backend';
import {
    type AgendaItem,
    type AgendaPeriodo,
    type AgendaTaskRef,
    casarImportados,
    itemSelecionavel,
    itensDoPlano,
    montarWorkDays,
    periodoAgenda,
    tarefaDoItem,
    tarefasDoPlano
} from '../utils/agenda';
import {formatDateBR} from '../utils/dates';
import {errMsg} from '../utils/errors';

const formatDate = (dateString: string) =>
    formatDateBR(dateString, 'dd/MM (EEE)', dateString || 'Data inválida', {locale: ptBR});
const formatDateLong = (dateString: string) =>
    formatDateBR(dateString, 'dd/MM/yyyy (EEEE)', dateString || 'Data inválida', {locale: ptBR});

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white';
const labelClass = 'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';

const ATALHOS: {id: AgendaPeriodo; label: string}[] = [
    {id: 'hoje', label: 'Hoje'},
    {id: 'ontem', label: 'Ontem'},
    {id: 'semana', label: 'Esta semana'},
];

// "Agenda": importa reuniões de agendas iCal (link privado ou arquivo .ics),
// mapeia cada evento para uma tarefa por regras e lança com o horário real.
// Agendas e regras ficam nesta página (e não em Configurações) porque são
// usadas junto com a lista: dá para criar uma regra a partir de um evento sem
// sair daqui. O envio reaproveita conflitos, reenviar falhas e desfazer.
const Agenda = () => {
    const [period, setPeriod] = useState(() => periodoAgenda('hoje'));
    const [settings, setSettings] = useState<AgendaSettings | null>(null);
    const [savedTasks, setSavedTasks] = useState<Task[]>([]);
    const [items, setItems] = useState<AgendaItem[]>([]);
    const [warnings, setWarnings] = useState<string[]>([]);
    const [searched, setSearched] = useState(false);
    const [selected, setSelected] = useState<Set<string>>(new Set());
    const [chosenTasks, setChosenTasks] = useState<Map<string, AgendaTaskRef>>(new Map());
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [ruleFrom, setRuleFrom] = useState<AgendaItem | null>(null);
    const [savingRule, setSavingRule] = useState(false);

    // Eventos do plano enviado, na ordem do plano (casam os resultados).
    const [planItems, setPlanItems] = useState<AgendaItem[]>([]);
    // Chave do evento de cada entrada criada, para desmarcar ao desfazer.
    const entryKeys = useRef(new Map<number, string>());
    const markedKeys = useRef(new Set<string>());
    const pendingUndo = useRef<number[] | null>(null);
    // Eventos do lote efetivamente enviado: uma prévia nova não pode ser
    // casada com os resultados do lote anterior.
    const submittedItems = useRef<AgendaItem[]>([]);

    const review = usePlanReview();

    useEffect(() => {
        let cancelled = false;
        Promise.all([GetAgendaSettings(), GetSavedTasks()])
            .then(([cfg, tasks]) => {
                if (cancelled) return;
                setSettings(cfg);
                setSavedTasks(tasks ?? []);
            })
            .catch((err: unknown) => {
                if (!cancelled) setError('Erro ao carregar a configuração da agenda: ' + errMsg(err));
            });
        return () => {
            cancelled = true;
        };
    }, []);

    const search = useCallback(async (p: {start: string; end: string}) => {
        if (!p.start || !p.end || p.end < p.start) {
            toast.warning('Informe um período válido.');
            return;
        }
        setIsLoading(true);
        setError(null);
        try {
            const plan = await PlanAgenda(p.start, p.end);
            const lista = plan.items ?? [];
            setItems(lista);
            setWarnings(plan.warnings ?? []);
            setSearched(true);
            // Mapeados entram selecionados; o resto o usuário escolhe.
            setSelected(new Set(lista.filter(i => i.status === 'mapped').map(i => i.key)));
            setChosenTasks(new Map());
            for (const aviso of plan.warnings ?? []) toast.warning(aviso);
        } catch (err) {
            setError('Erro ao ler a agenda: ' + errMsg(err));
            setItems([]);
        } finally {
            setIsLoading(false);
        }
    }, []);

    const applyShortcut = (atalho: AgendaPeriodo) => {
        const p = periodoAgenda(atalho);
        setPeriod(p);
        void search(p);
    };

    const batch = useBatchSubmit({
        workDays: review.workDays,
        conflicts: review.conflicts,
        isCheckingConflicts: review.isCheckingConflicts,
        conflictCheckFailed: review.conflictCheckFailed,
        checkConflicts: review.checkConflicts,
        refreshCalendar: () => undefined,
        reloadCalendar: () => undefined,
        onError: setError
    });

    const setItemsStatus = (keys: ReadonlySet<string>, status: string, reason: string) =>
        setItems(prev => prev.map(i => (keys.has(i.key) ? {...i, status, reason} : i)));

    // Mantém o registro de "já lançado" em dia com os resultados do lote:
    // marca os sucessos (inclusive os do reenvio) e, depois de um desfazer,
    // desmarca as entradas apagadas.
    const {results, isSubmitting, isRetrying, isUndoing} = batch;
    useEffect(() => {
        if (isSubmitting || isRetrying) {
            // Um desfazer cancelado na confirmação não pode ser confundido com
            // as entradas que somem dos resultados num envio novo.
            pendingUndo.current = null;
            return;
        }
        if (isUndoing) return;

        if (pendingUndo.current) {
            const vivos = new Set(results.map(r => r.entryId));
            const keys = pendingUndo.current
                .filter(id => !vivos.has(id))
                .map(id => entryKeys.current.get(id))
                .filter((k): k is string => !!k);
            pendingUndo.current = null;
            if (keys.length === 0) return;
            keys.forEach(k => markedKeys.current.delete(k));
            void UnmarkAgendaImported(keys)
                .then(() => setItemsStatus(new Set(keys), 'mapped', 'Lançamento desfeito'))
                .catch((err: unknown) => toast.error('Erro ao atualizar os eventos importados: ' + errMsg(err)));
            return;
        }

        const novos = casarImportados(results, submittedItems.current).filter(r => !markedKeys.current.has(r.key));
        if (novos.length === 0) return;
        novos.forEach(r => {
            markedKeys.current.add(r.key);
            if (r.entryId > 0) entryKeys.current.set(r.entryId, r.key);
        });
        void MarkAgendaImported(paraBinding<agenda.ImportedRecord[]>(novos))
            .then(() => {
                const keys = new Set(novos.map(r => r.key));
                setItemsStatus(keys, 'imported', 'Lançado agora');
                setSelected(prev => new Set([...prev].filter(k => !keys.has(k))));
            })
            .catch((err: unknown) => toast.error('Erro ao registrar os eventos importados: ' + errMsg(err)));
    }, [results, isSubmitting, isRetrying, isUndoing]);

    const submit = async () => {
        submittedItems.current = planItems;
        await batch.submitPlan();
    };

    const undo = async () => {
        pendingUndo.current = batch.undoableEntries.map(r => r.entryId);
        await batch.undoBatch();
    };

    const selectedForPlan = useMemo(
        () => itensDoPlano(items, selected, chosenTasks),
        [items, selected, chosenTasks]
    );

    const buildPreview = async () => {
        if (selectedForPlan.length === 0) {
            toast.warning('Selecione ao menos um evento com tarefa.');
            return;
        }
        setPlanItems(selectedForPlan);
        await review.setPlan(montarWorkDays(selectedForPlan));
    };

    const toggle = (key: string) => setSelected(prev => {
        const next = new Set(prev);
        if (next.has(key)) next.delete(key); else next.add(key);
        return next;
    });

    const toggleAll = () => {
        const elegiveis = items.filter(i => itemSelecionavel(i) && i.minutes > 0 && tarefaDoItem(i, chosenTasks));
        const todos = elegiveis.every(i => selected.has(i.key));
        setSelected(todos ? new Set() : new Set(elegiveis.map(i => i.key)));
    };

    const chooseTask = (key: string, task: AgendaTaskRef | null) => {
        setChosenTasks(prev => {
            const next = new Map(prev);
            if (task) next.set(key, task); else next.delete(key);
            return next;
        });
        setSelected(prev => {
            const next = new Set(prev);
            if (task) next.add(key); else next.delete(key);
            return next;
        });
    };

    const createRule = async (rule: AgendaRule) => {
        if (!settings) return;
        setSavingRule(true);
        const next = {...settings, rules: [...settings.rules, rule]};
        try {
            await SaveAgendaSettings(paraBinding<config.AgendaSettings>(next));
            setSettings(next);
            setRuleFrom(null);
            toast.success('Regra criada.');
            if (searched) await search(period);
        } catch (err) {
            toast.error('Erro ao criar a regra: ' + errMsg(err));
        } finally {
            setSavingRule(false);
        }
    };

    const contagem = useMemo(() => {
        const c = {mapped: 0, unmapped: 0, ignored: 0, imported: 0};
        for (const i of items) {
            if (i.status in c) c[i.status as keyof typeof c]++;
        }
        return c;
    }, [items]);

    return (
        <div>
            <div className="mb-6">
                <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">Agenda</h1>
                <p className="text-gray-600 dark:text-gray-400">
                    Importa as reuniões da sua agenda (Google, Outlook ou arquivo .ics) como lançamentos de horas
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
                <div className="lg:col-span-2 space-y-6">
                    <div className="card">
                        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                            <FiCalendar className="w-5 h-5 mr-2" aria-hidden="true"/>
                            Período
                        </h2>
                        <div className="flex flex-wrap gap-2 mb-4">
                            {ATALHOS.map(a => (
                                <button key={a.id} type="button" onClick={() => applyShortcut(a.id)} disabled={isLoading}
                                        className="btn-secondary text-sm disabled:opacity-50">
                                    {a.label}
                                </button>
                            ))}
                        </div>
                        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 items-end">
                            <div>
                                <label htmlFor="agenda-inicio" className={labelClass}>De</label>
                                <input id="agenda-inicio" type="date" value={period.start}
                                       onChange={e => setPeriod(p => ({...p, start: e.target.value}))} className={inputClass}/>
                            </div>
                            <div>
                                <label htmlFor="agenda-fim" className={labelClass}>Até</label>
                                <input id="agenda-fim" type="date" value={period.end}
                                       onChange={e => setPeriod(p => ({...p, end: e.target.value}))} className={inputClass}/>
                            </div>
                            <button type="button" onClick={() => void search(period)} disabled={isLoading}
                                    className="btn-primary flex items-center justify-center disabled:opacity-50">
                                {isLoading
                                    ? <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/>
                                    : <FiSearch className="w-4 h-4 mr-2" aria-hidden="true"/>}
                                {isLoading ? 'Lendo agendas...' : 'Buscar eventos'}
                            </button>
                        </div>
                    </div>

                    {searched && (
                        <div className="card">
                            <div className="flex flex-wrap items-center justify-between gap-2 mb-4">
                                <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Eventos</h2>
                                <p className="text-xs text-gray-600 dark:text-gray-400">
                                    {contagem.mapped} mapeado(s) • {contagem.unmapped} sem regra • {contagem.ignored} ignorado(s)
                                    • {contagem.imported} já lançado(s)
                                </p>
                            </div>
                            {warnings.length > 0 && (
                                <ul className="mb-3 text-xs text-amber-700 dark:text-amber-300 list-disc pl-5">
                                    {warnings.map(w => <li key={w}>{w}</li>)}
                                </ul>
                            )}
                            <EventsTable
                                items={items}
                                selected={selected}
                                chosenTasks={chosenTasks}
                                savedTasks={savedTasks}
                                formatDate={formatDate}
                                onToggle={toggle}
                                onChooseTask={chooseTask}
                                onCreateRule={setRuleFrom}
                            />
                            {items.length > 0 && (
                                <div className="mt-4 flex flex-wrap justify-between gap-2">
                                    <button type="button" onClick={toggleAll} className="btn-secondary text-sm">
                                        Marcar/desmarcar todos
                                    </button>
                                    <button type="button" onClick={() => void buildPreview()}
                                            disabled={selectedForPlan.length === 0}
                                            className="btn-primary flex items-center disabled:opacity-50">
                                        <FiEye className="w-4 h-4 mr-2" aria-hidden="true"/>
                                        Gerar prévia ({selectedForPlan.length})
                                    </button>
                                </div>
                            )}
                        </div>
                    )}
                </div>

                <div className="space-y-6">
                    <CalendarsCard/>
                    {settings && (
                        <RulesCard
                            settings={settings}
                            savedTasks={savedTasks}
                            onChange={setSettings}
                            onSaved={() => {
                                if (searched) void search(period);
                            }}
                        />
                    )}
                </div>
            </div>

            <PlanPreview
                workDays={review.workDays}
                savedTasks={tarefasDoPlano(planItems)}
                conflicts={review.conflicts}
                isCheckingConflicts={review.isCheckingConflicts}
                conflictCheckFailed={review.conflictCheckFailed}
                isSubmitting={batch.isSubmitting}
                onSubmit={() => void submit()}
                formatDate={formatDateLong}
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
                    onUndo={() => void undo()}
                    onRefreshCalendar={() => void search(period)}
                />
            )}

            {ruleFrom && (
                <CreateRuleModal
                    item={ruleFrom}
                    savedTasks={savedTasks}
                    initialTask={chosenTasks.get(ruleFrom.key) ?? null}
                    saving={savingRule}
                    onClose={() => setRuleFrom(null)}
                    onCreate={rule => void createRule(rule)}
                />
            )}
        </div>
    );
};

export default Agenda;
