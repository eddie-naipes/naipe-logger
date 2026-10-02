import {useState} from 'react';
import {toast} from 'react-toastify';
import {FiEdit, FiLoader, FiSave} from 'react-icons/fi';
import {ptBR} from 'date-fns/locale';
import {UpdateTimeEntry} from '@wailsjs/go/backend/App';
import Modal from '../Modal';
import TimeInputComponent from '../TimeInputComponent';
import {formatDateBR} from '../../utils/dates';
import {errMsg} from '../../utils/errors';
import {hoursAndMinutesToMinutes, minutesToHoursAndMinutes} from '../../utils/time';
import {paraBinding, type TimeEntry, type TimeEntryReport} from '../../types/backend';

interface EntryForm {
    hours: number;
    minutes: number;
    description: string;
    time: string;
    date: string;
    isBillable: boolean;
}

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white';
const labelClass = 'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';

const formFromEntry = (entry: TimeEntryReport | null): EntryForm => {
    const {hours, minutes} = minutesToHoursAndMinutes(entry?.minutes || 0);
    return {
        hours,
        minutes,
        description: entry?.description || '',
        time: entry?.startTime ? entry.startTime.substring(0, 5) : '09:00',
        date: entry?.date || '',
        isBillable: entry?.isBillable !== undefined ? entry.isBillable : true
    };
};

// Edição de uma entrada de tempo. onSaved() é chamado só quando a API confirma.
// O formulário nasce da entrada recebida: quem usa deve passar key={entry.id}
// para que trocar de entrada recrie o estado (em vez de um efeito copiando
// props para o estado).
interface EditEntryModalProps {
    entry: TimeEntryReport | null;
    onClose: () => void;
    onSaved: () => Promise<void> | void;
}

