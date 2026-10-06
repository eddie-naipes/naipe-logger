import clsx from 'clsx';
import {FiPlusCircle} from 'react-icons/fi';
import type {Task} from '../../types/backend';
import {type AgendaItem, type AgendaTaskRef, itemSelecionavel, tarefaDoItem} from '../../utils/agenda';
import TaskSelect from './TaskSelect';

interface StatusVisual {label: string; className: string}

const IGNORADO: StatusVisual = {label: 'Ignorado', className: 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300'};

const STATUS: Partial<Record<string, StatusVisual>> = {
    mapped: {label: 'Mapeado', className: 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-200'},
    unmapped: {label: 'Sem regra', className: 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200'},
    ignored: IGNORADO,
    imported: {label: 'Já lançado', className: 'bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-200'},
};

interface EventsTableProps {
    items: readonly AgendaItem[];
    selected: ReadonlySet<string>;
    chosenTasks: ReadonlyMap<string, AgendaTaskRef>;
    savedTasks: readonly Task[];
    formatDate: (date: string) => string;
    onToggle: (key: string) => void;
    onChooseTask: (key: string, task: AgendaTaskRef | null) => void;
    onCreateRule: (item: AgendaItem) => void;
}

const selectClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-xs rounded-lg p-1.5 w-full dark:bg-gray-700 dark:border-gray-600 dark:text-white';

// Eventos do período com o status de cada um. Eventos sem regra ganham um
// seletor de tarefa e o atalho para criar uma regra a partir do título.
const EventsTable = ({items, selected, chosenTasks, savedTasks, formatDate, onToggle, onChooseTask, onCreateRule}: EventsTableProps) => {
    if (items.length === 0) {
        return <p className="text-sm text-gray-500 dark:text-gray-400">Nenhum evento no período.</p>;
    }

    return (
        <div className="overflow-x-auto">
            <table className="min-w-full text-sm">
                <thead>
                <tr className="text-left text-xs uppercase text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-700">
                    <th className="py-2 pr-2"><span className="sr-only">Selecionar</span></th>
                    <th className="py-2 pr-2">Data</th>
                    <th className="py-2 pr-2">Horário</th>
                    <th className="py-2 pr-2">Evento</th>
                    <th className="py-2 pr-2">Situação</th>
                    <th className="py-2">Tarefa</th>
                </tr>
                </thead>
                <tbody>
                {items.map(item => {
                    const status = STATUS[item.status] ?? IGNORADO;
                    const tarefa = tarefaDoItem(item, chosenTasks);
                    const selecionavel = itemSelecionavel(item) && tarefa !== null && item.minutes > 0;
                    return (
                        <tr key={item.key} className={clsx('border-b border-gray-100 dark:border-gray-800 align-top',
                            !itemSelecionavel(item) && 'opacity-60')}>
                            <td className="py-2 pr-2">
                                <input
                                    type="checkbox"
                                    aria-label={`Incluir ${item.title}`}
                                    checked={selecionavel && selected.has(item.key)}
                                    disabled={!selecionavel}
                                    onChange={() => onToggle(item.key)}
                                />
                            </td>
                            <td className="py-2 pr-2 whitespace-nowrap text-gray-700 dark:text-gray-300">{formatDate(item.date)}</td>
                            <td className="py-2 pr-2 whitespace-nowrap text-gray-700 dark:text-gray-300">
                                {item.startTime}–{item.endTime}
                                {item.minutes > 0 && <span className="block text-xs text-gray-500">{item.minutes} min</span>}
                            </td>
                            <td className="py-2 pr-2 text-gray-900 dark:text-white">
                                {item.title || '(sem título)'}
                                <span className="block text-xs text-gray-500 dark:text-gray-400">{item.source}</span>
                            </td>
                            <td className="py-2 pr-2">
                                <span className={clsx('inline-block px-2 py-0.5 rounded-full text-xs font-medium', status.className)}>
                                    {status.label}
                                </span>
                                {item.reason && <span className="block text-xs text-gray-500 dark:text-gray-400 mt-1">{item.reason}</span>}
                            </td>
                            <td className="py-2 min-w-48">
                                {item.status === 'unmapped' ? (
                                    <div className="space-y-1">
                                        <TaskSelect
                                            id={`agenda-evento-${item.key}`}
                                            ariaLabel={`Tarefa para ${item.title}`}
                                            value={chosenTasks.get(item.key) ?? null}
                                            savedTasks={savedTasks}
                                            onChange={task => onChooseTask(item.key, task)}
                                            emptyLabel="Escolha a tarefa"
                                            className={selectClass}
                                        />
                                        <button type="button" onClick={() => onCreateRule(item)}
                                                className="flex items-center text-xs text-primary-600 hover:underline dark:text-primary-400">
                                            <FiPlusCircle className="w-3 h-3 mr-1" aria-hidden="true"/>
                                            Criar regra a partir deste evento
                                        </button>
                                    </div>
                                ) : tarefa ? (
                                    <span className="text-gray-700 dark:text-gray-300">
                                        {tarefa.taskName || `Tarefa ${tarefa.taskId}`}
                                        {tarefa.projectName && <span className="block text-xs text-gray-500">{tarefa.projectName}</span>}
                                    </span>
                                ) : null}
                            </td>
                        </tr>
                    );
                })}
                </tbody>
            </table>
        </div>
    );
};

export default EventsTable;
