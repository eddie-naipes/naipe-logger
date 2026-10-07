import { useCallback, useState, useEffect, useRef } from 'react';
import { toast } from 'react-toastify';
import { FiList, FiRefreshCw, FiSearch, FiPlus, FiTrash2, FiSave, FiClock, FiEdit, FiFilter } from 'react-icons/fi';
import TimeInputComponent from '../components/TimeInputComponent';
import CommitSuggestButton from '../components/git/CommitSuggestButton';
import {
    GetProjects,
    GetSavedTasks,
    GetTasks,
    GetTasksByProject,
    RemoveTask,
    SaveTask
} from '@wailsjs/go/backend/App';
import {errMsg} from '../utils/errors';
import {
    DIAS_SEMANA,
    formatWorkingDays,
    hoursAndMinutesToMinutes,
    minutesToHoursAndMinutes,
    sumEntryMinutes
} from '../utils/time';
import {paraBinding, type Project, type Task, type TeamworkTask, type TimeEntry} from '../types/backend';

const Tasks = () => {
    const [projects, setProjects] = useState<Project[]>([]);
    // '' = todos os projetos.
    const [selectedProjectId, setSelectedProjectId] = useState<number | ''>('');
    const [isLoadingProjects, setIsLoadingProjects] = useState(false);
    const [tasks, setTasks] = useState<TeamworkTask[]>([]);
    const [savedTasks, setSavedTasks] = useState<Task[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [isRefreshing, setIsRefreshing] = useState(false);
    const [searchTerm, setSearchTerm] = useState('');
    const [selectedTask, setSelectedTask] = useState<Task | null>(null);

    // Cada carga de tarefas recebe um id; respostas de cargas antigas (ex.: o
    // usuário trocou de projeto antes da anterior terminar) são descartadas.
    const tasksRequestRef = useRef(0);
    const mountedRef = useRef(true);

    useEffect(() => {
        mountedRef.current = true;
        return () => {
            mountedRef.current = false;
        };
    }, []);

    // Os carregadores usam só refs e setters, então são estáveis (useCallback
    // sem dependências) e o efeito de carga inicial roda uma única vez.
    const loadSavedTasks = useCallback(async (): Promise<void> => {
        try {
            const saved: Task[] = (await GetSavedTasks()) ?? [];
            if (mountedRef.current) setSavedTasks(saved);
        } catch (error) {
            console.error('Erro ao carregar tarefas salvas:', error);
            toast.error('Erro ao carregar tarefas salvas: ' + errMsg(error));
        }
    }, []);

    // loadAllTasks/loadTasksForProject devolvem true em caso de sucesso, para
    // que "atualizar" não anuncie sucesso depois de uma falha.
    const loadAllTasks = useCallback(async (): Promise<boolean> => {
        const requestId = ++tasksRequestRef.current;
        const isCurrent = () => mountedRef.current && requestId === tasksRequestRef.current;

        try {
            setIsLoading(true);
            setTasks([]);

            const teamworkTasks: TeamworkTask[] = (await GetTasks()) ?? [];
            if (!isCurrent()) return false;
            setTasks(teamworkTasks);

            if (teamworkTasks.length === 0) {
                toast.info('Nenhuma tarefa encontrada.');
            }
            return true;
        } catch (error) {
            console.error('Erro ao carregar todas as tarefas:', error);
            if (!isCurrent()) return false;
            toast.error('Erro ao carregar tarefas: ' + errMsg(error));
            setTasks([]);
            return false;
        } finally {
            if (isCurrent()) setIsLoading(false);
        }
    }, []);

    const loadTasksForProject = useCallback(async (projectId: number): Promise<boolean> => {
        const requestId = ++tasksRequestRef.current;
        const isCurrent = () => mountedRef.current && requestId === tasksRequestRef.current;
        const id = Number(projectId);

        try {
            setIsLoading(true);
            setTasks([]);

            const teamworkTasks: TeamworkTask[] | null = await GetTasksByProject(id);
            if (!isCurrent()) return false;

            if (teamworkTasks && teamworkTasks.length > 0) {
                // projectId 0 = a API não informou o projeto da tarefa.
                const validTasks = teamworkTasks.filter(task =>
                    Number(task.projectId) === id || task.projectId === 0
                );

                setTasks(validTasks);

                if (validTasks.length === 0) {
                    toast.info('Nenhuma tarefa encontrada neste projeto.');
                }
            } else {
                setTasks([]);
                toast.info('Nenhuma tarefa encontrada neste projeto.');
            }
            return true;
        } catch (error) {
            console.error(`Erro ao carregar tarefas do projeto ${id}:`, error);
            if (!isCurrent()) return false;
            toast.error('Erro ao carregar tarefas do projeto: ' + errMsg(error));
            setTasks([]);
            return false;
        } finally {
            if (isCurrent()) setIsLoading(false);
        }
    }, []);

    useEffect(() => {
        const loadProjects = async () => {
            try {
                setIsLoadingProjects(true);
                const projectsList: Project[] = (await GetProjects()) ?? [];
                if (!mountedRef.current) return;
                setProjects(projectsList);

                const primeiro = projectsList[0];
                if (primeiro) {
                    const firstId = Number(primeiro.id);
                    setSelectedProjectId(firstId);
                    void loadTasksForProject(firstId);
                } else {
                    void loadAllTasks();
                }
            } catch (error) {
                console.error('Erro ao carregar projetos:', error);
                if (!mountedRef.current) return;
                toast.error('Erro ao carregar projetos (' + errMsg(error) + '). Tentando carregar tarefas diretamente...');
                void loadAllTasks();
            } finally {
                if (mountedRef.current) setIsLoadingProjects(false);
            }
        };

        void loadProjects();
        // Carga inicial de dados; o setState dentro do carregador é o resultado
        // da busca, não estado derivado de props.
        // eslint-disable-next-line react-hooks/set-state-in-effect
        void loadSavedTasks();
    }, [loadAllTasks, loadTasksForProject, loadSavedTasks]);

    const refreshTasks = async () => {
        setIsRefreshing(true);
        try {
            const ok = selectedProjectId
                ? await loadTasksForProject(selectedProjectId)
                : await loadAllTasks();
            if (ok) {
                toast.success('Tarefas atualizadas com sucesso!');
            }
        } finally {
            if (mountedRef.current) setIsRefreshing(false);
        }
    };

    const handleProjectChange = (projectId: string) => {
        setTasks([]);
        setSearchTerm('');
        setSelectedTask(null);

        // O <select> devolve string; o id é guardado como Number para casar com
        // task.projectId e com o tipo esperado pelo binding.
        if (projectId === '') {
            setSelectedProjectId('');
            void loadAllTasks();
            return;
        }

        const projectIdNumber = Number(projectId);
        if (!Number.isInteger(projectIdNumber) || projectIdNumber <= 0) {
            console.error('Project ID inválido:', projectId);
            toast.error('ID do projeto inválido');
            setSelectedProjectId('');
            return;
        }

        setSelectedProjectId(projectIdNumber);
        void loadTasksForProject(projectIdNumber);
    };

    const handleSelectTask = (task: TeamworkTask) => {
        const isSaved = savedTasks.some(t => t.taskId === task.id);

        if (isSaved) {
            toast.info('Esta tarefa já está na sua lista de favoritas.');
            return;
        }

        setSelectedTask({
            taskId: task.id,
            taskName: task.content,
            projectId: task.projectId,
            projectName: task.projectName,
            workingDays: [1, 2, 3, 4, 5],
            entries: [
                {
                    minutes: 60, // Será convertido para horas/minutos na UI
                    userId: 0,
                    time: "09:00:00",
                    description: "Desenvolvimento",
                    isBillable: true
                }
            ]
        });
    };

    const selectAllDays = () => {
        if (!selectedTask) return;

        const allDays = [0, 1, 2, 3, 4, 5, 6];
        const isAllSelected = allDays.every(day => selectedTask.workingDays?.includes(day));

        setSelectedTask({
            ...selectedTask,
            workingDays: isAllSelected ? [] : allDays
        });
    };

    const selectWeekdays = () => {
        if (!selectedTask) return;

        setSelectedTask({
            ...selectedTask,
            workingDays: [1, 2, 3, 4, 5]
        });
    };

    const editSavedTask = (task: Task) => {
        setSelectedTask({...task});
    };

    const toggleWorkingDay = (dayId: number) => {
        if (!selectedTask) return;

        const currentDays = selectedTask.workingDays || [];
        const newDays = currentDays.includes(dayId)
            ? currentDays.filter(day => day !== dayId)
            : [...currentDays, dayId].sort((a, b) => a - b);

        setSelectedTask({
            ...selectedTask,
            workingDays: newDays
        });
    };

    const addEntry = () => {
        if (!selectedTask) return;

        setSelectedTask({
            ...selectedTask,
            entries: [
                ...selectedTask.entries,
                {
                    minutes: 60,
                    userId: 0,
                    time: "10:00:00",
                    description: "Desenvolvimento",
                    isBillable: true
                }
            ]
        });
    };

    const removeEntry = (index: number) => {
        if (!selectedTask) return;

        const newEntries = [...selectedTask.entries];
        newEntries.splice(index, 1);

        setSelectedTask({
            ...selectedTask,
            entries: newEntries
        });
    };

    const updateEntry = <K extends keyof TimeEntry>(index: number, field: K, value: TimeEntry[K]) => {
        if (!selectedTask) return;

        const newEntries = [...selectedTask.entries];
        const atual = newEntries[index];
        if (!atual) return;

        newEntries[index] = {
            ...atual,
            [field]: value
        };

        setSelectedTask({
            ...selectedTask,
            entries: newEntries
        });
    };

    // Atualiza a duração a partir de horas e minutos.
    const updateEntryTime = (index: number, hours: number, minutes: number) => {
        const totalMinutes = hoursAndMinutesToMinutes(hours, minutes);
        updateEntry(index, 'minutes', totalMinutes);
    };

    const saveTask = async () => {
        if (!selectedTask) return;

        if (selectedTask.entries.length === 0) {
            toast.warning('Adicione pelo menos uma entrada de tempo.');
            return;
        }

        try {
            await SaveTask(paraBinding(selectedTask));
            await loadSavedTasks();
            setSelectedTask(null);
            toast.success('Tarefa salva com sucesso!');
        } catch (error) {
            console.error('Erro ao salvar tarefa:', error);
            toast.error('Erro ao salvar tarefa: ' + errMsg(error));
        }
    };

    const removeTask = async (taskId: number) => {
        try {
            await RemoveTask(taskId);
            await loadSavedTasks();
            toast.success('Tarefa removida com sucesso!');
        } catch (error) {
            console.error('Erro ao remover tarefa:', error);
            toast.error('Erro ao remover tarefa: ' + errMsg(error));
        }
    };

    const filteredTasks = tasks.filter(task =>
        task.content.toLowerCase().includes(searchTerm.toLowerCase()) ||
        (task.projectName && task.projectName.toLowerCase().includes(searchTerm.toLowerCase()))
    );

    const totalMinutes = selectedTask ? sumEntryMinutes(selectedTask.entries) : 0;

    return (
        <div>
            <div className="mb-6">
                <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">Tarefas</h1>
                <p className="text-gray-600 dark:text-gray-400">Gerencie suas tarefas do Teamwork</p>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="card">
                    <div className="flex items-center justify-between mb-4">
                        <h2 className="text-xl font-semibold text-gray-900 dark:text-white flex items-center">
                            <FiList className="w-5 h-5 mr-2" />
                            Tarefas do Teamwork
                        </h2>
                        <button
                            type="button"
                            onClick={() => void refreshTasks()}
                            disabled={isRefreshing}
                            aria-label="Atualizar tarefas"
                            title="Atualizar tarefas"
                            className="p-2 text-gray-500 hover:text-primary-600 dark:text-gray-400 dark:hover:text-primary-500"
                        >
                            <FiRefreshCw className={`w-5 h-5 ${isRefreshing ? 'animate-spin' : ''}`} />
                        </button>
                    </div>

                    <div className="mb-4">
                        <label htmlFor="projectSelect" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            <FiFilter className="inline w-4 h-4 mr-1" /> Selecione um Projeto
                        </label>
                        <select
                            id="projectSelect"
                            value={selectedProjectId}
                            onChange={(e) => handleProjectChange(e.target.value)}
                            className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white"
                            disabled={isLoadingProjects}
                        >
                            <option value="">Todos os Projetos</option>
                            {isLoadingProjects ? (
                                <option disabled>Carregando projetos...</option>
                            ) : projects.length === 0 ? (
                                <option disabled>Nenhum projeto encontrado</option>
                            ) : (
                                projects.map(project => (
                                    <option key={project.id} value={project.id}>
                                        {project.name}
                                    </option>
                                ))
                            )}
                        </select>
                    </div>

                    <div className="relative mb-4">
                        <div className="absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none">
                            <FiSearch className="w-5 h-5 text-gray-500 dark:text-gray-400" />
                        </div>
                        <input
                            type="text"
                            className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full pl-10 p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white"
                            placeholder="Pesquisar tarefas..."
                            aria-label="Pesquisar tarefas"
                            value={searchTerm}
                            onChange={(e) => setSearchTerm(e.target.value)}
                        />
                    </div>

                    {isLoading ? (
                        <div className="flex justify-center items-center h-60">
                            <div className="animate-spin-slow w-10 h-10 border-4 border-primary-600 border-t-transparent rounded-full"></div>
                            <span className="ml-3 text-gray-500 dark:text-gray-400">
            {selectedProjectId ? 'Carregando tarefas do projeto...' : 'Carregando tarefas...'}
        </span>
                        </div>
                    ) : (
                        <div className="overflow-y-auto max-h-96">
                            {filteredTasks.length === 0 ? (
                                <div className="text-center py-8">
                                    <p className="text-gray-500 dark:text-gray-400 mb-2">
                                        {searchTerm
                                            ? `Nenhuma tarefa encontrada para "${searchTerm}".`
                                            : selectedProjectId
                                                ? 'Não há tarefas disponíveis neste projeto.'
                                                : 'Não há tarefas disponíveis.'}
                                    </p>
                                    {selectedProjectId && !searchTerm && (
                                        <button
                                            type="button"
                                            onClick={() => void refreshTasks()}
                                            className="text-sm text-primary-600 hover:text-primary-700 dark:text-primary-500"
                                        >
                                            Tentar recarregar
                                        </button>
                                    )}
                                </div>
                            ) : (
                                <ul className="divide-y divide-gray-200 dark:divide-gray-700">
                                    {filteredTasks.map(task => (
                                        <li key={task.id} className="py-3">
                                            <div className="flex justify-between items-start">
                                                <div className="flex-1">
                                                    <h3 className="text-sm font-medium text-gray-900 dark:text-white">
                                                        {task.content || task.name || `Tarefa #${task.id}`}
                                                    </h3>
                                                    {task.projectName && (
                                                        <p className="text-xs text-gray-500 dark:text-gray-400">
                                                            {task.projectName}
                                                        </p>
                                                    )}
                                                    {import.meta.env.DEV && (
                                                        <p className="text-xs text-gray-400">
                                                            ID: {task.id} | ProjectID: {task.projectId || 'N/A'}
                                                        </p>
                                                    )}
                                                </div>
                                                <button
                                                    type="button"
                                                    onClick={() => handleSelectTask(task)}
                                                    aria-label={`Adicionar tarefa ${task.content || task.name || task.id} às favoritas`}
                                                    className="ml-2 p-1 text-primary-600 hover:bg-primary-50 rounded-sm dark:text-primary-500 dark:hover:bg-gray-700"
                                                    title="Adicionar tarefa à lista de favoritas"
                                                >
                                                    <FiPlus className="w-5 h-5" />
                                                </button>
                                            </div>
                                        </li>
                                    ))}
                                </ul>
                            )}
                        </div>
                    )}
                </div>

                <div className="card">
                    {selectedTask ? (
                        <div>
                            <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">
                                Configurar Tarefa
                            </h2>

                            <div className="mb-4">
                                <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                                    {selectedTask.taskName}
                                </h3>
                                <p className="text-sm text-gray-500 dark:text-gray-400">
                                    {selectedTask.projectName}
                                </p>
                            </div>

                            <div className="mb-6">
                                <div className="flex justify-between items-center mb-3">
                                    <h4 className="text-md font-medium text-gray-700 dark:text-gray-300">
                                        Dias da Semana para Lançamento
                                    </h4>
                                    <div className="flex space-x-2">
                                        <button
                                            type="button"
                                            onClick={selectWeekdays}
                                            className="text-xs text-primary-600 dark:text-primary-500 hover:underline"
                                        >
                                            Dias Úteis
                                        </button>
                                        <button
                                            type="button"
                                            onClick={selectAllDays}
                                            className="text-xs text-primary-600 dark:text-primary-500 hover:underline"
                                        >
                                            {DIAS_SEMANA.every(day => selectedTask.workingDays?.includes(day.id)) ? 'Desmarcar Todos' : 'Todos os Dias'}
                                        </button>
                                    </div>
                                </div>

                                <div className="grid grid-cols-7 gap-2">
                                    {DIAS_SEMANA.map(dia => (
                                        <label
                                            key={dia.id}
                                            className={`flex flex-col items-center p-2 rounded-lg border-2 cursor-pointer transition-colors ${
                                                selectedTask.workingDays?.includes(dia.id)
                                                    ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20 dark:border-primary-700'
                                                    : 'border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600'
                                            }`}
                                        >
                                            <input
                                                type="checkbox"
                                                checked={selectedTask.workingDays?.includes(dia.id) || false}
                                                onChange={() => toggleWorkingDay(dia.id)}
                                                className="sr-only"
                                            />
                                            <span className="text-xs font-medium">{dia.abrev}</span>
                                            <span className="text-xs text-gray-500 dark:text-gray-400">{dia.nome}</span>
                                        </label>
                                    ))}
                                </div>

                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                                    {selectedTask.workingDays?.length === 0
                                        ? 'Nenhum dia selecionado - tarefa não será lançada'
                                        : `Selecionados: ${selectedTask.workingDays?.length || 0} dia(s)`
                                    }
                                </p>
                            </div>

                            <div className="mb-4">
                                <div className="flex justify-between items-center mb-2">
                                    <h4 className="text-md font-medium text-gray-700 dark:text-gray-300">
                                        Entradas de Tempo
                                    </h4>
                                    <button
                                        type="button"
                                        onClick={addEntry}
                                        className="text-xs flex items-center text-primary-600 dark:text-primary-500"
                                    >
                                        <FiPlus className="w-4 h-4 mr-1" />
                                        Adicionar Entrada
                                    </button>
                                </div>

                                {selectedTask.entries.map((entry, index) => {
                                    const { hours, minutes } = minutesToHoursAndMinutes(entry.minutes);

                                    return (
                                        <div key={index} className="bg-gray-50 dark:bg-gray-700 p-3 rounded-lg mb-3">
                                            <div className="flex justify-between items-center mb-2">
                                                <h5 className="text-sm font-medium flex items-center">
                                                    <FiClock className="w-4 h-4 mr-1 text-primary-600 dark:text-primary-500" />
                                                    Entrada {index + 1}
                                                </h5>
                                                <button
                                                    type="button"
                                                    onClick={() => removeEntry(index)}
                                                    aria-label={`Remover entrada ${index + 1}`}
                                                    title="Remover entrada"
                                                    className="text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
                                                >
                                                    <FiTrash2 className="w-4 h-4" />
                                                </button>
                                            </div>

                                            <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-3">
                                                <TimeInputComponent
                                                    hours={hours}
                                                    minutes={minutes}
                                                    onTimeChange={(h, m) => updateEntryTime(index, h, m)}
                                                    label="Duração"
                                                    showTotalMinutes={false}
                                                />

                                                <div>
                                                    <label htmlFor={`entry-${index}-time`} className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                                                        Hora
                                                    </label>
                                                    <input
                                                        id={`entry-${index}-time`}
                                                        type="time"
                                                        value={entry.time ? entry.time.substring(0, 5) : "09:00"}
                                                        onChange={(e) => updateEntry(index, 'time', e.target.value + ":00")}
                                                        className="bg-white border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2 dark:bg-gray-600 dark:border-gray-500 dark:text-white"
                                                    />
                                                </div>
                                            </div>

                                            <div className="mb-3">
                                                <div className="flex items-center justify-between">
                                                    <label htmlFor={`entry-${index}-description`} className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                                                        Descrição
                                                    </label>
                                                    <CommitSuggestButton
                                                        onSuggest={(texto) => updateEntry(index, 'description', texto)}
                                                    />
                                                </div>
                                                <input
                                                    id={`entry-${index}-description`}
                                                    type="text"
                                                    value={entry.description}
                                                    onChange={(e) => updateEntry(index, 'description', e.target.value)}
                                                    className="bg-white border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2 dark:bg-gray-600 dark:border-gray-500 dark:text-white"
                                                />
                                            </div>

                                            <div>
                                                <label htmlFor={`entry-${index}-billable`} className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                                                    Cobrável
                                                </label>
                                                <select
                                                    id={`entry-${index}-billable`}
                                                    value={entry.isBillable.toString()}
                                                    onChange={(e) => updateEntry(index, 'isBillable', e.target.value === 'true')}
                                                    className="bg-white border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2 dark:bg-gray-600 dark:border-gray-500 dark:text-white"
                                                >
                                                    <option value="true">Sim</option>
                                                    <option value="false">Não</option>
                                                </select>
                                            </div>
                                        </div>
                                    );
                                })}

                                <div className="text-right mt-2 text-sm">
                                    <span className="font-medium">Total:</span> {totalMinutes} minutos ({(totalMinutes / 60).toFixed(1)} horas)
                                </div>
                            </div>

                            <div className="flex space-x-3">
                                <button
                                    type="button"
                                    onClick={() => setSelectedTask(null)}
                                    className="btn-secondary"
                                >
                                    Cancelar
                                </button>
                                <button
                                    type="button"
                                    onClick={() => void saveTask()}
                                    className="btn-primary flex items-center"
                                >
                                    <FiSave className="w-4 h-4 mr-2" />
                                    Salvar Tarefa
                                </button>
                            </div>
                        </div>
                    ) : (
                        <div>
                            <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">
                                Tarefas Salvas
                            </h2>

                            {savedTasks.length === 0 ? (
                                <div className="text-center py-8">
                                    <p className="text-gray-500 dark:text-gray-400 mb-4">
                                        Você ainda não salvou nenhuma tarefa.
                                    </p>
                                    <p className="text-sm text-gray-500 dark:text-gray-400">
                                        Selecione uma tarefa da lista à esquerda para configurar entradas de tempo.
                                    </p>
                                </div>
                            ) : (
                                <div className="overflow-y-auto max-h-96">
                                    <ul className="divide-y divide-gray-200 dark:divide-gray-700">
                                        {savedTasks.map(task => (
                                            <li key={task.taskId} className="py-3">
                                                <div className="flex justify-between items-start">
                                                    <div className="flex-1">
                                                        <h3 className="text-sm font-medium text-gray-900 dark:text-white">
                                                            {task.taskName}
                                                        </h3>
                                                        <p className="text-xs text-gray-500 dark:text-gray-400">
                                                            {task.projectName} • {task.entries.length} entradas •
                                                            {sumEntryMinutes(task.entries)} min
                                                            {task.workingDays && (
                                                                <span className="block text-blue-600 dark:text-blue-400 mt-1">
                                                                   <FiClock className="inline w-3 h-3 mr-1" />
                                                                    {formatWorkingDays(task.workingDays)}
                                                               </span>
                                                            )}
                                                        </p>
                                                    </div>
                                                    <div className="flex space-x-1">
                                                        <button
                                                            type="button"
                                                            onClick={() => editSavedTask(task)}
                                                            aria-label={`Editar tarefa ${task.taskName}`}
                                                            title="Editar tarefa"
                                                            className="p-1 text-primary-600 hover:bg-primary-50 rounded-sm dark:text-primary-500 dark:hover:bg-gray-700"
                                                        >
                                                            <FiEdit className="w-5 h-5" />
                                                        </button>
                                                        <button
                                                            type="button"
                                                            onClick={() => void removeTask(task.taskId)}
                                                            aria-label={`Remover tarefa ${task.taskName}`}
                                                            title="Remover tarefa"
                                                            className="p-1 text-red-500 hover:bg-red-50 rounded-sm dark:text-red-400 dark:hover:bg-gray-700"
                                                        >
                                                            <FiTrash2 className="w-5 h-5" />
                                                        </button>
                                                    </div>
                                                </div>
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default Tasks;