import {useMemo, useState} from 'react';
import {toast} from 'react-toastify';
import {useLocation, useNavigate} from 'react-router';
import {
    FiAlertCircle,
    FiAward,
    FiCheckCircle,
    FiCheckSquare,
    FiChevronLeft,
    FiChevronRight,
    FiLoader,
    FiRefreshCw,
    FiSettings
} from 'react-icons/fi';
import {ptBR} from 'date-fns/locale';
import {DeleteMultipleTimeEntries, IgnoreAuditIssue, UnignoreAuditIssue} from '@wailsjs/go/backend/App';
import AuditIssueItem from '../components/audit/AuditIssueItem';
import AuditSettingsSection from '../components/audit/AuditSettingsSection';
import ConfirmDeleteEntriesModal, {type DeleteRequest} from '../components/audit/ConfirmDeleteEntriesModal';
import {groupIssues, shiftMonth, typeInfo} from '../components/audit/auditLabels';
import CellEntriesModal, {type CellEntriesTarget} from '../components/week/CellEntriesModal';
import EditEntryModal from '../components/timeEntries/EditEntryModal';
import {useTimeEntriesSignal} from '../contexts/TimeEntriesContext';
import useMonthAudit from '../hooks/useMonthAudit';
import type {AuditIssue, TimeEntryReport} from '../types/backend';
import {formatDateBR} from '../utils/dates';
import {errMsg} from '../utils/errors';
import {currentMonth, monthFromNavigationState} from '../utils/fillGaps';
import {formatHoursMinutes} from '../utils/time';

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block p-2 dark:bg-gray-700 dark:border-gray-600 dark:text-white';

const nomeDoMes = (month: string): string =>
    formatDateBR(`${month}-01`, "MMMM 'de' yyyy", month, {locale: ptBR});

