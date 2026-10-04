import {type KeyboardEvent, useState} from 'react';
import {toast} from 'react-toastify';
import {FiList} from 'react-icons/fi';
import clsx from 'clsx';
import type {NonWorkingDaysMap} from '../../hooks/useNonWorkingDays';
import {formatDateBR} from '../../utils/dates';
import {DIAS_ABREV} from '../../utils/time';
import {
    computeTotals,
    dayStatus,
    type DayStatus,
    formatHHMM,
    parseDuration,
    type WeekGrid,
    type WeekRow
} from '../../utils/weekGrid';

const COLUNA_STATUS: Record<DayStatus, string> = {
    nonworking: 'bg-gray-100 dark:bg-gray-900/60',
    future: '',
    below: 'bg-amber-50 dark:bg-amber-900/10',
    complete: ''
};

interface CellInputProps {
    minutes: number;
    label: string;
    disabled: boolean;
    onCommit: (target: number) => void;
}

// Célula editável. O rascunho é local; o pai recria a célula (key com os
// minutos) quando a grade recarrega, então ela volta ao valor real sozinha.
const CellInput = ({minutes, label, disabled, onCommit}: CellInputProps) => {
    const original = minutes > 0 ? formatHHMM(minutes) : '';
    const [draft, setDraft] = useState(original);

    const commit = () => {
        if (draft.trim() === original) return;
        const alvo = parseDuration(draft);
        if (alvo === null) {
            toast.warning(`"${draft}" não é uma duração válida. Use hh:mm, 7h30 ou 1,5.`);
            setDraft(original);
            return;
        }
        // Volta ao valor real: o aumento/redução só vale depois de confirmado
        // no diálogo e da grade recarregar.
        setDraft(original);
        if (alvo !== minutes) onCommit(alvo);
    };

    const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'Enter') {
            e.currentTarget.blur();
        } else if (e.key === 'Escape') {
            setDraft(original);
        }
    };

    return (
        <input
            type="text"
            inputMode="decimal"
            aria-label={label}
            value={draft}
            placeholder="—"
            disabled={disabled}
            onChange={e => setDraft(e.target.value)}
            onBlur={commit}
            onKeyDown={onKeyDown}
            className="w-16 text-center tabular-nums text-sm rounded border border-transparent bg-transparent px-1 py-1 hover:border-gray-300 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:bg-white dark:text-white dark:hover:border-gray-600 dark:focus:bg-gray-700 disabled:cursor-not-allowed"
        />
    );
};

interface WeekGridTableProps {
    grid: WeekGrid;
    visibleDays: readonly number[];
    nonWorkingDays: NonWorkingDaysMap;
    jornada: number;
    today: string;
    disabled: boolean;
    onCellCommit: (row: WeekRow, dayIndex: number, target: number) => void;
    onOpenCell: (row: WeekRow, dayIndex: number) => void;
}

