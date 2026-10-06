import {useState} from 'react';
import {FiLoader, FiPlus} from 'react-icons/fi';
import {ptBR} from 'date-fns/locale';
import Modal from '../Modal';
import {formatDateBR} from '../../utils/dates';
import {formatHoursMinutes} from '../../utils/time';
import type {EntryDefaults} from '../../utils/weekGrid';

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white';
const labelClass = 'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';

export interface AddTimeTarget {
    taskId: number;
    taskName: string;
    date: string;
    currentMinutes: number;
    addMinutes: number;
    nonWorkingLabel?: string | undefined;
}

interface AddTimeModalProps {
    target: AddTimeTarget;
    defaults: EntryDefaults;
    saving: boolean;
    onClose: () => void;
    onConfirm: (values: EntryDefaults) => void;
}

// Confirma o lançamento da diferença ao aumentar uma célula da grade. A
// descrição e o billable vêm da tarefa salva, mas podem ser ajustados aqui.
// Quem usa deve passar key para recriar o formulário a cada célula.
const AddTimeModal = ({target, defaults, saving, onClose, onConfirm}: AddTimeModalProps) => {
    const [form, setForm] = useState<EntryDefaults>(defaults);
    const podeSalvar = !saving && form.description.trim() !== '' && /^\d{2}:\d{2}$/.test(form.time);

    return (
        <Modal
            isOpen
            onClose={onClose}
            closeDisabled={saving}
            title="Adicionar tempo"
            icon={<FiPlus className="w-5 h-5 mr-2" aria-hidden="true"/>}
            footer={
                <>
                    <button type="button" onClick={onClose} disabled={saving} className="btn-secondary disabled:opacity-50">
                        Cancelar
                    </button>
                    <button
                        type="button"
                        onClick={() => onConfirm(form)}
                        disabled={!podeSalvar}
                        className="btn-primary flex items-center disabled:opacity-50"
                    >
                        {saving && <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/>}
                        Lançar {formatHoursMinutes(target.addMinutes)}
                    </button>
                </>
            }
        >
            <p className="text-sm text-gray-700 dark:text-gray-300 mb-4">
                <strong>{target.taskName}</strong> em{' '}
                {formatDateBR(target.date, "EEEE, dd/MM/yyyy", target.date, {locale: ptBR})}: de{' '}
                {formatHoursMinutes(target.currentMinutes)} para{' '}
                {formatHoursMinutes(target.currentMinutes + target.addMinutes)}. Será criado um lançamento de{' '}
                <strong>{formatHoursMinutes(target.addMinutes)}</strong>.
            </p>

            {target.nonWorkingLabel && (
                <p role="alert" className="mb-4 p-3 rounded-lg text-sm bg-amber-50 text-amber-800 dark:bg-amber-900/20 dark:text-amber-300">
                    Atenção: este dia não é útil ({target.nonWorkingLabel}).
                </p>
            )}

            <div className="space-y-4">
                <div>
                    <label htmlFor="add-time-description" className={labelClass}>Descrição *</label>
                    <textarea
                        id="add-time-description"
                        rows={2}
                        value={form.description}
                        onChange={e => setForm(prev => ({...prev, description: e.target.value}))}
                        className={inputClass}
                        disabled={saving}
                    />
                </div>
                <div className="grid grid-cols-2 gap-4">
                    <div>
                        <label htmlFor="add-time-start" className={labelClass}>Horário de início</label>
                        <input
                            id="add-time-start"
                            type="time"
                            value={form.time}
                            onChange={e => setForm(prev => ({...prev, time: e.target.value}))}
                            className={inputClass}
                            disabled={saving}
                        />
                    </div>
                    <label className="flex items-center gap-2 mt-6 text-sm text-gray-700 dark:text-gray-300">
                        <input
                            type="checkbox"
                            checked={form.isBillable}
                            onChange={e => setForm(prev => ({...prev, isBillable: e.target.checked}))}
                            disabled={saving}
                        />
                        Contabilizável (billable)
                    </label>
                </div>
            </div>
        </Modal>
    );
};

export default AddTimeModal;
