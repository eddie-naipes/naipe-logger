import {useCallback, useRef, useState} from 'react';
import {toast} from 'react-toastify';
import {CheckPlanConflicts, CreateDistributionPlan, GetWorkingDays} from '@wailsjs/go/backend/App';
import {type DayConflict, paraBinding, type Task, type WorkDay} from '../types/backend';
import {errMsg} from '../utils/errors';

export interface GeneratePlanParams {
    start: string;
    end: string;
    taskIds: readonly number[];
}

// Texto do window.confirm mostrado antes de enviar um plano com conflitos.
export const buildConflictWarning = (conflicts: readonly DayConflict[]): string => {
    const dias = conflicts.map(c => {
        const horasExistentes = (c.existingMinutes / 60).toFixed(1);
        const mesmaTarefa = (c.sameTask && c.sameTask.length > 0)
            ? ` — ${c.sameTask.length} na(s) MESMA(S) tarefa(s) do plano`
            : '';
        return `• ${c.date}: já tem ${horasExistentes}h em ${c.existingEntries} entrada(s)${mesmaTarefa}`;
    }).join('\n');

    return `ATENÇÃO: ${conflicts.length} dia(s) do plano já possuem tempo lançado.\n\n`
        + `${dias}\n\n`
        + `Enviar mesmo assim vai DUPLICAR essas horas. Não há como desfazer `
        + `automaticamente — a correção seria apagar cada entrada manualmente.\n\n`
        + `Deseja continuar?`;
};

// Geração do plano de lançamento e verificação de conflitos com o que já
// existe no Teamwork.
const usePlan = (savedTasks: readonly Task[], onError?: (message: string | null) => void) => {
    const [workDays, setWorkDays] = useState<WorkDay[]>([]);
    const [conflicts, setConflicts] = useState<DayConflict[]>([]);
    const [isGenerating, setIsGenerating] = useState(false);
    const [isCheckingConflicts, setIsCheckingConflicts] = useState(false);
    const [conflictCheckFailed, setConflictCheckFailed] = useState(false);

    // Só a geração mais recente pode gravar o plano (ex.: dois cliques em dias
    // do calendário em sequência).
    const planRequestRef = useRef(0);
    const conflictRequestRef = useRef(0);

    // Verifica se os dias do plano já têm tempo lançado no Teamwork. Como não
    // existe rollback, um lote duplicado só se desfaz apagando entrada a entrada.
    const checkConflicts = useCallback(async (plan: WorkDay[]): Promise<void> => {
        const requestId = ++conflictRequestRef.current;
        setIsCheckingConflicts(true);
        setConflictCheckFailed(false);
        try {
            const found = await CheckPlanConflicts(paraBinding(plan));
            if (requestId !== conflictRequestRef.current) return;
            setConflicts(found || []);

            if (found && found.length > 0) {
                toast.warning(`${found.length} dia(s) do plano já possuem lançamentos. Revise antes de enviar.`);
            }
        } catch (error) {
            if (requestId !== conflictRequestRef.current) return;
            console.error('Erro ao verificar lançamentos existentes:', error);
            setConflicts([]);
            setConflictCheckFailed(true);
        } finally {
            if (requestId === conflictRequestRef.current) setIsCheckingConflicts(false);
        }
    }, []);

    // generatePlan recebe período e tarefas por parâmetro: chamada logo após um
    // setState (clique no calendário), a closure ainda veria o estado antigo.
    const generatePlan = useCallback(async ({start, end, taskIds}: GeneratePlanParams): Promise<void> => {
        if (!taskIds || taskIds.length === 0) {
            toast.warning('Selecione pelo menos uma tarefa para lançar horas.');
            return;
        }

        if (!start || !end) {
            toast.warning('Selecione um intervalo de datas válido.');
            return;
        }

        if (start > end) {
            toast.warning('A data inicial deve ser anterior ou igual à data final.');
            return;
        }

        const requestId = ++planRequestRef.current;
        const isCurrent = () => requestId === planRequestRef.current;

        setIsGenerating(true);
        setWorkDays([]);
        setConflicts([]);
        setConflictCheckFailed(false);
        onError?.(null);

        try {
            const workingDays = await GetWorkingDays(start, end);
            if (!isCurrent()) return;

            if (!workingDays || workingDays.length === 0) {
                toast.warning('Não foram encontrados dias úteis no período selecionado.');
                return;
            }

            const filteredTasks = savedTasks.filter(task => taskIds.includes(task.taskId));

            const plan = await CreateDistributionPlan(workingDays, paraBinding(filteredTasks));
            if (!isCurrent()) return;

            if (!plan || plan.length === 0) {
                toast.warning('Não foi possível gerar um plano de lançamento.');
                return;
            }

            setWorkDays(plan);
            toast.success(`Plano gerado com sucesso para ${plan.length} dias!`);

            await checkConflicts(plan);
        } catch (error) {
            if (!isCurrent()) return;
            console.error('Erro ao gerar plano:', error);
            toast.error('Erro ao gerar plano de lançamento: ' + errMsg(error));
            onError?.('Erro ao gerar plano: ' + errMsg(error));
        } finally {
            if (isCurrent()) setIsGenerating(false);
        }
    }, [savedTasks, checkConflicts, onError]);

    return {
        workDays,
        conflicts,
        isGenerating,
        isCheckingConflicts,
        conflictCheckFailed,
        generatePlan,
        checkConflicts
    };
};

export default usePlan;
