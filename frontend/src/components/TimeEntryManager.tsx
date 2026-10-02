import {useMemo, useState} from 'react';
import {toast} from 'react-toastify';
import {FiAlertCircle, FiLoader, FiRefreshCw, FiTrash2} from 'react-icons/fi';
import {DeleteMultipleTimeEntries} from '@wailsjs/go/backend/App';
import Modal from './Modal';
import useTimeEntries from '../hooks/useTimeEntries';
import EntryFilters, {type EntryFiltersState} from './timeEntries/EntryFilters';
import EntriesTable, {isDeletedEntry} from './timeEntries/EntriesTable';
import EditEntryModal from './timeEntries/EditEntryModal';
import DeleteResults from './timeEntries/DeleteResults';
import type {DateRange} from '../hooks/useTimeEntries';
import type {DeleteTimeEntryResult, TimeEntryReport} from '../types/backend';
import {errMsg} from '../utils/errors';
import {failedDeleteIds} from '../utils/retry';

const FILTROS_INICIAIS: EntryFiltersState = {
    projectName: '',
    taskName: '',
    minHours: '',
    maxHours: '',
    isBillable: 'all',
    status: 'active'
};

const applyFilters = (entries: readonly TimeEntryReport[], filters: EntryFiltersState): TimeEntryReport[] => {
    const projeto = filters.projectName.toLowerCase();
    const tarefa = filters.taskName.toLowerCase();
    const minMinutes = filters.minHours !== '' ? parseFloat(filters.minHours) * 60 : null;
    const maxMinutes = filters.maxHours !== '' ? parseFloat(filters.maxHours) * 60 : null;

    return entries.filter(entry => {
        if (projeto && !entry.projectName?.toLowerCase().includes(projeto)) return false;
        if (tarefa && !entry.taskName?.toLowerCase().includes(tarefa)) return false;
        if (minMinutes !== null && !Number.isNaN(minMinutes) && entry.minutes < minMinutes) return false;
        if (maxMinutes !== null && !Number.isNaN(maxMinutes) && entry.minutes > maxMinutes) return false;
        if (filters.isBillable !== 'all' && entry.isBillable !== (filters.isBillable === 'true')) return false;
        if (filters.status === 'active' && isDeletedEntry(entry)) return false;
        if (filters.status === 'deleted' && !isDeletedEntry(entry)) return false;
        return true;
    });
};

export interface EntriesChange {
    type: 'delete' | 'update';
    succeeded: number;
    failed: number;
}

interface TimeEntryManagerProps {
    isOpen: boolean;
    onClose: () => void;
    onEntriesChanged?: (change: EntriesChange) => void;
}

