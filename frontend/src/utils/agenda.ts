import {addDays, startOfWeek} from 'date-fns';
import type {agenda} from '@wailsjs/go/models';
import type {Dados, Task, TimeLogResult, WorkDay} from '../types/backend';
import {toYMD} from './dates';

export type AgendaItem = Dados<agenda.PlanItem>;
export type AgendaTaskRef = Dados<agenda.TaskRef>;
export type AgendaImportedRecord = Dados<agenda.ImportedRecord>;

export type AgendaPeriodo = 'hoje' | 'ontem' | 'semana';

// Atalhos de período. "Esta semana" vai de segunda até hoje.
export const periodoAgenda = (atalho: AgendaPeriodo, hoje: Date = new Date()): {start: string; end: string} => {
    switch (atalho) {
        case 'ontem': {
            const ontem = toYMD(addDays(hoje, -1));
            return {start: ontem, end: ontem};
        }
        case 'semana':
            return {start: toYMD(startOfWeek(hoje, {weekStartsOn: 1})), end: toYMD(hoje)};
        default:
            return {start: toYMD(hoje), end: toYMD(hoje)};
    }
};

// Tarefa efetiva do item: a escolhida na tela vence a da regra.
export const tarefaDoItem = (item: AgendaItem, escolhidas: ReadonlyMap<string, AgendaTaskRef>): AgendaTaskRef | null => {
    const escolhida = escolhidas.get(item.key);
    if (escolhida && escolhida.taskId > 0) return escolhida;
    return item.task.taskId > 0 ? item.task : null;
};

// Só entram no plano eventos mapeados ou sem regra (com tarefa escolhida);
// ignorados e já lançados ficam de fora.
export const itemSelecionavel = (item: AgendaItem): boolean =>
    item.status === 'mapped' || item.status === 'unmapped';

// Itens que vão para o plano, na ordem (data, horário). É essa ordem que
// casa os resultados do envio com os eventos (ver casarImportados).
export const itensDoPlano = (
    items: readonly AgendaItem[],
    selecionados: ReadonlySet<string>,
    escolhidas: ReadonlyMap<string, AgendaTaskRef>
): AgendaItem[] => items
    .filter(item => itemSelecionavel(item) && selecionados.has(item.key) && item.minutes > 0)
    .map(item => ({...item, task: tarefaDoItem(item, escolhidas) ?? item.task}))
    .filter(item => item.task.taskId > 0)
    .sort((a, b) => (a.date === b.date ? a.startTime.localeCompare(b.startTime) : a.date.localeCompare(b.date)));

// Monta os WorkDay do envio: uma entrada por evento, com o horário real.
export const montarWorkDays = (planItems: readonly AgendaItem[]): WorkDay[] => {
    const dias = new Map<string, WorkDay>();
    for (const item of planItems) {
        const dia = dias.get(item.date) ?? {date: item.date, entries: [], totalMin: 0};
        dia.entries.push({
            taskId: item.task.taskId,
            entry: {
                minutes: item.minutes,
                userId: 0,
                time: `${item.startTime}:00`,
                description: item.description || item.title,
                isBillable: item.billable,
                date: item.date
            }
        });
        dia.totalMin += item.minutes;
        dias.set(item.date, dia);
    }
    return [...dias.values()].sort((a, b) => a.date.localeCompare(b.date));
};

// Tarefas do plano no formato que o PlanPreview usa para mostrar nomes.
export const tarefasDoPlano = (planItems: readonly AgendaItem[]): Task[] => {
    const vistas = new Map<number, Task>();
    for (const {task} of planItems) {
        if (!vistas.has(task.taskId)) {
            vistas.set(task.taskId, {
                taskId: task.taskId,
                taskName: task.taskName,
                projectId: task.projectId,
                projectName: task.projectName,
                entries: []
            });
        }
    }
    return [...vistas.values()];
};

// Casa os resultados bem-sucedidos com os eventos enviados. Os resultados não
// trazem o horário, então o casamento é por (data, tarefa): o n-ésimo sucesso
// de uma chave vai para o n-ésimo evento dessa chave no plano — o mesmo
// critério do "reenviar só as falhas".
export const casarImportados = (
    results: readonly Pick<TimeLogResult, 'success' | 'date' | 'taskId' | 'entryId'>[],
    planItems: readonly AgendaItem[]
): AgendaImportedRecord[] => {
    const filas = new Map<string, AgendaItem[]>();
    for (const item of planItems) {
        const chave = `${item.date}::${item.task.taskId}`;
        filas.set(chave, [...(filas.get(chave) ?? []), item]);
    }
    const registros: AgendaImportedRecord[] = [];
    for (const r of results) {
        if (!r.success) continue;
        const fila = filas.get(`${r.date}::${r.taskId}`);
        const item = fila?.shift();
        if (!item) continue;
        registros.push({key: item.key, date: item.date, taskId: item.task.taskId, entryId: r.entryId, importedAt: ''});
    }
    return registros;
};
