import React, { useState, useEffect } from 'react';
import { toast } from 'react-toastify';
import { FiSave, FiEdit, FiTrash2, FiPlus, FiFolder, FiLoader, FiCopy } from 'react-icons/fi';

const Templates = () => {
    const [isLoading, setIsLoading] = useState(true);
    const [templates, setTemplates] = useState({});
    const [savedTasks, setSavedTasks] = useState([]);
    const [selectedTasks, setSelectedTasks] = useState([]);
    const [templateName, setTemplateName] = useState('');
    const [isEditing, setIsEditing] = useState(false);
    const [currentTemplate, setCurrentTemplate] = useState(null);
    const [isSaving, setIsSaving] = useState(false);

    // Carrega os templates e tarefas
    useEffect(() => {
        const loadData = async () => {
            try {
                setIsLoading(true);

                // Carregar templates
                const templatesData = await window.go.backend.App.GetTemplates();
                setTemplates(templatesData);

                // Carregar tarefas salvas
                const tasksData = await window.go.backend.App.GetSavedTasks();
                setSavedTasks(tasksData);
            } catch (error) {
                console.error('Erro ao carregar dados:', error);
                toast.error('Erro ao carregar templates e tarefas.');
            } finally {
                setIsLoading(false);
            }
        };

        loadData();
    }, []);

    // Limpa o formulário
    const resetForm = () => {
        setTemplateName('');
        setSelectedTasks([]);
        setIsEditing(false);
        setCurrentTemplate(null);
    };

    // Toggle seleção de tarefa
    const toggleTaskSelection = (taskId) => {
        if (selectedTasks.includes(taskId)) {
            setSelectedTasks(selectedTasks.filter(id => id !== taskId));
        } else {
            setSelectedTasks([...selectedTasks, taskId]);
        }
    };

    // Seleciona todas as tarefas
    const selectAllTasks = () => {
        if (selectedTasks.length === savedTasks.length) {
            setSelectedTasks([]);
        } else {
            setSelectedTasks(savedTasks.map(task => task.taskId));
        }
    };

    // Edita um template existente
    const editTemplate = (name) => {
        const template = templates[name];
        if (!template) return;

        setCurrentTemplate(name);
        setTemplateName(name);
        setIsEditing(true);

        // Seleciona as tarefas do template
        const taskIds = template.tasks.map(task => task.taskId);
        setSelectedTasks(taskIds);
    };

    // Remove um template
    const deleteTemplate = async (name) => {
        if (!window.confirm(`Tem certeza que deseja excluir o template "${name}"?`)) {
            return;
        }

        try {
            await window.go.backend.App.DeleteTemplate(name);

            // Atualiza a lista de templates
            const updatedTemplates = { ...templates };
            delete updatedTemplates[name];
            setTemplates(updatedTemplates);

            toast.success('Template excluído com sucesso!');
        } catch (error) {
            console.error('Erro ao excluir template:', error);
            toast.error('Erro ao excluir template.');
        }
    };

    // Salva um template
    const saveTemplate = async (e) => {
        e.preventDefault();

        if (!templateName.trim()) {
            toast.warning('Informe um nome para o template.');
            return;
        }

        if (selectedTasks.length === 0) {
            toast.warning('Selecione pelo menos uma tarefa para o template.');
            return;
        }

        // Verifica se o nome já existe (quando não estiver editando)
        if (!isEditing && templates[templateName]) {
            toast.warning('Já existe um template com este nome.');
            return;
        }

        setIsSaving(true);

        try {
            // Filtra as tarefas selecionadas
            const taskList = savedTasks.filter(task =>
                selectedTasks.includes(task.taskId)
            );

            // Calcula o total de minutos
            const totalMin = taskList.reduce((total, task) => {
                return total + task.entries.reduce((sum, entry) => sum + entry.minutes, 0);
            }, 0);

            // Monta o objeto do template
            const templateData = {
                name: templateName,
                tasks: taskList,
                totalMin
            };

            await window.go.backend.App.SaveTemplate(templateData);

            // Atualiza a lista de templates
            const updatedTemplates = {
                ...templates,
                [templateName]: templateData
            };
            setTemplates(updatedTemplates);

            toast.success(`Template ${isEditing ? 'atualizado' : 'salvo'} com sucesso!`);
            resetForm();
        } catch (error) {
            console.error('Erro ao salvar template:', error);
            toast.error('Erro ao salvar template.');
        } finally {
            setIsSaving(false);
        }
    };

    // Aplica um template às tarefas selecionadas
    const applyTemplateToTimelog = async (name) => {
        const template = templates[name];
        if (!template) return;

        try {
            // Primeiro salva todas as tarefas do template
            for (const task of template.tasks) {
                await window.go.backend.App.SaveTask(task);
            }

            toast.success('Template aplicado com sucesso! Vá para "Lançar Horas" para continuar.');
            setTimeout(() => {
                window.location.hash = '#/timelog';
            }, 1500);
        } catch (error) {
            console.error('Erro ao aplicar template:', error);
            toast.error('Erro ao aplicar template.');
        }
    };

    if (isLoading) {
        return (
            <div className="flex justify-center items-center h-full">
                <div className="animate-spin-slow w-12 h-12 border-4 border-primary-600 border-t-transparent rounded-full"></div>
            </div>
        );
    }

    return (
        <div>
            <div className="mb-6">
                <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">Templates</h1>
                <p className="text-gray-600 dark:text-gray-400">Salve conjuntos de tarefas para uso rápido</p>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* Formulário de Templates */}
                <div className="card">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                        <FiSave className="w-5 h-5 mr-2" />
                        {isEditing ? 'Editar Template' : 'Novo Template'}
                    </h2>

                    <form onSubmit={saveTemplate}>
                        <div className="mb-4">
                            <label htmlFor="templateName" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Nome do Template
                            </label>
                            <input
                                type="text"
                                id="templateName"
                                value={templateName}
                                onChange={(e) => setTemplateName(e.target.value)}
                                className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                                placeholder="Ex: Sprint Semanal"
                                required
                            />
                        </div>

                        <div className="mb-4">
                            <div className="flex justify-between items-center mb-2">
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                    Tarefas Incluídas
                                </label>
                                {savedTasks.length > 0 && (
                                    <button
                                        type="button"
                                        onClick={selectAllTasks}
                                        className="text-xs text-primary-600 hover:text-primary-700 dark:text-primary-500 dark:hover:text-primary-400"
                                    >
                                        {selectedTasks.length === savedTasks.length ? 'Desmarcar todas' : 'Selecionar todas'}
                                    </button>
                                )}
                            </div>

                            {savedTasks.length === 0 ? (
                                <p className="text-sm text-gray-500 dark:text-gray-400 py-2">
                                    Nenhuma tarefa disponível. Adicione tarefas na seção "Tarefas".
                                </p>
                            ) : (
                                <div className="overflow-y-auto max-h-60">
                                    <div className="space-y-2">
                                        {savedTasks.map(task => (
                                            <div
                                                key={task.taskId}
                                                className="flex items-start p-2 border border-gray-200 rounded-md dark:border-gray-700"
                                            >
                                                <input
                                                    type="checkbox"
                                                    checked={selectedTasks.includes(task.taskId)}
                                                    onChange={() => toggleTaskSelection(task.taskId)}
                                                    className="mt-1 w-4 h-4 text-primary-600 bg-gray-100 border-gray-300 rounded focus:ring-primary-500 dark:focus:ring-primary-600 dark:ring-offset-gray-800 dark:focus:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                                                />
                                                <div className="ml-3">
                                                    <p className="text-sm font-medium text-gray-900 dark:text-white">
                                                        {task.taskName}
                                                    </p>
                                                    <p className="text-xs text-gray-500 dark:text-gray-400">
                                                        {task.projectName} • {task.entries.length} entradas •
                                                        {task.entries.reduce((sum, e) => sum + e.minutes, 0)} min
                                                    </p>
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}
                        </div>

                        <div className="flex space-x-3">
                            {isEditing && (
                                <button
                                    type="button"
                                    onClick={resetForm}
                                    className="btn-secondary"
                                >
                                    Cancelar
                                </button>
                            )}

                            <button
                                type="submit"
                                disabled={isSaving || savedTasks.length === 0}
                                className="btn-primary flex items-center"
                            >
                                {isSaving ? (
                                    <>
                                        <FiLoader className="w-4 h-4 mr-2 animate-spin" />
                                        Salvando...
                                    </>
                                ) : (
                                    <>
                                        <FiSave className="w-4 h-4 mr-2" />
                                        {isEditing ? 'Atualizar Template' : 'Salvar Template'}
                                    </>
                                )}
                            </button>
                        </div>
                    </form>
                </div>

                {/* Lista de Templates */}
                <div className="card">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                        <FiFolder className="w-5 h-5 mr-2" />
                        Templates Salvos
                    </h2>

                    {Object.keys(templates).length === 0 ? (
                        <div className="text-center py-8">
                            <p className="text-gray-500 dark:text-gray-400 mb-2">
                                Nenhum template salvo.
                            </p>
                            <p className="text-sm text-gray-500 dark:text-gray-400">
                                Crie templates para facilitar o lançamento recorrente de horas.
                            </p>
                        </div>
                    ) : (
                        <div className="space-y-4">
                            {Object.entries(templates).map(([name, template]) => (
                                <div
                                    key={name}
                                    className="border border-gray-200 rounded-lg p-4 dark:border-gray-700"
                                >
                                    <div className="flex justify-between items-start">
                                        <div>
                                            <h3 className="text-md font-medium text-gray-900 dark:text-white">
                                                {name}
                                            </h3>
                                            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                                                {template.tasks.length} tarefas •
                                                {template.totalMin} minutos ({(template.totalMin / 60).toFixed(1)}h)
                                            </p>
                                        </div>
                                        <div className="flex space-x-1">
                                            <button
                                                onClick={() => editTemplate(name)}
                                                className="p-1.5 text-gray-500 hover:text-primary-600 dark:text-gray-400 dark:hover:text-primary-500"
                                            >
                                                <FiEdit className="w-5 h-5" />
                                            </button>
                                            <button
                                                onClick={() => deleteTemplate(name)}
                                                className="p-1.5 text-gray-500 hover:text-red-600 dark:text-gray-400 dark:hover:text-red-500"
                                            >
                                                <FiTrash2 className="w-5 h-5" />
                                            </button>
                                        </div>
                                    </div>

                                    <div className="mt-3">
                                        <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                            Tarefas incluídas:
                                        </h4>
                                        <ul className="space-y-1">
                                            {template.tasks.map(task => (
                                                <li key={task.taskId} className="text-sm text-gray-600 dark:text-gray-400">
                                                    • {task.taskName} ({task.entries.length} entradas,
                                                    {task.entries.reduce((sum, e) => sum + e.minutes, 0)} min)
                                                </li>
                                            ))}
                                        </ul>
                                    </div>

                                    <div className="mt-4">
                                        <button
                                            onClick={() => applyTemplateToTimelog(name)}
                                            className="w-full flex items-center justify-center text-sm px-3 py-2 bg-primary-50 text-primary-700 hover:bg-primary-100 rounded-md dark:bg-primary-900/30 dark:text-primary-400 dark:hover:bg-primary-900/50"
                                        >
                                            <FiCopy className="w-4 h-4 mr-2" />
                                            Aplicar Template
                                        </button>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default Templates;