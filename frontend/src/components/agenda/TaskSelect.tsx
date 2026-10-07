import type {Task} from '../../types/backend';
import type {AgendaTaskRef} from '../../utils/agenda';

interface TaskSelectProps {
    id: string;
    value: AgendaTaskRef | null;
    savedTasks: readonly Task[];
    onChange: (task: AgendaTaskRef | null) => void;
    emptyLabel: string;
    className?: string;
    ariaLabel?: string;
}

export const tarefaSalvaParaRef = (t: Task): AgendaTaskRef => ({
    taskId: t.taskId,
    taskName: t.taskName,
    projectId: t.projectId,
    projectName: t.projectName
});

// Seletor de tarefa a partir das tarefas salvas. Uma tarefa já escolhida que
// não está mais salva continua aparecendo, para não ser trocada sem querer.
const TaskSelect = ({id, value, savedTasks, onChange, emptyLabel, className, ariaLabel}: TaskSelectProps) => {
    const atual = value && value.taskId > 0 ? value : null;
    const foraDaLista = atual && !savedTasks.some(t => t.taskId === atual.taskId);

    return (
        <select
            id={id}
            aria-label={ariaLabel}
            value={atual ? String(atual.taskId) : ''}
            onChange={e => {
                const taskId = Number(e.target.value);
                if (!taskId) {
                    onChange(null);
                    return;
                }
                if (atual && atual.taskId === taskId) return;
                const salva = savedTasks.find(t => t.taskId === taskId);
                onChange(salva ? tarefaSalvaParaRef(salva) : null);
            }}
            className={className}
        >
            <option value="">{emptyLabel}</option>
            {foraDaLista && (
                <option value={String(atual.taskId)}>
                    {atual.taskName || `Tarefa ${atual.taskId}`}{atual.projectName ? ` (${atual.projectName})` : ''}
                </option>
            )}
            {savedTasks.map(t => (
                <option key={t.taskId} value={String(t.taskId)}>
                    {t.taskName}{t.projectName ? ` (${t.projectName})` : ''}
                </option>
            ))}
        </select>
    );
};

export default TaskSelect;
