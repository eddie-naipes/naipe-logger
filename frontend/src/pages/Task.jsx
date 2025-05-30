import React, { useState, useEffect } from 'react';
import { toast } from 'react-toastify';
import { FiList, FiRefreshCw, FiSearch, FiPlus, FiTrash2, FiSave, FiClock, FiEdit, FiFilter } from 'react-icons/fi';

const Tasks = () => {
    const [projects, setProjects] = useState([]);
    const [selectedProjectId, setSelectedProjectId] = useState('');
    const [isLoadingProjects, setIsLoadingProjects] = useState(false);
    const [tasks, setTasks] = useState([]);
    const [savedTasks, setSavedTasks] = useState([]);
    const [isLoading, setIsLoading] = useState(true);
    const [isRefreshing, setIsRefreshing] = useState(false);
    const [searchTerm, setSearchTerm] = useState('');
    const [selectedTask, setSelectedTask] = useState(null);

    const diasSemana = [
        { id: 1, nome: 'Segunda', abrev: 'Seg' },
        { id: 2, nome: 'Terça', abrev: 'Ter' },
        { id: 3, nome: 'Quarta', abrev: 'Qua' },
        { id: 4, nome: 'Quinta', abrev: 'Qui' },
        { id: 5, nome: 'Sexta', abrev: 'Sex' },
        { id: 6, nome: 'Sábado', abrev: 'Sáb' },
        { id: 0, nome: 'Domingo', abrev: 'Dom' }
    ];

    useEffect(() => {
        const loadProjects = async () => {
            try {
                setIsLoadingProjects(true);
                const projectsList = await window.go.backend.App.GetProjects();
                setProjects(projectsList);

                if (projectsList && projectsList.length > 0) {
                    setSelectedProjectId(projectsList[0].id);
                    loadTasksForProject(projectsList[0].id);
                } else {
                    loadAllTasks();
                }
            } catch (error) {
                console.error('Erro ao carregar projetos:', error);
                toast.error('Erro ao carregar projetos. Tentando carregar tarefas diretamente...');
                loadAllTasks();
            } finally {
                setIsLoadingProjects(false);
            }
        };

        loadProjects();
        loadSavedTasks();
    }, []);

    const loadSavedTasks = async () => {
        try {
            const saved = await window.go.backend.App.GetSavedTasks();
            setSavedTasks(saved);
        } catch (error) {
            console.error('Erro ao carregar tarefas salvas:', error);
            toast.error('Erro ao carregar tarefas salvas.');
        }
    };

    const loadAllTasks = async () => {
        try {
            setIsLoading(true);
            const teamworkTasks = await window.go.backend.App.GetTasks();
            setTasks(teamworkTasks);
        } catch (error) {
            console.error('Erro ao carregar tarefas:', error);
            toast.error('Erro ao carregar tarefas.');
        } finally {
            setIsLoading(false);
        }
    };

    const loadTasksForProject = async (projectId) => {
        try {
            setIsLoading(true);
            const teamworkTasks = await window.go.backend.App.GetTasksByProject(projectId);
            setTasks(teamworkTasks);
        } catch (error) {
            console.error(`Erro ao carregar tarefas do projeto ${projectId}:`, error);
            toast.error('Erro ao carregar tarefas do projeto.');
        } finally {
            setIsLoading(false);
        }
    };

    const refreshTasks = async () => {
        try {
            setIsRefreshing(true);
            if (selectedProjectId) {
                await loadTasksForProject(selectedProjectId);
            } else {
                await loadAllTasks();
            }
            toast.success('Tarefas atualizadas com sucesso!');
        } catch (error) {
            console.error('Erro ao atualizar tarefas:', error);
            toast.error('Erro ao atualizar tarefas.');
        } finally {
            setIsRefreshing(false);
        }
    };

    const handleProjectChange = (projectId) => {
        setSelectedProjectId(projectId);
        if (projectId) {
            loadTasksForProject(projectId);
        } else {
            loadAllTasks();
        }
    };

    const handleSelectTask = (task) => {
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
                    minutes: 60,
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

    const editSavedTask = (task) => {
        setSelectedTask({...task});
    };

    const toggleWorkingDay = (dayId) => {
        if (!selectedTask) return;

        const currentDays = selectedTask.workingDays || [];
        const newDays = currentDays.includes(dayId)
            ? currentDays.filter(day => day !== dayId)
            : [...currentDays, dayId].sort();

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

    const removeEntry = (index) => {
        if (!selectedTask) return;

        const newEntries = [...selectedTask.entries];
        newEntries.splice(index, 1);

        setSelectedTask({
            ...selectedTask,
            entries: newEntries
        });
    };

    const updateEntry = (index, field, value) => {
        if (!selectedTask) return;

        const newEntries = [...selectedTask.entries];

        let parsedValue = value;
        if (field === 'minutes') {
            parsedValue = parseInt(value) || 0;
        } else if (field === 'isBillable') {
            parsedValue = value === 'true';
        }

        newEntries[index] = {
            ...newEntries[index],
            [field]: parsedValue
        };

        setSelectedTask({
            ...selectedTask,
            entries: newEntries
        });
    };

    const saveTask = async () => {
        if (!selectedTask) return;

        if (selectedTask.entries.length === 0) {
            toast.warning('Adicione pelo menos uma entrada de tempo.');
            return;
        }

        try {
            await window.go.backend.App.SaveTask(selectedTask);
            await loadSavedTasks();
            setSelectedTask(null);
            toast.success('Tarefa salva com sucesso!');
        } catch (error) {
            console.error('Erro ao salvar tarefa:', error);
            toast.error('Erro ao salvar tarefa.');
        }
    };

    const removeTask = async (taskId) => {
        try {
            await window.go.backend.App.RemoveTask(taskId);
            await loadSavedTasks();
            toast.success('Tarefa removida com sucesso!');
        } catch (error) {
            console.error('Erro ao remover tarefa:', error);
            toast.error('Erro ao remover tarefa.');
        }
    };

    const formatWorkingDays = (workingDays) => {
        if (!workingDays || workingDays.length === 0) return 'Todos os dias';

        const diasNomes = {
            0: 'Dom', 1: 'Seg', 2: 'Ter', 3: 'Qua',
            4: 'Qui', 5: 'Sex', 6: 'Sáb'
        };

        if (workingDays.length === 7) return 'Todos os dias';
        if (workingDays.length === 5 &&
            [1,2,3,4,5].every(day => workingDays.includes(day))) {
            return 'Dias úteis';
        }

        return workingDays
            .sort()
            .map(day => diasNomes[day])
            .join(', ');
    };

    const filteredTasks = tasks.filter(task =>
        task.content.toLowerCase().includes(searchTerm.toLowerCase()) ||
        (task.projectName && task.projectName.toLowerCase().includes(searchTerm.toLowerCase()))
    );

    const totalMinutes = selectedTask
        ? selectedTask.entries.reduce((sum, entry) => sum + entry.minutes, 0)
        : 0;

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
                            onClick={refreshTasks}
                            disabled={isRefreshing}
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
                            value={searchTerm}
                            onChange={(e) => setSearchTerm(e.target.value)}
                        />
                    </div>

                    {isLoading ? (
                        <div className="flex justify-center items-center h-60">
                            <div className="animate-spin-slow w-10 h-10 border-4 border-primary-600 border-t-transparent rounded-full"></div>
                        </div>
                    ) : (
                        <div className="overflow-y-auto max-h-96">
                            {filteredTasks.length === 0 ? (
                                <p className="text-gray-500 dark:text-gray-400 text-center py-8">
                                    {searchTerm
                                        ? 'Nenhuma tarefa encontrada com esse termo.'
                                        : selectedProjectId
                                            ? 'Não há tarefas disponíveis neste projeto.'
                                            : 'Não há tarefas disponíveis.'}
                                </p>
                            ) : (
                                <ul className="divide-y divide-gray-200 dark:divide-gray-700">
                                    {filteredTasks.map(task => (
                                        <li key={task.id} className="py-3">
                                            <div className="flex justify-between items-start">
                                                <div className="flex-1">
                                                    <h3 className="text-sm font-medium text-gray-900 dark:text-white">{task.content}</h3>
                                                    {task.projectName && (
                                                        <p className="text-xs text-gray-500 dark:text-gray-400">{task.projectName}</p>
                                                    )}
                                                </div>
                                                <button
                                                    onClick={() => handleSelectTask(task)}
                                                    className="ml-2 p-1 text-primary-600 hover:bg-primary-50 rounded dark:text-primary-500 dark:hover:bg-gray-700"
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
                                            {diasSemana.every(day => selectedTask.workingDays?.includes(day.id)) ? 'Desmarcar Todos' : 'Todos os Dias'}
                                        </button>
                                    </div>
                                </div>

                                <div className="grid grid-cols-7 gap-2">
                                    {diasSemana.map(dia => (
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
                                        onClick={addEntry}
                                        className="text-xs flex items-center text-primary-600 dark:text-primary-500"
                                    >
                                        <FiPlus className="w-4 h-4 mr-1" />
                                        Adicionar Entrada
                                    </button>
                                </div>

                                {selectedTask.entries.map((entry, index) => (
                                    <div key={index} className="bg-gray-50 dark:bg-gray-700 p-3 rounded-lg mb-3">
                                        <div className="flex justify-between items-center mb-2">
                                            <h5 className="text-sm font-medium flex items-center">
                                                <FiClock className="w-4 h-4 mr-1 text-primary-600 dark:text-primary-500" />
                                                Entrada {index + 1}
                                            </h5>
                                            <button
                                                onClick={() => removeEntry(index)}
                                                className="text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
                                            >
                                                <FiTrash2 className="w-4 h-4" />
                                            </button>
                                        </div>

                                        <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-3">
                                            <div>
                                                <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                                                    Tempo (minutos)
                                                </label>
                                                <input
                                                    type="number"
                                                    value={entry.minutes}
                                                    onChange={(e) => updateEntry(index, 'minutes', e.target.value)}
                                                    className="bg-white border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2 dark:bg-gray-600 dark:border-gray-500 dark:text-white"
                                                />
                                            </div>
                                            <div>
                                                <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                                                    Hora
                                                </label>
                                                <input
                                                    type="time"
                                                    value={entry.time ? entry.time.substring(0, 5) : "09:00"}
                                                    onChange={(e) => updateEntry(index, 'time', e.target.value + ":00")}
                                                    className="bg-white border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2 dark:bg-gray-600 dark:border-gray-500 dark:text-white"
                                                />
                                            </div>
                                        </div>

                                        <div className="mb-3">
                                            <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                                                Descrição
                                            </label>
                                            <input
                                                type="text"
                                                value={entry.description}
                                                onChange={(e) => updateEntry(index, 'description', e.target.value)}
                                                className="bg-white border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2 dark:bg-gray-600 dark:border-gray-500 dark:text-white"
                                            />
                                        </div>

                                        <div>
                                            <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                                                Cobrável
                                            </label>
                                            <select
                                                value={entry.isBillable.toString()}
                                                onChange={(e) => updateEntry(index, 'isBillable', e.target.value)}
                                                className="bg-white border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2 dark:bg-gray-600 dark:border-gray-500 dark:text-white"
                                            >
                                                <option value="true">Sim</option>
                                                <option value="false">Não</option>
                                            </select>
                                        </div>
                                    </div>
                                ))}

                                <div className="text-right mt-2 text-sm">
                                    <span className="font-medium">Total:</span> {totalMinutes} minutos ({(totalMinutes / 60).toFixed(1)} horas)
                                </div>
                            </div>

                            <div className="flex space-x-3">
                                <button
                                    onClick={() => setSelectedTask(null)}
                                    className="btn-secondary"
                                >
                                    Cancelar
                                </button>
                                <button
                                    onClick={saveTask}
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
                                                            {task.entries.reduce((sum, e) => sum + e.minutes, 0)} min
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
                                                            onClick={() => editSavedTask(task)}
                                                            className="p-1 text-primary-600 hover:bg-primary-50 rounded dark:text-primary-500 dark:hover:bg-gray-700"
                                                        >
                                                            <FiEdit className="w-5 h-5" />
                                                        </button>
                                                        <button
                                                            onClick={() => removeTask(task.taskId)}
                                                            className="p-1 text-red-500 hover:bg-red-50 rounded dark:text-red-400 dark:hover:bg-gray-700"
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