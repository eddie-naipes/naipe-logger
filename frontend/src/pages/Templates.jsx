import React, { useState, useEffect } from 'react';
import { toast } from 'react-toastify';
import { FiSave, FiEdit, FiTrash2, FiFolder, FiLoader, FiCopy, FiClock } from 'react-icons/fi';
import { useNavigate } from 'react-router-dom';
import {
    ApplyTemplate,
    ClearSavedTasks,
    DeleteTemplate,
    GetSavedTasks,
    GetTemplates,
    SaveTemplate
} from '../../wailsjs/go/backend/App';
import {errMsg} from '../utils/errors';
import {formatWorkingDays, sumEntryMinutes} from '../utils/time';

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

    useEffect(() => {
        const loadData = async () => {
            try {
                setIsLoading(true);

                const [templatesData, tasksData] = await Promise.all([
                    GetTemplates(),
                    GetSavedTasks()
                ]);
                setTemplates(templatesData || {});
                setSavedTasks(tasksData || []);
            } catch (error) {
                console.error('Erro ao carregar dados:', error);
                toast.error('Erro ao carregar templates e tarefas: ' + errMsg(error));
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
            await DeleteTemplate(name);

            // Excluir um template não mexe nas tarefas salvas: elas pertencem à
            // tela de Tarefas e podem estar em outros templates.
            setTemplates(prev => {
                const updated = { ...prev };
                delete updated[name];
                return updated;
            });

            if (currentTemplate === name) {
                resetForm();
            }

            toast.success('Template excluído com sucesso!');
        } catch (error) {
            console.error('Erro ao excluir template:', error);
            toast.error('Erro ao excluir template: ' + errMsg(error));
        }
    };

    const saveTemplate = async (e) => {
        e.preventDefault();

        const nome = templateName.trim();

        if (!nome) {
            toast.warning('Informe um nome para o template.');
            return;
        }

        if (selectedTasks.length === 0) {
            toast.warning('Selecione pelo menos uma tarefa para o template.');
            return;
        }

        // Renomear = salvar com o novo nome e remover o antigo; sem isso o
        // template antigo continuava existindo como duplicata.
        const renomeando = isEditing && currentTemplate !== null && currentTemplate !== nome;

        if (templates[nome] && (!isEditing || renomeando)) {
            toast.warning(`Já existe um template chamado "${nome}".`);
            return;
        }

        setIsSaving(true);

        try {
            const taskList = savedTasks.filter(task =>
                selectedTasks.includes(task.taskId)
            );

            const totalMin = taskList.reduce((total, task) => total + sumEntryMinutes(task.entries), 0);

            const templateData = {
                name: nome,
                tasks: taskList,
                totalMin
            };

            await SaveTemplate(templateData);

            let antigoRemovido = true;
            if (renomeando) {
                try {
                    await DeleteTemplate(currentTemplate);
                } catch (error) {
                    antigoRemovido = false;
                    console.error('Erro ao remover template renomeado:', error);
                    toast.warning(`Template salvo como "${nome}", mas o antigo "${currentTemplate}" não pôde ser removido: ${errMsg(error)}`);
                }
            }

            setTemplates(prev => {
                const updated = { ...prev, [nome]: templateData };
                if (renomeando && antigoRemovido) {
                    delete updated[currentTemplate];
                }
                return updated;
            });

            toast.success(`Template ${isEditing ? 'atualizado' : 'salvo'} com sucesso!`);
            resetForm();
        } catch (error) {
            console.error('Erro ao salvar template:', error);
            toast.error('Erro ao salvar template: ' + errMsg(error));
        } finally {
            setIsSaving(false);
        }
    };

    const applyTemplateToTimelog = async (name) => {
        const template = templates[name];
        if (!template) return;

        try {
            setApplyingTemplate(name);

            // Aplicar substitui a lista de tarefas salvas pelas do template.
            // ApplyTemplate faz no backend o mesmo que um SaveTask por tarefa,
            // numa única chamada.
            await ClearSavedTasks();
            await ApplyTemplate(name);

            toast.success(`Template "${name}" aplicado com sucesso! Configure o período e gere o plano.`);

            // O aviso de template aplicado viaja no state da navegação, e não em
            // localStorage, para não vazar para visitas futuras ao TimeLog.
            navigate('/timelog', {
                state: {
                    templateApplied: name,
                    taskIds: template.tasks.map(task => task.taskId)
                }
            });
        } catch (error) {
            console.error('Erro ao aplicar template:', error);
            toast.error('Erro ao aplicar template: ' + errMsg(error));
            setApplyingTemplate(null);
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
                                <span id="templateTasksLabel" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                    Tarefas Incluídas
                                </span>
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
                                                    aria-label={`Incluir tarefa ${task.taskName} no template`}
                                                    className="mt-1 w-4 h-4 text-primary-600 bg-gray-100 border-gray-300 rounded focus:ring-primary-500 dark:focus:ring-primary-600 dark:ring-offset-gray-800 dark:focus:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                                                />
                                                <div className="ml-3">
                                                    <p className="text-sm font-medium text-gray-900 dark:text-white">
                                                        {task.taskName}
                                                    </p>
                                                    <p className="text-xs text-gray-500 dark:text-gray-400">
                                                        {task.projectName} • {task.entries.length} entradas •
                                                        {sumEntryMinutes(task.entries)} min
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
                                                type="button"
                                                onClick={() => editTemplate(name)}
                                                aria-label={`Editar template ${name}`}
                                                title="Editar template"
                                                className="p-1.5 text-gray-500 hover:text-primary-600 dark:text-gray-400 dark:hover:text-primary-500"
                                            >
                                                <FiEdit className="w-5 h-5" aria-hidden="true" />
                                            </button>
                                            <button
                                                type="button"
                                                onClick={() => deleteTemplate(name)}
                                                aria-label={`Excluir template ${name}`}
                                                title="Excluir template"
                                                className="p-1.5 text-gray-500 hover:text-red-600 dark:text-gray-400 dark:hover:text-red-500"
                                            >
                                                <FiTrash2 className="w-5 h-5" aria-hidden="true" />
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
                                                    {sumEntryMinutes(task.entries)} min)
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