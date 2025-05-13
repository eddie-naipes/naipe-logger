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

    // Carrega os projetos
    useEffect(() => {
        const loadProjects = async () => {
            try {
                setIsLoadingProjects(true);
                const projectsList = await window.go.backend.App.GetProjects();
                setProjects(projectsList);

                // Seleciona o primeiro projeto automaticamente se a lista não estiver vazia
                if (projectsList && projectsList.length > 0) {
                    setSelectedProjectId(projectsList[0].id);
                    // Carrega as tarefas do primeiro projeto
                    loadTasksForProject(projectsList[0].id);
                } else {
                    // Se não há projetos, tenta carregar todas as tarefas
                    loadAllTasks();
                }
            } catch (error) {
                console.error('Erro ao carregar projetos:', error);
                toast.error('Erro ao carregar projetos. Tentando carregar tarefas diretamente...');
                // Tenta carregar todas as tarefas como fallback
                loadAllTasks();
            } finally {
                setIsLoadingProjects(false);
            }
        };

        loadProjects();

        // Carregar tarefas salvas independentemente dos projetos
        loadSavedTasks();
    }, []);

    // Carrega tarefas salvas
    const loadSavedTasks = async () => {
        try {
            const saved = await window.go.backend.App.GetSavedTasks();
            setSavedTasks(saved);
        } catch (error) {
            console.error('Erro ao carregar tarefas salvas:', error);
            toast.error('Erro ao carregar tarefas salvas.');
        }
    };

    // Carrega todas as tarefas (método antigo)
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

    // Atualiza as tarefas do projeto atual
    const refreshTasks = async () => {
        try {
            setIsRefreshing(true);
            if (selectedProjectId) {
                // Se há um projeto selecionado, atualiza suas tarefas
                await loadTasksForProject(selectedProjectId);
            } else {
                // Caso contrário, carrega todas as tarefas
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

    // Seleciona uma tarefa para adicionar
    const handleSelectTask = (task) => {
        // Verifica se a tarefa já está salva
        const isSaved = savedTasks.some(t => t.taskId === task.id);

        if (isSaved) {
            toast.info('Esta tarefa já está na sua lista de favoritas.');
            return;
        }

        setSelectedTask({
            taskId: task.id,  // Agora taskId é um número, não uma string
            taskName: task.content,
            projectId: task.projectId, // Também é um número
            projectName: task.projectName,
            entries: [
                {
                    minutes: 60,
                    userId: 0, // Será preenchido pelo backend
                    time: "09:00:00",
                    description: "Desenvolvimento",
                    isBillable: true
                }
            ]
        });
    };

    // Seleciona uma tarefa salva para edição
    const editSavedTask = (task) => {
        setSelectedTask({...task});
    };

    // Adiciona uma entrada à tarefa selecionada
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

    // Remove uma entrada da tarefa selecionada
    const removeEntry = (index) => {
        if (!selectedTask) return;

        const newEntries = [...selectedTask.entries];
        newEntries.splice(index, 1);

        setSelectedTask({
            ...selectedTask,
            entries: newEntries
        });
    };

    // Atualiza um campo de entrada
    const updateEntry = (index, field, value) => {
        if (!selectedTask) return;

        const newEntries = [...selectedTask.entries];

        // Ajusta o valor para o tipo correto
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

    // Salva a tarefa
    const saveTask = async () => {
        if (!selectedTask) return;

        // Verifica se há pelo menos uma entrada
        if (selectedTask.entries.length === 0) {
            toast.warning('Adicione pelo menos uma entrada de tempo.');
            return;
        }

        try {
            await window.go.backend.App.SaveTask(selectedTask);

            // Atualiza a lista de tarefas salvas
            await loadSavedTasks();

            // Limpa a seleção
            setSelectedTask(null);

            toast.success('Tarefa salva com sucesso!');
        } catch (error) {
            console.error('Erro ao salvar tarefa:', error);
            toast.error('Erro ao salvar tarefa.');
        }
    };

    // Remove uma tarefa salva
    const removeTask = async (taskId) => {
        try {
            // Passa o taskId diretamente como número
            await window.go.backend.App.RemoveTask(taskId);

            // Atualiza a lista de tarefas salvas
            await loadSavedTasks();

            toast.success('Tarefa removida com sucesso!');
        } catch (error) {
            console.error('Erro ao remover tarefa:', error);
            toast.error('Erro ao remover tarefa.');
        }
    };

    // Filtra as tarefas com base no termo de pesquisa
    const filteredTasks = tasks.filter(task =>
        task.content.toLowerCase().includes(searchTerm.toLowerCase()) ||
        (task.projectName && task.projectName.toLowerCase().includes(searchTerm.toLowerCase()))
    );

    // Calcula o total de minutos da tarefa selecionada
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
                {/* Lista de tarefas */}
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

                    {/* Seletor de Projetos */}
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

                    {/* Barra de pesquisa */}
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

                    {/* Lista de tarefas */}
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

                {/* Painel de edição */}
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