// Grade tarefa × dia com totais por linha e por coluna.
const WeekGridTable = ({
                           grid,
                           visibleDays,
                           nonWorkingDays,
                           jornada,
                           today,
                           disabled,
                           onCellCommit,
                           onOpenCell
                       }: WeekGridTableProps) => {
    const totals = computeTotals(grid.rows ?? []);
    const dias = grid.days ?? [];

    const statusDoDia = (i: number): DayStatus => {
        const data = dias[i] ?? '';
        return dayStatus(data, totals.perDay[i] ?? 0, jornada, Boolean(nonWorkingDays[data]), today);
    };

    return (
        <div className="overflow-x-auto">
            <table className="min-w-full text-sm">
                <thead>
                    <tr className="border-b border-gray-200 dark:border-gray-700">
                        <th scope="col" className="py-2 pr-4 text-left font-medium text-gray-600 dark:text-gray-400">
                            Tarefa
                        </th>
                        {visibleDays.map(i => {
                            const data = dias[i] ?? '';
                            const naoUtil = nonWorkingDays[data];
                            return (
                                <th
                                    key={data}
                                    scope="col"
                                    className={clsx('py-2 px-1 text-center font-medium', COLUNA_STATUS[statusDoDia(i)],
                                        data === today ? 'text-primary-700 dark:text-primary-300' : 'text-gray-600 dark:text-gray-400')}
                                >
                                    <div>{DIAS_ABREV[(i + 1) % 7]} {formatDateBR(data, 'dd/MM')}</div>
                                    {naoUtil?.type === 'holiday' && (
                                        <div className="text-xs font-normal text-red-600 dark:text-red-400 truncate max-w-[6rem] mx-auto" title={naoUtil.name}>
                                            {naoUtil.name}
                                        </div>
                                    )}
                                </th>
                            );
                        })}
                        <th scope="col" className="py-2 pl-2 text-right font-medium text-gray-600 dark:text-gray-400">Total</th>
                    </tr>
                </thead>
                <tbody>
                    {(grid.rows ?? []).map((row, ri) => (
                        <tr key={row.taskId} className="border-b border-gray-100 dark:border-gray-800">
                            <th scope="row" className="py-1 pr-4 text-left font-normal max-w-xs">
                                <div className="text-gray-900 dark:text-white truncate" title={row.taskName}>
                                    {row.taskName || `Tarefa ${row.taskId}`}
                                </div>
                                <div className="text-xs text-gray-500 dark:text-gray-400 truncate">
                                    {row.projectName}{row.saved ? ' • salva' : ''}
                                </div>
                            </th>
                            {visibleDays.map(i => {
                                const cell = row.cells?.[i];
                                if (!cell) return <td key={i}/>;
                                const temLancamentos = (cell.entries ?? []).length > 0;
                                const rotulo = `${row.taskName || row.taskId} em ${formatDateBR(cell.date, 'dd/MM')}`;
                                return (
                                    <td key={cell.date} className={clsx('py-1 px-1 text-center', COLUNA_STATUS[statusDoDia(i)])}>
                                        <div className="flex items-center justify-center gap-0.5">
                                            <CellInput
                                                key={`${cell.date}-${cell.minutes}`}
                                                minutes={cell.minutes}
                                                label={`Horas de ${rotulo}`}
                                                disabled={disabled || row.taskId <= 0}
                                                onCommit={target => onCellCommit(row, i, target)}
                                            />
                                            {temLancamentos && (
                                                <button
                                                    type="button"
                                                    onClick={() => onOpenCell(row, i)}
                                                    aria-label={`Ver lançamentos de ${rotulo}`}
                                                    title="Ver, editar ou apagar lançamentos"
                                                    className="p-1 rounded text-gray-400 hover:text-primary-600 hover:bg-gray-100 dark:hover:bg-gray-700"
                                                >
                                                    <FiList className="w-3.5 h-3.5" aria-hidden="true"/>
                                                </button>
                                            )}
                                        </div>
                                    </td>
                                );
                            })}
                            <td className="py-1 pl-2 text-right tabular-nums font-medium text-gray-900 dark:text-white">
                                {formatHHMM(totals.perRow[ri] ?? 0)}
                            </td>
                        </tr>
                    ))}
                </tbody>
                <tfoot>
                    <tr className="border-t-2 border-gray-200 dark:border-gray-700">
                        <th scope="row" className="py-2 pr-4 text-left font-medium text-gray-700 dark:text-gray-300">Total do dia</th>
                        {visibleDays.map(i => {
                            const status = statusDoDia(i);
                            return (
                                <td
                                    key={i}
                                    className={clsx('py-2 px-1 text-center tabular-nums font-semibold', COLUNA_STATUS[status],
                                        status === 'below' && 'text-amber-700 dark:text-amber-400',
                                        status === 'complete' && 'text-green-700 dark:text-green-400')}
                                    title={status === 'below' ? 'Abaixo da jornada diária' : undefined}
                                >
                                    {formatHHMM(totals.perDay[i] ?? 0)}
                                </td>
                            );
                        })}
                        <td className="py-2 pl-2 text-right tabular-nums font-semibold text-gray-900 dark:text-white">
                            {formatHHMM(totals.total)}
                        </td>
                    </tr>
                </tfoot>
            </table>
        </div>
    );
};

export default WeekGridTable;
