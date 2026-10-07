import {FiEdit2, FiEye, FiEyeOff, FiList, FiLoader, FiTarget, FiTrash2} from 'react-icons/fi';
import {ptBR} from 'date-fns/locale';
import {formatDateBR} from '../../utils/dates';
import {formatHoursMinutes} from '../../utils/time';
import type {AuditIssue, TimeEntryReport} from '../../types/backend';

export interface AuditIssueHandlers {
    onCompletePeriod: (issue: AuditIssue) => void;
    onEdit: (entry: TimeEntryReport) => void;
    onDeleteDuplicates: (issue: AuditIssue) => void;
    onDelete: (entry: TimeEntryReport) => void;
    onViewDay: (issue: AuditIssue) => void;
    onIgnore: (issue: AuditIssue) => void;
    onUnignore: (issue: AuditIssue) => void;
}

interface AuditIssueItemProps extends AuditIssueHandlers {
    issue: AuditIssue;
    // Chave do problema com uma ação em andamento (ignorar/reexibir).
    busyKey: string | null;
}

const acaoClass = 'inline-flex items-center px-3 py-1.5 text-xs font-medium rounded-lg border focus:outline-hidden focus:ring-2 disabled:opacity-50';
const acaoPrimaria = `${acaoClass} text-white bg-primary-600 border-primary-600 hover:bg-primary-700 focus:ring-primary-300 dark:focus:ring-primary-800`;
const acaoNeutra = `${acaoClass} text-gray-700 bg-white border-gray-300 hover:bg-gray-100 focus:ring-gray-200 dark:bg-gray-800 dark:text-gray-200 dark:border-gray-600 dark:hover:bg-gray-700`;
const acaoPerigo = `${acaoClass} text-red-700 bg-white border-red-300 hover:bg-red-50 focus:ring-red-200 dark:bg-gray-800 dark:text-red-400 dark:border-red-800 dark:hover:bg-red-900/20`;

const EntryLine = ({entry}: {entry: TimeEntryReport}) => (
    <li className="text-xs text-gray-600 dark:text-gray-400 wrap-break-word">
        <span className="font-medium text-gray-800 dark:text-gray-200 tabular-nums">{formatHoursMinutes(entry.minutes)}</span>
        {' · '}{entry.taskName || entry.projectName || 'Sem tarefa'}
        {' · '}<span className="italic">{entry.description?.trim() || 'sem descrição'}</span>
    </li>
);

// Um problema da auditoria com as ações de um clique da verificação que o
// gerou (ver audit.Action no Go).
const AuditIssueItem = ({issue, busyKey, ...h}: AuditIssueItemProps) => {
    const entries = issue.entries ?? [];
    const first = entries[0];
    const busy = busyKey === issue.key;
    const erro = issue.severity === 'error';

    return (
        <li className={`py-3 flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between ${issue.ignored ? 'opacity-60' : ''}`}
            data-testid={`issue-${issue.key}`}>
            <div className="min-w-0">
                <p className="text-sm text-gray-900 dark:text-white">
                    <span className={`inline-block w-2 h-2 rounded-full mr-2 align-middle ${erro ? 'bg-red-500' : 'bg-amber-500'}`}
                          aria-hidden="true"/>
                    <span className="font-medium capitalize">
                        {formatDateBR(issue.date, 'EEE, dd/MM', issue.date, {locale: ptBR})}
                    </span>
                    {' — '}{issue.message}
                    {issue.ignored && <span className="ml-2 text-xs text-gray-500 dark:text-gray-400">(ignorado)</span>}
                </p>
                {issue.type !== 'incomplete_day' && issue.type !== 'over_daily_limit' && entries.length > 0 && (
                    <ul className="mt-1 ml-4 space-y-0.5">
                        {entries.map(e => <EntryLine key={e.id} entry={e}/>)}
                    </ul>
                )}
            </div>

            <div className="flex flex-wrap gap-2 shrink-0">
                {!issue.ignored && issue.action === 'complete_period' && (
                    <button type="button" className={acaoPrimaria} onClick={() => h.onCompletePeriod(issue)}>
                        <FiTarget className="w-3.5 h-3.5 mr-1" aria-hidden="true"/>Completar
                    </button>
                )}
                {!issue.ignored && (issue.action === 'edit' || issue.action === 'edit_or_delete') && first && (
                    <button type="button" className={acaoPrimaria} onClick={() => h.onEdit(first)}>
                        <FiEdit2 className="w-3.5 h-3.5 mr-1" aria-hidden="true"/>Editar
                    </button>
                )}
                {!issue.ignored && issue.action === 'edit_or_delete' && first && (
                    <button type="button" className={acaoPerigo} onClick={() => h.onDelete(first)}>
                        <FiTrash2 className="w-3.5 h-3.5 mr-1" aria-hidden="true"/>Apagar
                    </button>
                )}
                {!issue.ignored && issue.action === 'delete_duplicates' && (
                    <button type="button" className={acaoPerigo} onClick={() => h.onDeleteDuplicates(issue)}>
                        <FiTrash2 className="w-3.5 h-3.5 mr-1" aria-hidden="true"/>
                        Apagar {(issue.deleteEntryIds ?? []).length === 1 ? 'a cópia' : `${(issue.deleteEntryIds ?? []).length} cópias`}
                    </button>
                )}
                {!issue.ignored && issue.action === 'view_day' && (
                    <button type="button" className={acaoPrimaria} onClick={() => h.onViewDay(issue)}>
                        <FiList className="w-3.5 h-3.5 mr-1" aria-hidden="true"/>Ver lançamentos
                    </button>
                )}
                {issue.ignored ? (
                    <button type="button" className={acaoNeutra} disabled={busy} onClick={() => h.onUnignore(issue)}>
                        {busy ? <FiLoader className="w-3.5 h-3.5 mr-1 animate-spin" aria-hidden="true"/> : <FiEye className="w-3.5 h-3.5 mr-1" aria-hidden="true"/>}
                        Reexibir
                    </button>
                ) : (
                    <button type="button" className={acaoNeutra} disabled={busy} onClick={() => h.onIgnore(issue)}
                            title="Não mostrar mais este problema">
                        {busy ? <FiLoader className="w-3.5 h-3.5 mr-1 animate-spin" aria-hidden="true"/> : <FiEyeOff className="w-3.5 h-3.5 mr-1" aria-hidden="true"/>}
                        Ignorar
                    </button>
                )}
            </div>
        </li>
    );
};

export default AuditIssueItem;