// Gerenciador de Apontamentos: lista, filtra, edita e exclui entradas de tempo.
//
// onEntriesChanged({type: 'delete' | 'update', succeeded, failed}) avisa quem
// abriu o gerenciador que algo mudou no Teamwork, com a contagem real — os
// toasts de sucesso/falha já são mostrados aqui.
const TimeEntryManager = ({isOpen, onClose, onEntriesChanged}: TimeEntryManagerProps) => {
    const {entries, loading, dateRange, setDateRange, showDeleted, setShowDeleted, reload} = useTimeEntries(isOpen);
    const [filters, setFilters] = useState<EntryFiltersState>(FILTROS_INICIAIS);
    const [selectedIds, setSelectedIds] = useState<Set<number>>(() => new Set());
    const [deleting, setDeleting] = useState(false);
    const [deleteResults, setDeleteResults] = useState<DeleteTimeEntryResult[]>([]);
    const [editingEntry, setEditingEntry] = useState<TimeEntryReport | null>(null);

    const filteredEntries = useMemo(() => applyFilters(entries, filters), [entries, filters]);

    const visibleActive = useMemo(
        () => filteredEntries.filter(entry => !isDeletedEntry(entry)),
        [filteredEntries]
    );

    // A seleção pode conter entradas que o filtro atual esconde. Elas não são
    // excluídas (só o que está visível), e a interface informa quantas são.
    const visibleSelectedIds = useMemo(
        () => visibleActive.filter(entry => selectedIds.has(entry.id)).map(entry => entry.id),
        [visibleActive, selectedIds]
    );
    const hiddenSelectedCount = selectedIds.size - visibleSelectedIds.length;

    const totals = useMemo(() => {
        let total = 0;
        let billable = 0;
        let deleted = 0;
        filteredEntries.forEach(entry => {
            if (isDeletedEntry(entry)) {
                deleted++;
                return;
            }
            total += entry.minutes || 0;
            if (entry.isBillable) billable += entry.minutes || 0;
        });
        return {hours: total / 60, billableHours: billable / 60, deleted};
    }, [filteredEntries]);

    const allVisibleSelected = visibleActive.length > 0 && visibleSelectedIds.length === visibleActive.length;

    // Recarrega a lista e zera seleção/resultados, como antes.
    const reloadEntries = async () => {
        const ok = await reload();
        if (ok) {
            setSelectedIds(new Set());
            setDeleteResults([]);
        }
        return ok;
    };

    const handleDateRangeChange = (range: DateRange) => {
        setSelectedIds(new Set());
        setDeleteResults([]);
        setDateRange(range);
    };

    const toggleDeleted = () => {
        const next = !showDeleted;
        setShowDeleted(next);
        setSelectedIds(new Set());
        setDeleteResults([]);
        // Sem isso o filtro de situação continuava em "só ativas" e as deletadas
        // buscadas no backend nunca apareciam.
        setFilters(prev => ({...prev, status: next ? 'all' : 'active'}));
    };

    const toggleEntrySelection = (entryId: number) => {
        setSelectedIds(prev => {
            const next = new Set(prev);
            if (next.has(entryId)) next.delete(entryId);
            else next.add(entryId);
            return next;
        });
    };

    const toggleAllVisible = () => {
        setSelectedIds(prev => {
            const next = new Set(prev);
            if (allVisibleSelected) {
                visibleActive.forEach(entry => next.delete(entry.id));
            } else {
                visibleActive.forEach(entry => next.add(entry.id));
            }
            return next;
        });
    };

    const clearHiddenSelection = () => setSelectedIds(new Set(visibleSelectedIds));

    // performDelete concentra a exclusão em lote para que o botão principal e o
    // "reenviar só as que falharam" compartilhem a mesma lógica. O backend faz o
    // lote com 3 exclusões simultâneas, respiro entre elas e repetição em rate
    // limit — um laço serial aqui desistiria de cada entrada no primeiro 429.
    const performDelete = async (ids: readonly number[]): Promise<void> => {
        if (!ids || ids.length === 0) return;

        setDeleting(true);
        setDeleteResults([]);

        try {
            const results: DeleteTimeEntryResult[] = (await DeleteMultipleTimeEntries([...ids])) ?? [];

            const successCount = results.filter(r => r.success).length;
            const failureCount = results.length - successCount;

            if (failureCount === 0) {
                toast.success(`${successCount} entradas deletadas com sucesso!`);
            } else if (successCount === 0) {
                toast.error(`Falha ao deletar todas as ${failureCount} entradas.`);
            } else {
                toast.warning(`${successCount} entradas deletadas, ${failureCount} falharam.`);
            }

            // reloadEntries recarrega a lista, limpa a seleção e esconde o
            // painel. Reexibimos o painel só quando há falhas — é quando o botão
            // de reenvio importa; num sucesso total, some como antes.
            await reloadEntries();
            setDeleteResults(failureCount > 0 ? results : []);

            if (onEntriesChanged) {
                onEntriesChanged({type: 'delete', succeeded: successCount, failed: failureCount});
            }
        } catch (error) {
            console.error('Erro ao deletar entradas:', error);
            toast.error('Erro ao deletar entradas: ' + errMsg(error));
        } finally {
            setDeleting(false);
        }
    };

    const deleteSelectedEntries = async () => {
        if (visibleSelectedIds.length === 0) {
            toast.warning('Selecione pelo menos uma entrada ativa visível para deletar.');
            return;
        }

        const avisoOcultas = hiddenSelectedCount > 0
            ? `\n\n${hiddenSelectedCount} entrada(s) selecionada(s) estão ocultas pelos filtros e NÃO serão deletadas.`
            : '';
        const confirmMessage = `Tem certeza que deseja deletar ${visibleSelectedIds.length} entrada(s) de tempo?${avisoOcultas}\n\nEsta ação não pode ser desfeita.`;

        if (!window.confirm(confirmMessage)) {
            return;
        }

        await performDelete(visibleSelectedIds);
    };

    const retryFailedDeletes = async () => {
        const failedIds = failedDeleteIds(deleteResults);
        if (failedIds.length === 0) return;
        await performDelete(failedIds);
    };

    const handleEntrySaved = async () => {
        setEditingEntry(null);
        await reloadEntries();
        if (onEntriesChanged) {
            onEntriesChanged({type: 'update', succeeded: 1, failed: 0});
        }
    };

    return (
        <>
            <Modal
                isOpen={isOpen}
                onClose={onClose}
                size="2xl"
                title="Gerenciar Apontamentos de Horas"
                closeDisabled={deleting}
                footer={
                    <button
                        type="button"
                        onClick={onClose}
                        disabled={deleting}
                        className="px-4 py-2 bg-gray-600 hover:bg-gray-700 text-white rounded-md disabled:opacity-50"
                    >
                        Fechar
                    </button>
                }
            >
                <EntryFilters
                    dateRange={dateRange}
                    onDateRangeChange={handleDateRangeChange}
                    showDeleted={showDeleted}
                    onToggleDeleted={toggleDeleted}
                    filters={filters}
                    onFiltersChange={setFilters}
                />

                <div className="flex flex-wrap justify-between items-center gap-2 mb-4">
                    <div className="flex items-center space-x-4">
                        <button
                            type="button"
                            onClick={() => void reloadEntries()}
                            disabled={loading}
                            className="flex items-center px-3 py-2 text-sm bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 rounded-lg"
                        >
                            <FiRefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} aria-hidden="true"/>
                            Atualizar
                        </button>

                        <div className="text-sm text-gray-600 dark:text-gray-400" aria-live="polite">
                            <span className="font-medium">{filteredEntries.length}</span> entradas •{' '}
                            <span className="font-medium">{totals.hours.toFixed(1)}h</span> total •{' '}
                            <span className="font-medium">{totals.billableHours.toFixed(1)}h</span> contabilizável
                            {showDeleted && totals.deleted > 0 && (
                                <span className="text-red-600 dark:text-red-400 ml-2">
                                    • <span className="font-medium">{totals.deleted}</span> deletadas
                                </span>
                            )}
                        </div>
                    </div>

                    <div className="flex items-center space-x-2">
                        {visibleActive.length > 0 && (
                            <button
                                type="button"
                                onClick={toggleAllVisible}
                                className="text-sm text-primary-600 hover:text-primary-700 dark:text-primary-500"
                            >
                                {allVisibleSelected ? 'Desmarcar todas' : 'Selecionar todas'}
                            </button>
                        )}

                        {visibleSelectedIds.length > 0 && (
                            <button
                                type="button"
                                onClick={() => void deleteSelectedEntries()}
                                disabled={deleting}
                                className="flex items-center px-3 py-2 text-sm bg-red-600 hover:bg-red-700 text-white rounded-lg disabled:opacity-50"
                            >
                                {deleting ? (
                                    <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/>
                                ) : (
                                    <FiTrash2 className="w-4 h-4 mr-2" aria-hidden="true"/>
                                )}
                                Deletar ({visibleSelectedIds.length})
                            </button>
                        )}
                    </div>
                </div>

                {hiddenSelectedCount > 0 && (
                    <div role="status"
                         className="mb-4 flex items-center justify-between p-3 bg-amber-50 dark:bg-amber-900/20 border-l-4 border-amber-500 rounded text-sm text-amber-800 dark:text-amber-300">
                        <span className="flex items-center">
                            <FiAlertCircle className="w-4 h-4 mr-2 flex-shrink-0" aria-hidden="true"/>
                            {hiddenSelectedCount} entrada(s) selecionada(s) estão ocultas pelos filtros e não
                            serão deletadas.
                        </span>
                        <button
                            type="button"
                            onClick={clearHiddenSelection}
                            className="ml-3 underline whitespace-nowrap"
                        >
                            Desmarcar ocultas
                        </button>
                    </div>
                )}

                <EntriesTable
                    entries={filteredEntries}
                    selectedIds={selectedIds}
                    allSelected={allVisibleSelected}
                    onToggleAll={toggleAllVisible}
                    onToggleEntry={toggleEntrySelection}
                    onEdit={setEditingEntry}
                    loading={loading}
                />

                <DeleteResults
                    results={deleteResults}
                    deleting={deleting}
                    onRetryFailed={() => void retryFailedDeletes()}
                />
            </Modal>

            <EditEntryModal
                entry={isOpen ? editingEntry : null}
                onClose={() => setEditingEntry(null)}
                onSaved={handleEntrySaved}
            />
        </>
    );
};

export default TimeEntryManager;
