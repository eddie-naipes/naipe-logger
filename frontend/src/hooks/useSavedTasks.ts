import {useEffect, useState} from 'react';
import {useLocation, useNavigate} from 'react-router';
import {toast} from 'react-toastify';
import {GetSavedTasks} from '@wailsjs/go/backend/App';
import type {Task} from '../types/backend';
import {errMsg} from '../utils/errors';

// State de navegação deixado pela tela de Templates ao aplicar um template.
export interface TemplateNavigationState {
    templateApplied: string;
    taskIds?: number[];
}

const isTemplateState = (state: unknown): state is TemplateNavigationState =>
    typeof state === 'object' && state !== null
    && typeof (state as {templateApplied?: unknown}).templateApplied === 'string'
    && (state as {templateApplied: string}).templateApplied !== '';

// Tarefas salvas do TimeLog e a seleção do usuário. Carrega uma única vez na
// montagem (antes eram duas chamadas a GetSavedTasks) e, se o usuário veio de
// "Aplicar Template", pré-seleciona as tarefas do template a partir do state da
// navegação.
const useSavedTasks = () => {
    const location = useLocation();
    const navigate = useNavigate();
    const [savedTasks, setSavedTasks] = useState<Task[]>([]);
    const [selectedTasks, setSelectedTasks] = useState<number[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [appliedTemplate, setAppliedTemplate] = useState<string | null>(null);

    useEffect(() => {
        let cancelled = false;
        const templateState = isTemplateState(location.state) ? location.state : null;

        const load = async () => {
            try {
                const tasks: Task[] = (await GetSavedTasks()) ?? [];
                if (cancelled) return;
                setSavedTasks(tasks);

                if (templateState && tasks.length > 0) {
                    const idsDoTemplate = Array.isArray(templateState.taskIds) ? templateState.taskIds : null;
                    const ids = tasks
                        .map(task => task.taskId)
                        .filter(id => !idsDoTemplate || idsDoTemplate.includes(id));
                    setSelectedTasks(ids);
                    setAppliedTemplate(templateState.templateApplied);
                    toast.info(`${ids.length} tarefas carregadas do template. Clique em "Gerar Plano" para continuar.`);
                }
            } catch (err) {
                if (cancelled) return;
                console.error('Erro ao carregar tarefas salvas:', err);
                toast.error('Erro ao carregar tarefas salvas: ' + errMsg(err));
                setError('Erro ao carregar tarefas salvas: ' + errMsg(err));
            } finally {
                if (!cancelled) setIsLoading(false);
            }
        };

        void load();

        if (templateState) {
            // Consome o aviso: voltar/recarregar não deve reaplicar a seleção.
            void navigate(location.pathname, {replace: true, state: null});
        }

        return () => {
            cancelled = true;
        };
        // Só na montagem: o state do template é lido uma vez e descartado.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const toggleTaskSelection = (taskId: number) => {
        setSelectedTasks(prev => (
            prev.includes(taskId) ? prev.filter(id => id !== taskId) : [...prev, taskId]
        ));
    };

    const selectAllTasks = () => {
        setSelectedTasks(prev => (
            prev.length === savedTasks.length ? [] : savedTasks.map(task => task.taskId)
        ));
    };

    return {
        savedTasks,
        selectedTasks,
        toggleTaskSelection,
        selectAllTasks,
        isLoading,
        error,
        appliedTemplate
    };
};

export default useSavedTasks;
