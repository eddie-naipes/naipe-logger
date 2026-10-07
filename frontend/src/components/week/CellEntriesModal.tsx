import {FiEdit2, FiList, FiLoader, FiTrash2} from 'react-icons/fi';
import {ptBR} from 'date-fns/locale';
import Modal from '../Modal';
import {formatDateBR} from '../../utils/dates';
import {formatHoursMinutes} from '../../utils/time';
import type {TimeEntryReport} from '../../types/backend';

export interface CellEntriesTarget {
    taskName: string;
    date: string;
    entries: TimeEntryReport[];
    // Quanto o usuário pediu para reduzir (0 quando só abriu a lista).
    reduceBy: number;
}

interface CellEntriesModalProps {
    target: CellEntriesTarget;
    deletingId: number | null;
    onClose: () => void;
    onEdit: (entry: TimeEntryReport) => void;
    onDelete: (entry: TimeEntryReport) => void;
}

// Lançamentos de uma célula (tarefa × dia). Reduzir uma célula nunca apaga
// nada sozinho: o usuário escolhe aqui o que editar ou apagar.
const CellEntriesModal = ({target, deletingId, onClose, onEdit, onDelete}: CellEntriesModalProps) => {
    const total = target.entries.reduce((s, e) => s + (e.minutes || 0), 0);

    return (
        <Modal
            isOpen
            onClose={onClose}
            closeDisabled={deletingId !== null}
            size="lg"
            title={`${target.taskName} — ${formatDateBR(target.date, 'EEEE, dd/MM', target.date, {locale: ptBR})}`}
            icon={<FiList className="w-5 h-5 mr-2" aria-hidden="true"/>}
            footer={
                <button type="button" onClick={onClose} disabled={deletingId !== null} className="btn-secondary disabled:opacity-50">
                    Fechar
                </button>
            }
        >
            {target.reduceBy > 0 && (
                <p role="status" className="mb-4 p-3 rounded-lg text-sm bg-blue-50 text-blue-800 dark:bg-blue-900/20 dark:text-blue-300">
                    Para reduzir {formatHoursMinutes(target.reduceBy)}, edite a duração de um lançamento ou apague-o.
                    Nada é apagado automaticamente.
                </p>
            )}

            {target.entries.length === 0 ? (
                <p className="text-sm text-gray-600 dark:text-gray-400">Nenhum lançamento nesta célula.</p>
            ) : (
                <ul className="divide-y divide-gray-200 dark:divide-gray-700">
                    {target.entries.map(entry => (
                        <li key={entry.id} className="py-3 flex items-start justify-between gap-4">
                            <div className="min-w-0">
                                <p className="text-sm font-medium text-gray-900 dark:text-white">
                                    {formatHoursMinutes(entry.minutes)}
                                    {entry.isBillable ? '' : ' • não contabilizável'}
                                </p>
                                <p className="text-sm text-gray-600 dark:text-gray-400 wrap-break-word">
                                    {entry.description || 'Sem descrição'}
                                </p>
                            </div>
                            <div className="flex gap-2 shrink-0">
                                <button
                                    type="button"
                                    onClick={() => onEdit(entry)}
                                    disabled={deletingId !== null}
                                    aria-label={`Editar lançamento de ${formatHoursMinutes(entry.minutes)}`}
                                    className="p-2 rounded-sm text-gray-500 hover:text-primary-600 hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50"
                                >
                                    <FiEdit2 className="w-4 h-4" aria-hidden="true"/>
                                </button>
                                <button
                                    type="button"
                                    onClick={() => onDelete(entry)}
                                    disabled={deletingId !== null}
                                    aria-label={`Apagar lançamento de ${formatHoursMinutes(entry.minutes)}`}
                                    className="p-2 rounded-sm text-gray-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-50"
                                >
                                    {deletingId === entry.id
                                        ? <FiLoader className="w-4 h-4 animate-spin" aria-hidden="true"/>
                                        : <FiTrash2 className="w-4 h-4" aria-hidden="true"/>}
                                </button>
                            </div>
                        </li>
                    ))}
                </ul>
            )}

            <p className="mt-4 text-right text-sm text-gray-600 dark:text-gray-400">
                Total: <strong>{formatHoursMinutes(total)}</strong>
            </p>
        </Modal>
    );
};

export default CellEntriesModal;
