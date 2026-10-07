import {FiLoader, FiTrash2} from 'react-icons/fi';
import {ptBR} from 'date-fns/locale';
import Modal from '../Modal';
import {formatDateBR} from '../../utils/dates';
import {formatHoursMinutes} from '../../utils/time';
import type {TimeEntryReport} from '../../types/backend';

export interface DeleteRequest {
    // Lançamentos que serão apagados.
    entries: TimeEntryReport[];
    // Lançamento mantido (duplicatas), só para exibição.
    kept?: TimeEntryReport;
}

interface ConfirmDeleteEntriesModalProps {
    request: DeleteRequest;
    deleting: boolean;
    onCancel: () => void;
    onConfirm: () => void;
}

const Linha = ({entry}: {entry: TimeEntryReport}) => (
    <li className="py-2 text-sm">
        <p className="text-gray-900 dark:text-white">
            <span className="capitalize">{formatDateBR(entry.date, 'EEE, dd/MM', entry.date, {locale: ptBR})}</span>
            {' · '}<span className="tabular-nums">{formatHoursMinutes(entry.minutes)}</span>
            {' · '}{entry.taskName || entry.projectName || 'Sem tarefa'}
        </p>
        <p className="text-gray-600 dark:text-gray-400 wrap-break-word">{entry.description?.trim() || 'Sem descrição'}</p>
    </li>
);

// Confirmação de exclusão listando exatamente o que será apagado (e, nas
// duplicatas, o que fica).
const ConfirmDeleteEntriesModal = ({request, deleting, onCancel, onConfirm}: ConfirmDeleteEntriesModalProps) => {
    const n = request.entries.length;
    return (
        <Modal
            isOpen
            onClose={onCancel}
            closeDisabled={deleting}
            zIndex="z-60"
            title={n === 1 ? 'Apagar lançamento?' : `Apagar ${n} lançamentos?`}
            icon={<FiTrash2 className="w-5 h-5 mr-2 text-red-600" aria-hidden="true"/>}
            footer={
                <>
                    <button type="button" onClick={onCancel} disabled={deleting} className="btn-secondary disabled:opacity-50">
                        Cancelar
                    </button>
                    <button type="button" onClick={onConfirm} disabled={deleting}
                            className="btn-danger inline-flex items-center disabled:opacity-50">
                        {deleting
                            ? <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/>
                            : <FiTrash2 className="w-4 h-4 mr-2" aria-hidden="true"/>}
                        Apagar
                    </button>
                </>
            }
        >
            <p className="text-sm text-gray-700 dark:text-gray-300 mb-2">
                {n === 1 ? 'Este lançamento será apagado' : 'Estes lançamentos serão apagados'} no Teamwork.
                Esta ação não pode ser desfeita.
            </p>
            <ul className="divide-y divide-gray-200 dark:divide-gray-700" aria-label="Lançamentos a apagar">
                {request.entries.map(e => <Linha key={e.id} entry={e}/>)}
            </ul>
            {request.kept && (
                <div className="mt-4">
                    <p className="text-sm font-medium text-green-700 dark:text-green-400">Fica mantido:</p>
                    <ul aria-label="Lançamento mantido"><Linha entry={request.kept}/></ul>
                </div>
            )}
        </Modal>
    );
};

export default ConfirmDeleteEntriesModal;