// "Fechamento do mês": antes de entregar o mês, mostra tudo o que está errado
// nos lançamentos (backend/audit) com a correção a um clique. Depois de cada
// correção a auditoria roda de novo (via notifyChanged → useOnTimeEntriesChanged).
const Fechamento = () => {
    const location = useLocation();
    const navigate = useNavigate();
    const {notifyChanged} = useTimeEntriesSignal();

    const [month, setMonth] = useState(() => monthFromNavigationState(location.state) ?? currentMonth());
    const {result, loading, error: loadError, reload} = useMonthAudit(month);
    const error = loadError ? 'Não foi possível auditar o mês: ' + loadError : null;
    const [showIgnored, setShowIgnored] = useState(false);
    const [showSettings, setShowSettings] = useState(false);
    const [busyKey, setBusyKey] = useState<string | null>(null);

    const [editing, setEditing] = useState<TimeEntryReport | null>(null);
    const [dayView, setDayView] = useState<CellEntriesTarget | null>(null);
    const [deleteRequest, setDeleteRequest] = useState<DeleteRequest | null>(null);
    const [deleting, setDeleting] = useState(false);

    const issues = useMemo(() => result?.issues ?? [], [result]);
    const visiveis = useMemo(() => issues.filter(i => showIgnored || !i.ignored), [issues, showIgnored]);
    const grupos = useMemo(() => groupIssues(visiveis), [visiveis]);
    const summary = result?.summary;
    const pendentes = summary ? summary.errorCount + summary.warningCount : 0;

    // --- Ações --------------------------------------------------------------

    const toggleIgnore = async (issue: AuditIssue, ignore: boolean) => {
        setBusyKey(issue.key);
        try {
            if (ignore) {
                await IgnoreAuditIssue(issue.key);
                toast.info('Problema ignorado. Use "Mostrar ignorados" para reexibir.');
            } else {
                await UnignoreAuditIssue(issue.key);
            }
            reload();
        } catch (err) {
            toast.error('Erro ao atualizar o problema: ' + errMsg(err));
        } finally {
            setBusyKey(null);
        }
    };

    const confirmDelete = async () => {
        if (!deleteRequest) return;
        const ids = deleteRequest.entries.map(e => e.id);
        setDeleting(true);
        try {
            const results = (await DeleteMultipleTimeEntries(ids)) ?? [];
            const ok = results.filter(r => r.success).length;
            const falhas = results.filter(r => !r.success);
            if (ok > 0) toast.success(ok === 1 ? 'Lançamento apagado.' : `${ok} lançamentos apagados.`);
            if (falhas.length > 0 || results.length === 0) {
                toast.error('Erro ao apagar: ' + (falhas[0]?.message || 'sem resposta do Teamwork'));
            }
            setDeleteRequest(null);
            setDayView(null);
            if (ok > 0) notifyChanged();
        } catch (err) {
            console.error('Erro ao apagar lançamentos:', err);
            toast.error('Erro ao apagar: ' + errMsg(err));
        } finally {
            setDeleting(false);
        }
    };

    const handlers = {
        onCompletePeriod: () => void navigate('/completar', {state: {month}}),
        onEdit: (entry: TimeEntryReport) => setEditing(entry),
        onDelete: (entry: TimeEntryReport) => setDeleteRequest({entries: [entry]}),
        onDeleteDuplicates: (issue: AuditIssue) => {
            const apagar = new Set(issue.deleteEntryIds ?? []);
            const entries = issue.entries ?? [];
            setDeleteRequest({
                entries: entries.filter(e => apagar.has(e.id)),
                kept: entries.find(e => e.id === issue.keepEntryId)
            });
        },
        onViewDay: (issue: AuditIssue) => {
            const limite = summary?.dailyLimitMinutes ?? 0;
            setDayView({
                taskName: 'Lançamentos do dia',
                date: issue.date,
                entries: issue.entries ?? [],
                reduceBy: limite > 0 ? Math.max(0, (issue.minutes ?? 0) - limite) : 0
            });
        },
        onIgnore: (issue: AuditIssue) => void toggleIgnore(issue, true),
        onUnignore: (issue: AuditIssue) => void toggleIgnore(issue, false)
    };

    // --- Render ---------------------------------------------------------------

    const progresso = summary && summary.expectedToDateMinutes > 0
        ? Math.min(100, Math.round(summary.loggedToDateMinutes / summary.expectedToDateMinutes * 100))
        : 100;

    return (
        <div className="space-y-6">
            <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                    <h1 className="text-2xl font-semibold text-gray-900 dark:text-white flex items-center">
                        <FiCheckSquare className="w-6 h-6 mr-2" aria-hidden="true"/>
                        Fechamento do mês
                    </h1>
                    <p className="text-gray-600 dark:text-gray-400">Confira e corrija os lançamentos antes de entregar o mês.</p>
                </div>
                <div className="flex items-center gap-2">
                    <button type="button" onClick={() => setMonth(m => shiftMonth(m, -1))} aria-label="Mês anterior"
                            className="p-2 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-600 dark:text-gray-300">
                        <FiChevronLeft className="w-5 h-5" aria-hidden="true"/>
                    </button>
                    <input type="month" aria-label="Mês" className={inputClass} value={month}
                           onChange={e => e.target.value && setMonth(e.target.value)}/>
                    <button type="button" onClick={() => setMonth(m => shiftMonth(m, 1))} aria-label="Próximo mês"
                            className="p-2 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-600 dark:text-gray-300">
                        <FiChevronRight className="w-5 h-5" aria-hidden="true"/>
                    </button>
                    <button type="button" onClick={reload} disabled={loading} aria-label="Auditar novamente"
                            title="Auditar novamente"
                            className="p-2 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50">
                        <FiRefreshCw className={`w-5 h-5 text-gray-500 dark:text-gray-400 ${loading ? 'animate-spin' : ''}`} aria-hidden="true"/>
                    </button>
                </div>
            </div>

            {error && (
                <div role="alert" className="flex items-start p-4 bg-red-50 dark:bg-red-900/20 border-l-4 border-red-500 rounded-sm">
                    <FiAlertCircle className="w-5 h-5 mr-2 text-red-500 shrink-0" aria-hidden="true"/>
                    <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
                </div>
            )}

            {loading && !result && (
                <div className="flex justify-center py-12" role="status" aria-label="Auditando o mês">
                    <FiLoader className="w-8 h-8 animate-spin text-primary-600" aria-hidden="true"/>
                </div>
            )}

            {summary && (
                <section className={`card p-4! space-y-3 ${loading ? 'opacity-60' : ''}`} aria-busy={loading}
                         aria-labelledby="resumo-fechamento">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                        <h2 id="resumo-fechamento" className="text-lg font-semibold text-gray-900 dark:text-white capitalize">
                            {nomeDoMes(month)}
                        </h2>
                        {summary.ready ? (
                            <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300">
                                <FiCheckCircle className="w-4 h-4 mr-1" aria-hidden="true"/>Pronto para entregar
                            </span>
                        ) : (
                            <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300">
                                <FiAlertCircle className="w-4 h-4 mr-1" aria-hidden="true"/>
                                {summary.errorCount === 1 ? '1 erro a corrigir' : `${summary.errorCount} erros a corrigir`}
                            </span>
                        )}
                    </div>

                    <div>
                        <div className="flex justify-between text-sm text-gray-700 dark:text-gray-300 mb-1">
                            <span>Lançado até hoje: <strong className="tabular-nums">{formatHoursMinutes(summary.loggedToDateMinutes)}</strong> de {formatHoursMinutes(summary.expectedToDateMinutes)}</span>
                            <span className="tabular-nums">{progresso}%</span>
                        </div>
                        <div className="w-full h-2.5 rounded-full bg-gray-200 dark:bg-gray-700" role="progressbar"
                             aria-valuemin={0} aria-valuemax={100} aria-valuenow={progresso} aria-label="Progresso do mês">
                            <div className={`h-2.5 rounded-full ${summary.ready ? 'bg-green-500' : 'bg-primary-600'}`}
                                 style={{width: `${progresso}%`}}/>
                        </div>
                        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                            Mês inteiro: {formatHoursMinutes(summary.loggedMinutes)} de {formatHoursMinutes(summary.expectedMinutes)}
                            {' '}({summary.workingDays} dias úteis × {formatHoursMinutes(summary.minutesPerDay)}) · {summary.entryCount} lançamentos
                        </p>
                    </div>

                    {(result?.counts ?? []).length > 0 && (
                        <ul className="flex flex-wrap gap-2" aria-label="Problemas por tipo">
                            {(result?.counts ?? []).map(c => (
                                <li key={c.type}
                                    className={`px-2.5 py-1 rounded-full text-xs font-medium ${c.severity === 'error'
                                        ? 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300'
                                        : 'bg-amber-50 text-amber-800 dark:bg-amber-900/20 dark:text-amber-300'}`}>
                                    {typeInfo(c.type).title}: {c.count}
                                </li>
                            ))}
                        </ul>
                    )}

                    <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
                        <label className="inline-flex items-center gap-2 text-gray-700 dark:text-gray-300">
                            <input type="checkbox" checked={showIgnored} onChange={e => setShowIgnored(e.target.checked)}
                                   className="w-4 h-4 rounded-sm border-gray-300 text-primary-600 focus:ring-primary-500"/>
                            Mostrar ignorados ({summary.ignoredCount})
                        </label>
                        <button type="button" onClick={() => setShowSettings(s => !s)} aria-expanded={showSettings}
                                className="inline-flex items-center text-primary-600 dark:text-primary-400 hover:underline">
                            <FiSettings className="w-4 h-4 mr-1" aria-hidden="true"/>Configurações da auditoria
                        </button>
                    </div>

                    {showSettings && (
                        <div className="pt-3 border-t border-gray-200 dark:border-gray-700">
                            <AuditSettingsSection onSaved={reload}/>
                        </div>
                    )}
                </section>
            )}

            {result && pendentes === 0 && grupos.length === 0 && (
                <div className="card text-center py-10">
                    <FiAward className="w-12 h-12 mx-auto text-green-500" aria-hidden="true"/>
                    <h2 className="mt-3 text-xl font-semibold text-gray-900 dark:text-white">Tudo certo!</h2>
                    <p className="mt-1 text-gray-600 dark:text-gray-400">
                        Nenhum problema em {nomeDoMes(month)}. Pode entregar o mês tranquilo.
                    </p>
                </div>
            )}

            {grupos.map(g => {
                const info = typeInfo(g.type);
                const ativos = g.issues.filter(i => !i.ignored).length;
                const headingId = `grupo-${g.type}`;
                return (
                    <section key={g.type} className="card p-4!" aria-labelledby={headingId}>
                        <h2 id={headingId} className="text-base font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                            {info.title}
                            <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${g.severity === 'error'
                                ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
                                : 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300'}`}>
                                {ativos}
                            </span>
                        </h2>
                        {info.hint && <p className="text-xs text-gray-500 dark:text-gray-400">{info.hint}</p>}
                        <ul className="divide-y divide-gray-200 dark:divide-gray-700">
                            {g.issues.map(issue => (
                                <AuditIssueItem key={issue.key} issue={issue} busyKey={busyKey} {...handlers}/>
                            ))}
                        </ul>
                    </section>
                );
            })}

            {dayView && (
                <CellEntriesModal
                    target={dayView}
                    deletingId={null}
                    onClose={() => setDayView(null)}
                    onEdit={entry => {
                        setDayView(null);
                        setEditing(entry);
                    }}
                    onDelete={entry => setDeleteRequest({entries: [entry]})}
                />
            )}

            {deleteRequest && (
                <ConfirmDeleteEntriesModal
                    request={deleteRequest}
                    deleting={deleting}
                    onCancel={() => setDeleteRequest(null)}
                    onConfirm={() => void confirmDelete()}
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

export default Fechamento;
