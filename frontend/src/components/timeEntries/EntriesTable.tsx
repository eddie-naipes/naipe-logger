import {FiEdit} from 'react-icons/fi';
import type {TimeEntryReport} from '../../types/backend';
import {formatDateBR} from '../../utils/dates';

export const isDeletedEntry = (entry: Pick<TimeEntryReport, 'status'>): boolean => entry.status === 'deleted';

const thClass = 'px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase';
const tdClass = 'px-3 py-2 text-sm text-gray-900 dark:text-white';

interface EntriesTableProps {
    entries: readonly TimeEntryReport[];
    selectedIds: ReadonlySet<number>;
    allSelected: boolean;
    onToggleAll: () => void;
    onToggleEntry: (entryId: number) => void;
    onEdit: (entry: TimeEntryReport) => void;
    loading: boolean;
}

const EntriesTable = ({entries, selectedIds, allSelected, onToggleAll, onToggleEntry, onEdit, loading}: EntriesTableProps) => {
    if (loading) {
        return (
            <div className="flex justify-center items-center py-12" role="status" aria-label="Carregando entradas">
                <div className="animate-spin w-8 h-8 border-4 border-primary-600 border-t-transparent rounded-full"></div>
            </div>
        );
    }

    const selectable = entries.filter(e => !isDeletedEntry(e));

    return (
        <div className="overflow-y-auto max-h-96 border border-gray-200 rounded-lg dark:border-gray-700">
            {entries.length === 0 ? (
                <div className="text-center py-8">
                    <p className="text-gray-500 dark:text-gray-400">
                        Nenhuma entrada de tempo encontrada no período selecionado.
                    </p>
                </div>
            ) : (
                <table className="w-full">
                    <thead className="bg-gray-50 dark:bg-gray-700">
                    <tr>
                        <th className="w-8 px-3 py-2">
                            <input
                                type="checkbox"
                                checked={allSelected}
                                disabled={selectable.length === 0}
                                onChange={onToggleAll}
                                aria-label="Selecionar todas as entradas ativas visíveis"
                                className="w-4 h-4 text-primary-600 rounded-sm"
                            />
                        </th>
                        <th className={thClass}>Status</th>
                        <th className={thClass}>Data</th>
                        <th className={thClass}>Projeto</th>
                        <th className={thClass}>Tarefa</th>
                        <th className={thClass}>Descrição</th>
                        <th className={thClass}>Tempo</th>
                        <th className={thClass}>Horas</th>
                        <th className={thClass}>Contabilizável</th>
                        <th className={thClass}>Ações</th>
                    </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                    {entries.map((entry) => {
                        const isDeleted = isDeletedEntry(entry);
                        const isSelected = selectedIds.has(entry.id);
                        const dataFormatada = formatDateBR(entry.date, 'dd/MM/yyyy', entry.date);

                        return (
                            <tr
                                key={entry.id}
                                className={`${
                                    isDeleted
                                        ? 'bg-red-50 dark:bg-red-900/10 opacity-75'
                                        : 'hover:bg-gray-50 dark:hover:bg-gray-700'
                                } ${
                                    isSelected && !isDeleted ? 'bg-primary-50 dark:bg-primary-900/20' : ''
                                }`}
                            >
                                <td className="px-3 py-2">
                                    {!isDeleted && (
                                        <input
                                            type="checkbox"
                                            checked={isSelected}
                                            onChange={() => onToggleEntry(entry.id)}
                                            aria-label={`Selecionar entrada de ${dataFormatada} em ${entry.taskName || 'tarefa sem nome'}`}
                                            className="w-4 h-4 text-primary-600 rounded-sm"
                                        />
                                    )}
                                </td>
                                <td className="px-3 py-2">
                                    <span className={`inline-flex px-2 py-1 text-xs rounded-full ${
                                        isDeleted
                                            ? 'bg-red-100 text-red-800 dark:bg-red-800 dark:text-red-100'
                                            : 'bg-green-100 text-green-800 dark:bg-green-800 dark:text-green-100'
                                    }`}>
                                        {isDeleted ? 'Deletado' : 'Ativo'}
                                    </span>
                                </td>
                                <td className={tdClass}>{dataFormatada}</td>
                                <td className={tdClass}>{entry.projectName || 'N/A'}</td>
                                <td className={tdClass}>{entry.taskName || 'N/A'}</td>
                                <td className={`${tdClass} max-w-xs truncate`}>{entry.description || 'Sem descrição'}</td>
                                <td className={tdClass}>{entry.minutes || 0} min</td>
                                <td className={tdClass}>{((entry.minutes || 0) / 60).toFixed(2)}h</td>
                                <td className="px-3 py-2">
                                    <span className={`inline-flex px-2 py-1 text-xs rounded-full ${
                                        entry.isBillable
                                            ? 'bg-green-100 text-green-800 dark:bg-green-800 dark:text-green-100'
                                            : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
                                    }`}>
                                        {entry.isBillable ? 'Sim' : 'Não'}
                                    </span>
                                </td>
                                <td className="px-3 py-2">
                                    {!isDeleted && (
                                        <button
                                            type="button"
                                            onClick={() => onEdit(entry)}
                                            className="p-1 text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
                                            title="Editar entrada"
                                            aria-label={`Editar entrada de ${dataFormatada}`}
                                        >
                                            <FiEdit className="w-4 h-4" aria-hidden="true"/>
                                        </button>
                                    )}
                                </td>
                            </tr>
                        );
                    })}
                    </tbody>
                </table>
            )}
        </div>
    );
};

export default EntriesTable;
