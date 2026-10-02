import {useEffect, useState} from 'react';
import {useLocation, useNavigate} from 'react-router-dom';
import {toast} from 'react-toastify';
import {GetSavedTasks} from '../../wailsjs/go/backend/App';
import {errMsg} from '../utils/errors';

// Tarefas salvas do TimeLog e a seleção do usuário. Carrega uma única vez na
// montagem (antes eram duas chamadas a GetSavedTasks) e, se o usuário veio de
// "Aplicar Template", pré-seleciona as tarefas do template a partir do state da
// navegação.
const useSavedTasks = () => {
    const location = useLocation();
    const navigate = useNavigate();
    const [savedTasks, setSavedTasks] = useState([]);
    const [selectedTasks, setSelectedTasks] = useState([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState(null);
    const [appliedTemplate, setAppliedTemplate] = useState(null);

    useEffect(() => {
        let cancelled = false;
        const templateState = location.state?.templateApplied ? location.state : null;

        const load = async () => {
            try {
                const tasks = (await GetSavedTasks()) || [];
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

        load();

        if (templateState) {
            // Consome o aviso: voltar/recarregar não deve reaplicar a seleção.
            navigate(location.pathname, {replace: true, state: null});
        }

        return () => {
            cancelled = true;
        };
        // Só na montagem: o state do template é lido uma vez e descartado.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const toggleTaskSelection = (taskId) => {
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
