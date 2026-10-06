import {useMemo, useRef, useState} from 'react';
import {toast} from 'react-toastify';
import {DeleteMultipleTimeEntries, LogMultipleTimes} from '@wailsjs/go/backend/App';
import {type DayConflict, paraBinding, type TimeLogResult, type WorkDay} from '../types/backend';
import {errMsg} from '../utils/errors';
import {buildRetryWorkDays} from '../utils/retry';
import {buildConflictWarning} from './usePlan';
import {useTimeEntriesSignal} from '../contexts/TimeEntriesContext';

// Resultado do lote com uma chave estável para a lista do React.
export type KeyedTimeLogResult = TimeLogResult & {_key: number};

export interface UseBatchSubmitOptions {
    workDays: WorkDay[];
    conflicts: DayConflict[];
    isCheckingConflicts: boolean;
    conflictCheckFailed: boolean;
    checkConflicts: (plan: WorkDay[]) => Promise<void>;
    refreshCalendar: () => void;
    reloadCalendar: () => void;
    onError?: (message: string | null) => void;
}

// Envio do plano em lote, reenvio só das falhas e desfazer do lote.
// `checkConflicts` reavalia o plano após sucessos (um segundo envio do mesmo
// plano passa a ser sinalizado como duplicata) e `refreshCalendar` atualiza o
// calendário.
const useBatchSubmit = ({
                            workDays,
                            conflicts,
                            isCheckingConflicts,
                            conflictCheckFailed,
                            checkConflicts,
                            refreshCalendar,
                            reloadCalendar,
                            onError
                        }: UseBatchSubmitOptions) => {
    const [results, setResults] = useState<KeyedTimeLogResult[]>([]);
    const [showResults, setShowResults] = useState(false);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [isRetrying, setIsRetrying] = useState(false);
    const [isUndoing, setIsUndoing] = useState(false);
    const {notifyChanged} = useTimeEntriesSignal();

    // Chave estável para cada resultado: (data, tarefa) se repete quando a
    // mesma tarefa tem várias entradas no dia, e o índice muda ao reenviar.
    const keySeq = useRef(0);
    const withKeys = (lista: readonly TimeLogResult[]): KeyedTimeLogResult[] => lista.map(r => ({...r, _key: ++keySeq.current}));

    const failedEntries = useMemo(() => results.filter(r => !r.success), [results]);
    // Só dá para desfazer o que veio com ID: sem ele o Teamwork criou a entrada
    // mas não temos como identificá-la para apagar.
    const undoableEntries = useMemo(() => results.filter(r => r.success && r.entryId > 0), [results]);
    const notUndoableCount = useMemo(
        () => results.filter(r => r.success && !(r.entryId > 0)).length,
        [results]
    );

    const submitPlan = async () => {
        if (workDays.length === 0) {
            toast.warning('Gere um plano antes de lançar horas.');
            return;
        }

        if (isCheckingConflicts) {
            toast.info('Aguarde a verificação de lançamentos existentes.');
            return;
        }

        if (conflicts.length > 0 && !window.confirm(buildConflictWarning(conflicts))) {
            return;
        }

        if (conflictCheckFailed && !window.confirm(
            'Não foi possível verificar se já existem lançamentos nos dias do plano.\n\n'
            + 'Enviar sem essa verificação pode duplicar horas, e não há como desfazer '
            + 'automaticamente.\n\nDeseja continuar mesmo assim?'
        )) {
            return;
        }

        setIsSubmitting(true);
        setResults([]);
        setShowResults(false);
        onError?.(null);

        const toastId = toast.info('Processando lançamentos...', {
            autoClose: false,
            closeButton: false
        });

        try {
            const batchResults = await LogMultipleTimes(paraBinding(workDays));
            toast.dismiss(toastId);

            if (!batchResults || batchResults.length === 0) {
                toast.error('Não foram recebidos resultados do lançamento.');
                onError?.('Não foram recebidos resultados do lançamento.');
                return;
            }

            setResults(withKeys(batchResults));
            setShowResults(true);

            const successes = batchResults.filter(r => r.success).length;
            const failures = batchResults.length - successes;

            if (successes > 0) {
                notifyChanged();
                await checkConflicts(workDays);
            }

            if (failures === 0) {
                toast.success(`${successes} lançamentos realizados com sucesso!`);
                setTimeout(() => refreshCalendar(), 1000);
            } else if (successes === 0) {
                toast.error(`Falha em todos os ${failures} lançamentos.`);
            } else {
                toast.warning(`${successes} lançamentos com sucesso e ${failures} falhas.`);
                setTimeout(() => refreshCalendar(), 1000);
            }
        } catch (error) {
            toast.dismiss(toastId);
            console.error('Erro ao lançar horas:', error);
            toast.error('Erro ao lançar horas no Teamwork: ' + errMsg(error));
            onError?.('Erro ao lançar horas: ' + errMsg(error));
        } finally {
            setIsSubmitting(false);
        }
    };

    const retryFailed = async () => {
        if (failedEntries.length === 0) return;

        const retryWorkDays = buildRetryWorkDays(results, workDays);
        if (retryWorkDays.length === 0) {
            toast.info('Não há entradas para reenviar.');
            return;
        }

        setIsRetrying(true);
        const toastId = toast.info('Reenviando lançamentos que falharam...', {
            autoClose: false,
            closeButton: false
        });

        try {
            const retryResults = await LogMultipleTimes(paraBinding(retryWorkDays));
            toast.dismiss(toastId);

            if (!retryResults || retryResults.length === 0) {
                toast.error('Não foram recebidos resultados do reenvio.');
                return;
            }

            // Mantém os sucessos anteriores e troca as falhas pelo resultado do
            // reenvio, preservando os entryId para que o desfazer continue válido.
            setResults(prev => [...prev.filter(r => r.success), ...withKeys(retryResults)]);

            const novosSucessos = retryResults.filter(r => r.success).length;
            const aindaFalha = retryResults.length - novosSucessos;

            if (novosSucessos > 0) {
                notifyChanged();
                await checkConflicts(workDays);
                setTimeout(() => refreshCalendar(), 1000);
            }

            if (aindaFalha === 0) {
                toast.success(`${novosSucessos} lançamento(s) reenviado(s) com sucesso!`);
            } else if (novosSucessos === 0) {
                toast.error(`Falha novamente em ${aindaFalha} lançamento(s).`);
            } else {
                toast.warning(`${novosSucessos} reenviado(s), ${aindaFalha} ainda falharam.`);
            }
        } catch (error) {
            toast.dismiss(toastId);
            console.error('Erro ao reenviar lançamentos:', error);
            toast.error('Erro ao reenviar lançamentos: ' + errMsg(error));
        } finally {
            setIsRetrying(false);
        }
    };

    const undoBatch = async () => {
        if (undoableEntries.length === 0) return;

        const confirmacao =
            `Desfazer o lançamento apagará ${undoableEntries.length} entrada(s) do Teamwork.\n\n`
            + (notUndoableCount > 0
                ? `${notUndoableCount} entrada(s) NÃO serão apagadas porque o Teamwork não devolveu `
                + `o identificador delas — remova-as pelo Gerenciador de Apontamentos.\n\n`
                : '')
            + `Esta ação não pode ser revertida. Continuar?`;

        if (!window.confirm(confirmacao)) return;

        setIsUndoing(true);
        try {
            const entryIds = undoableEntries.map(r => r.entryId);
            const undoResults = (await DeleteMultipleTimeEntries(entryIds)) || [];

            const removed = undoResults.filter(r => r.success).length;
            const failed = undoResults.length - removed;
            if (removed > 0) notifyChanged();

            if (failed === 0) {
                toast.success(`${removed} lançamento(s) desfeito(s).`);
                // Só limpa o painel quando tudo saiu; senão o usuário perde a
                // lista do que ainda precisa remover à mão.
                setResults([]);
                setShowResults(false);
            } else {
                toast.warning(`${removed} desfeito(s), ${failed} não puderam ser removidos.`);
                setResults(prev => prev.filter(r => {
                    const undone = undoResults.find(u => u.entryId === r.entryId && u.success);
                    return !undone;
                }));
            }

            reloadCalendar();
        } catch (error) {
            console.error('Erro ao desfazer lançamentos:', error);
            toast.error('Erro ao desfazer lançamentos: ' + errMsg(error));
        } finally {
            setIsUndoing(false);
        }
    };

    return {
        results,
        showResults,
        isSubmitting,
        isRetrying,
        isUndoing,
        failedEntries,
        undoableEntries,
        notUndoableCount,
        submitPlan,
        retryFailed,
        undoBatch
    };
};

export default useBatchSubmit;
