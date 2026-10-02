import {useCallback, useRef, useState} from 'react';
import {toast} from 'react-toastify';
import {CheckPlanConflicts} from '@wailsjs/go/backend/App';
import {type DayConflict, paraBinding, type WorkDay} from '../types/backend';

// Revisão de um plano montado fora do usePlan (completar período, copiar
// semana): guarda o plano e verifica conflitos com o que já existe no Teamwork.
// Devolve os mesmos campos que useBatchSubmit espera do usePlan, para que
// envio, reenviar falhas e desfazer funcionem igual ao TimeLog.
const usePlanReview = () => {
    const [workDays, setWorkDays] = useState<WorkDay[]>([]);
    const [conflicts, setConflicts] = useState<DayConflict[]>([]);
    const [isCheckingConflicts, setIsCheckingConflicts] = useState(false);
    const [conflictCheckFailed, setConflictCheckFailed] = useState(false);

    // Só a verificação mais recente pode gravar o resultado.
    const requestRef = useRef(0);

    const checkConflicts = useCallback(async (plan: WorkDay[]): Promise<void> => {
        const requestId = ++requestRef.current;
        setIsCheckingConflicts(true);
        setConflictCheckFailed(false);
        try {
            const found = await CheckPlanConflicts(paraBinding(plan));
            if (requestId !== requestRef.current) return;
            setConflicts(found || []);
            if (found && found.length > 0) {
                toast.warning(`${found.length} dia(s) do plano já possuem lançamentos. Revise antes de enviar.`);
            }
        } catch (error) {
            if (requestId !== requestRef.current) return;
            console.error('Erro ao verificar lançamentos existentes:', error);
            setConflicts([]);
            setConflictCheckFailed(true);
        } finally {
            if (requestId === requestRef.current) setIsCheckingConflicts(false);
        }
    }, []);

    const setPlan = useCallback(async (plan: WorkDay[]): Promise<void> => {
        setWorkDays(plan);
        setConflicts([]);
        setConflictCheckFailed(false);
        if (plan.length > 0) {
            await checkConflicts(plan);
        } else {
            // Invalida uma verificação em andamento do plano anterior.
            requestRef.current++;
            setIsCheckingConflicts(false);
        }
    }, [checkConflicts]);

    return {workDays, conflicts, isCheckingConflicts, conflictCheckFailed, checkConflicts, setPlan};
};

export default usePlanReview;
