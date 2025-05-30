import React, { useState, useEffect } from 'react';
import { toast } from 'react-toastify';
import { FiSave, FiEdit, FiTrash2, FiPlus, FiFolder, FiLoader, FiCopy, FiCheck, FiClock } from 'react-icons/fi';
import { useNavigate } from 'react-router-dom';

const Templates = () => {
    const navigate = useNavigate();
    const [isLoading, setIsLoading] = useState(true);
    const [templates, setTemplates] = useState({});
    const [savedTasks, setSavedTasks] = useState([]);
    const [selectedTasks, setSelectedTasks] = useState([]);
    const [templateName, setTemplateName] = useState('');
    const [isEditing, setIsEditing] = useState(false);
    const [currentTemplate, setCurrentTemplate] = useState(null);
    const [isSaving, setIsSaving] = useState(false);
    const [applyingTemplate, setApplyingTemplate] = useState(null);

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
        const loadData = async () => {
            try {
                setIsLoading(true);

                const templatesData = await window.go.backend.App.GetTemplates();
                setTemplates(templatesData);

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

    const resetForm = () => {
        setTemplateName('');
        setSelectedTasks([]);
        setIsEditing(false);
        setCurrentTemplate(null);
    };

    const toggleTaskSelection = (taskId) => {
        if (selectedTasks.includes(taskId)) {
            setSelectedTasks(selectedTasks.filter(id => id !== taskId));
        } else {
            setSelectedTasks([...selectedTasks, taskId]);
        }
    };

    const selectAllTasks = () => {
        if (selectedTasks.length === savedTasks.length) {
            setSelectedTasks([]);
        } else {
            setSelectedTasks(savedTasks.map(task => task.taskId));
        }
    };

    const editTemplate = (name) => {
        const template = templates[name];
        if (!template) return;

        setCurrentTemplate(name);
        setTemplateName(name);
        setIsEditing(true);

        const taskIds = template.tasks.map(task => task.taskId);
        setSelectedTasks(taskIds);
    };

    const deleteTemplate = async (name) => {
        if (!window.confirm(`Tem certeza que deseja excluir o template "${name}"?`)) {
            return;
        }

        try {
            await window.go.backend.App.DeleteTemplate(name);

            const updatedTemplates = { ...templates };
            delete updatedTemplates[name];
            setTemplates(updatedTemplates);

            await window.go.backend.App.ClearSavedTasks();
            localStorage.removeItem('templateApplied');

            toast.success('Template excluído com sucesso!');
        } catch (error) {
            console.error('Erro ao excluir template:', error);
            toast.error('Erro ao excluir template.');
        }
    };

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

        if (!isEditing && templates[templateName]) {
            toast.warning('Já existe um template com este nome.');
            return;
        }

        setIsSaving(true);

        try {
            const taskList = savedTasks.filter(task =>
                selectedTasks.includes(task.taskId)
            );

            const totalMin = taskList.reduce((total, task) => {
                return total + task.entries.reduce((sum, entry) => sum + entry.minutes, 0);
            }, 0);

            const templateData = {
                name: templateName,
                tasks: taskList,
                totalMin
            };

            await window.go.backend.App.SaveTemplate(templateData);

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

    const applyTemplateToTimelog = async (name) => {
        const template = templates[name];
        if (!template) return;

        try {
            setApplyingTemplate(name);

            await window.go.backend.App.ClearSavedTasks();

            for (const task of template.tasks) {
                await window.go.backend.App.SaveTask(task);
            }

            toast.success(`Template "${name}" aplicado com sucesso! Redirecionando para lançamento de horas...`);

            localStorage.setItem('templateApplied', 'true');

            setTimeout(() => {
                navigate('/timelog');
            }, 1500);
        } catch (error) {
            console.error('Erro ao aplicar template:', error);
            toast.error('Erro ao aplicar template: ' + (error.message || 'Erro desconhecido'));
        } finally {
            setApplyingTemplate(null);
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
                                                    {task.workingDays && (
                                                        <p className="text-xs text-blue-600 dark:text-blue-400 mt-1">
                                                            <FiClock className="inline w-3 h-3 mr-1" />
                                                            {formatWorkingDays(task.workingDays)}
                                                        </p>
                                                    )}
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
                                                    <span className="text-xs text-gray-500 dark:text-gray-500 ml-2">
                                                       - {formatWorkingDays(task.workingDays)}
                                                   </span>
                                                </li>
                                            ))}
                                        </ul>
                                    </div>

                                    <div className="mt-4">
                                        <button
                                            onClick={() => applyTemplateToTimelog(name)}
                                            disabled={applyingTemplate === name}
                                            className="w-full flex items-center justify-center text-sm px-3 py-2 bg-primary-50 text-primary-700 hover:bg-primary-100 rounded-md dark:bg-primary-900/30 dark:text-primary-400 dark:hover:bg-primary-900/50 disabled:opacity-50 disabled:cursor-not-allowed"
                                        >
                                            {applyingTemplate === name ? (
                                                <>
                                                    <FiLoader className="w-4 h-4 mr-2 animate-spin" />
                                                    Aplicando...
                                                </>
                                            ) : (
                                                <>
                                                    <FiCopy className="w-4 h-4 mr-2" />
                                                    Aplicar Template
                                                </>
                                            )}
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