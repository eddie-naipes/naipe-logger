import type {DeleteTimeEntryResult, TimeLogResult, WorkDay} from '../types/backend';

// Reconstrói apenas as entradas que falharam a partir do plano original. O
// casamento é por (data, tarefa) — não pela posição do resultado — e reenviamos
// no máximo a quantidade de falhas de cada chave, para nunca reenviar uma
// entrada que já deu certo e acabar duplicando horas.
export const buildRetryWorkDays = (
    results: readonly Pick<TimeLogResult, 'success' | 'date' | 'taskId'>[],
    workDays: readonly WorkDay[]
): WorkDay[] => {
    const restante = new Map<string, number>();
    for (const r of results) {
        if (r.success) continue;
        const chave = `${r.date}::${r.taskId}`;
        restante.set(chave, (restante.get(chave) ?? 0) + 1);
    }

    const dias: WorkDay[] = [];
    for (const dia of workDays) {
        const entradas = (dia.entries ?? []).filter(entrada => {
            const chave = `${dia.date}::${entrada.taskId}`;
            const falhas = restante.get(chave) ?? 0;
            if (falhas > 0) {
                restante.set(chave, falhas - 1);
                return true;
            }
            return false;
        });
        if (entradas.length > 0) {
            dias.push({...dia, entries: entradas});
        }
    }
    return dias;
};

// IDs a reenviar para exclusão: só os que falharam, sem repetir, e nunca um ID
// que também aparece como excluído com sucesso no mesmo lote.
export const failedDeleteIds = (results: readonly DeleteTimeEntryResult[]): number[] => {
    const excluidos = new Set(results.filter(r => r.success).map(r => r.entryId));
    const ids = results
        .filter(r => !r.success && !excluidos.has(r.entryId))
        .map(r => r.entryId);
    return [...new Set(ids)];
};
