import {FiAlertCircle, FiClock, FiList} from 'react-icons/fi';
import type {Task} from '../../types/backend';
import {formatWorkingDays, sumEntryMinutes} from '../../utils/time';

interface TaskSelectorProps {
    savedTasks: readonly Task[];
    selectedTasks: readonly number[];
    onToggle: (taskId: number) => void;
    onToggleAll: () => void;
}

// Lista das tarefas salvas com checkbox para escolher o que entra no plano.
const TaskSelector = ({savedTasks, selectedTasks, onToggle, onToggleAll}: TaskSelectorProps) => (
    <div className="card lg:col-span-2">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
            <FiList className="w-5 h-5 mr-2" aria-hidden="true"/>
            Tarefas para Lançamento
        </h2>

        {savedTasks.length === 0 ? (
            <div className="bg-yellow-50 border-l-4 border-yellow-400 p-4 dark:bg-yellow-900/20 dark:border-yellow-600">
                <div className="flex items-start">
                    <FiAlertCircle className="mt-0.5 w-5 h-5 text-yellow-500 dark:text-yellow-600 mr-2" aria-hidden="true"/>
                    <div>
                        <h3 className="text-sm font-medium text-yellow-800 dark:text-yellow-400">
                            Nenhuma tarefa salva
                        </h3>
                        <p className="mt-1 text-sm text-yellow-700 dark:text-yellow-200">
                            Adicione tarefas na seção &quot;Tarefas&quot; ou aplique um template da seção &quot;Templates&quot;.
                        </p>
                    </div>
                </div>
            </div>
        ) : (
            <>
                <div className="mb-4">
                    <button
                        type="button"
                        onClick={onToggleAll}
                        className="text-sm text-primary-600 hover:text-primary-700 dark:text-primary-500 dark:hover:text-primary-400"
                    >
                        {selectedTasks.length === savedTasks.length ? 'Desmarcar todas' : 'Selecionar todas'}
                    </button>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {savedTasks.map(task => {
                        const selected = selectedTasks.includes(task.taskId);
                        const inputId = `timelog-task-${task.taskId}`;
                        return (
                            <div
                                key={task.taskId}
                                className={`border rounded-lg p-3 ${
                                    selected
                                        ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20 dark:border-primary-700'
                                        : 'border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600'
                                }`}
                            >
                                <div className="flex items-start">
                                    <div className="shrink-0">
                                        <input
                                            id={inputId}
                                            type="checkbox"
                                            checked={selected}
                                            onChange={() => onToggle(task.taskId)}
                                            aria-label={`Incluir tarefa ${task.taskName} no plano`}
                                            className="w-4 h-4 text-primary-600 bg-gray-100 border-gray-300 rounded-sm focus:ring-primary-500 dark:focus:ring-primary-600 dark:ring-offset-gray-800 dark:focus:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                                        />
                                    </div>
                                    <label htmlFor={inputId} className="ml-3 cursor-pointer">
                                        <span className="block text-sm font-medium text-gray-900 dark:text-white">
                                            {task.taskName}
                                        </span>
                                        <span className="block text-xs text-gray-500 dark:text-gray-400">
                                            {task.projectName}
                                        </span>
                                        <span className="block mt-1 text-xs">
                                            {task.entries.length} entradas • {sumEntryMinutes(task.entries)} min
                                        </span>
                                        {task.workingDays && (
                                            <span className="block mt-1 text-xs text-blue-600 dark:text-blue-400">
                                                <FiClock className="inline w-3 h-3 mr-1" aria-hidden="true"/>
                                                {formatWorkingDays(task.workingDays)}
                                            </span>
                                        )}
                                    </label>
                                </div>
                            </div>
                        );
                    })}
                </div>
            </>
        )}
    </div>
);

export default TaskSelector;
