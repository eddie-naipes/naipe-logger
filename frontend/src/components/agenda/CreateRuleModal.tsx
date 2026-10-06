import {useState} from 'react';
import {FiPlusCircle} from 'react-icons/fi';
import Modal from '../Modal';
import type {Task} from '../../types/backend';
import type {AgendaItem, AgendaTaskRef} from '../../utils/agenda';
import TaskSelect from './TaskSelect';
import type {AgendaRule} from './RulesCard';

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white';
const labelClass = 'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';

interface CreateRuleModalProps {
    item: AgendaItem;
    savedTasks: readonly Task[];
    initialTask: AgendaTaskRef | null;
    saving: boolean;
    onClose: () => void;
    onCreate: (rule: AgendaRule) => void;
}

// Cria uma regra com o título do evento como palavra-chave (editável).
const CreateRuleModal = ({item, savedTasks, initialTask, saving, onClose, onCreate}: CreateRuleModalProps) => {
    const [match, setMatch] = useState(item.title);
    const [task, setTask] = useState<AgendaTaskRef | null>(initialTask);
    const [description, setDescription] = useState('');

    const podeCriar = match.trim() !== '' && task !== null && task.taskId > 0;

    return (
        <Modal
            isOpen
            onClose={onClose}
            closeDisabled={saving}
            title="Criar regra"
            icon={<FiPlusCircle className="w-5 h-5 mr-2" aria-hidden="true"/>}
            footer={
                <>
                    <button type="button" onClick={onClose} disabled={saving} className="btn-secondary disabled:opacity-50">
                        Cancelar
                    </button>
                    <button
                        type="button"
                        disabled={!podeCriar || saving}
                        onClick={() => task && onCreate({match: match.trim(), isRegex: false, task, description: description.trim()})}
                        className="btn-primary disabled:opacity-50"
                    >
                        {saving ? 'Salvando...' : 'Criar regra'}
                    </button>
                </>
            }
        >
            <div className="space-y-3">
                <div>
                    <label htmlFor="nova-regra-palavra" className={labelClass}>Palavra-chave no título</label>
                    <input id="nova-regra-palavra" value={match} onChange={e => setMatch(e.target.value)} className={inputClass}/>
                    <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                        Encurte para pegar outras reuniões parecidas (ex.: “Daily” em vez de “Daily do time X”).
                    </p>
                </div>
                <div>
                    <label htmlFor="nova-regra-tarefa" className={labelClass}>Tarefa</label>
                    <TaskSelect id="nova-regra-tarefa" value={task} savedTasks={savedTasks} onChange={setTask}
                                emptyLabel="Escolha a tarefa" className={inputClass}/>
                </div>
                <div>
                    <label htmlFor="nova-regra-descricao" className={labelClass}>Descrição (opcional)</label>
                    <input id="nova-regra-descricao" value={description} onChange={e => setDescription(e.target.value)}
                           placeholder="Padrão: título do evento" className={inputClass}/>
                </div>
            </div>
        </Modal>
    );
};

export default CreateRuleModal;