const EditEntryModal = ({entry, onClose, onSaved}: EditEntryModalProps) => {
    const [form, setForm] = useState<EntryForm>(() => formFromEntry(entry));
    const [updating, setUpdating] = useState(false);


    const setField = <K extends keyof EntryForm>(field: K, value: EntryForm[K]) =>
        setForm(prev => ({...prev, [field]: value}));
    const totalMinutes = hoursAndMinutesToMinutes(form.hours, form.minutes);
    const podeSalvar = !updating && form.description.trim() !== '' && totalMinutes > 0 && form.date !== '';

    const save = async () => {
        if (!entry) return;

        if (totalMinutes <= 0) {
            toast.warning('Informe um tempo válido (maior que 0 minutos).');
            return;
        }
        if (!form.description.trim()) {
            toast.warning('Informe uma descrição para a entrada.');
            return;
        }

        setUpdating(true);
        try {
            const updatedEntry: TimeEntry = {
                minutes: totalMinutes,
                description: form.description.trim(),
                time: form.time + ':00',
                date: form.date,
                isBillable: form.isBillable,
                userId: entry.userId || 0
            };

            const result = await UpdateTimeEntry(entry.id, paraBinding(updatedEntry));

            if (result && result.success) {
                toast.success('Entrada atualizada com sucesso!');
                await onSaved();
            } else {
                toast.error('Erro ao atualizar entrada: ' + (result?.message || 'Erro desconhecido'));
            }
        } catch (error) {
            console.error('Erro ao atualizar entrada:', error);
            toast.error('Erro ao atualizar entrada: ' + errMsg(error));
        } finally {
            setUpdating(false);
        }
    };

    return (
        <Modal
            isOpen={Boolean(entry)}
            onClose={onClose}
            size="lg"
            zIndex="z-[60]"
            closeDisabled={updating}
            title="Editar Entrada de Tempo"
            icon={<FiEdit className="w-5 h-5 mr-2" aria-hidden="true"/>}
            footer={
                <>
                    <button
                        type="button"
                        onClick={onClose}
                        disabled={updating}
                        className="px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 dark:bg-gray-700 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        Cancelar
                    </button>
                    <button
                        type="button"
                        onClick={() => void save()}
                        disabled={!podeSalvar}
                        className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500 disabled:opacity-50 disabled:cursor-not-allowed dark:bg-primary-700 dark:hover:bg-primary-800 flex items-center"
                    >
                        {updating ? (
                            <>
                                <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/>
                                Salvando...
                            </>
                        ) : (
                            <>
                                <FiSave className="w-4 h-4 mr-2" aria-hidden="true"/>
                                Salvar Alterações
                            </>
                        )}
                    </button>
                </>
            }
        >
            {entry && (
                <div className="space-y-6">
                    <div className="p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
                        <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">Entrada Original</h3>
                        <dl className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                            <div>
                                <dt className="text-gray-500 dark:text-gray-400">Projeto:</dt>
                                <dd className="font-medium text-gray-900 dark:text-white">{entry.projectName || 'N/A'}</dd>
                            </div>
                            <div>
                                <dt className="text-gray-500 dark:text-gray-400">Tarefa:</dt>
                                <dd className="font-medium text-gray-900 dark:text-white">{entry.taskName || 'N/A'}</dd>
                            </div>
                            <div>
                                <dt className="text-gray-500 dark:text-gray-400">Data Original:</dt>
                                <dd className="font-medium text-gray-900 dark:text-white">
                                    {formatDateBR(entry.date, 'dd/MM/yyyy', entry.date)}
                                </dd>
                            </div>
                            <div>
                                <dt className="text-gray-500 dark:text-gray-400">Tempo Original:</dt>
                                <dd className="font-medium text-gray-900 dark:text-white">
                                    {entry.minutes} min ({((entry.minutes || 0) / 60).toFixed(2)}h)
                                </dd>
                            </div>
                        </dl>
                    </div>

                    <div className="space-y-4">
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div>
                                <label htmlFor="edit-entry-date" className={labelClass}>Data *</label>
                                <input
                                    id="edit-entry-date"
                                    type="date"
                                    value={form.date}
                                    onChange={(e) => setField('date', e.target.value)}
                                    className={inputClass}
                                    required
                                    disabled={updating}
                                />
                            </div>
                            <div>
                                <label htmlFor="edit-entry-time" className={labelClass}>Horário *</label>
                                <input
                                    id="edit-entry-time"
                                    type="time"
                                    value={form.time}
                                    onChange={(e) => setField('time', e.target.value)}
                                    className={inputClass}
                                    required
                                    disabled={updating}
                                />
                            </div>
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <TimeInputComponent
                                hours={form.hours}
                                minutes={form.minutes}
                                onTimeChange={(hours, minutes) => setForm(prev => ({...prev, hours, minutes}))}
                                label="Duração *"
                                disabled={updating}
                                showTotalMinutes={true}
                            />
                            <div>
                                <label htmlFor="edit-entry-billable" className={labelClass}>Contabilizável</label>
                                <select
                                    id="edit-entry-billable"
                                    value={form.isBillable.toString()}
                                    onChange={(e) => setField('isBillable', e.target.value === 'true')}
                                    className={inputClass}
                                    disabled={updating}
                                >
                                    <option value="true">Sim - Contabilizável</option>
                                    <option value="false">Não - Não Contabilizável</option>
                                </select>
                            </div>
                        </div>

                        <div>
                            <label htmlFor="edit-entry-description" className={labelClass}>Descrição *</label>
                            <textarea
                                id="edit-entry-description"
                                value={form.description}
                                onChange={(e) => setField('description', e.target.value)}
                                rows={3}
                                className={inputClass}
                                placeholder="Descreva o trabalho realizado..."
                                required
                                disabled={updating}
                            />
                            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                                Caracteres: {form.description.length}
                            </p>
                        </div>

                        <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
                            <h3 className="text-sm font-medium text-blue-800 dark:text-blue-300 mb-2">
                                Preview das Alterações
                            </h3>
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                                <div>
                                    <span className="text-blue-600 dark:text-blue-400">Data:</span>
                                    <p className="font-medium">
                                        {formatDateBR(form.date, 'dd/MM/yyyy', 'Não definida', {locale: ptBR})}
                                    </p>
                                </div>
                                <div>
                                    <span className="text-blue-600 dark:text-blue-400">Horário:</span>
                                    <p className="font-medium">{form.time}</p>
                                </div>
                                <div>
                                    <span className="text-blue-600 dark:text-blue-400">Tempo:</span>
                                    <p className="font-medium">
                                        {totalMinutes} min ({(totalMinutes / 60).toFixed(2)}h)
                                    </p>
                                </div>
                                <div>
                                    <span className="text-blue-600 dark:text-blue-400">Contabilizável:</span>
                                    <p className="font-medium">{form.isBillable ? 'Sim' : 'Não'}</p>
                                </div>
                            </div>
                            <div className="mt-2">
                                <span className="text-blue-600 dark:text-blue-400">Descrição:</span>
                                <p className="font-medium">{form.description || 'Sem descrição'}</p>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </Modal>
    );
};

export default EditEntryModal;
