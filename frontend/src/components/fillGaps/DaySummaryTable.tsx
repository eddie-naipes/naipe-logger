import {FiBarChart2} from 'react-icons/fi';
import clsx from 'clsx';
import type {planning} from '@wailsjs/go/models';
import type {Dados} from '../../types/backend';
import {formatHoursMinutes} from '../../utils/time';

export type DaySummary = Dados<planning.DaySummary>;

const CAUSAS: Record<string, string> = {
    'completo': 'Completo',
    'futuro': 'Futuro (use "mês inteiro")',
    'abaixo da granularidade': 'Falta pouco (abaixo da granularidade)',
    'sem tarefa para o dia': 'Nenhuma tarefa para este dia da semana'
};

interface DaySummaryTableProps {
    days: readonly DaySummary[];
    minutesPerDay: number;
    formatDate: (date: string) => string;
}

// Resumo por dia útil do "Completar período": quanto já existe, quanto falta e
// quanto o plano vai lançar.
const DaySummaryTable = ({days, minutesPerDay, formatDate}: DaySummaryTableProps) => {
    if (days.length === 0) return null;

    const totalFaltante = days.reduce((s, d) => s + d.missing, 0);
    const totalALancar = days.reduce((s, d) => s + d.toLog, 0);

    return (
        <div className="card mb-6">
            <div className="flex flex-wrap items-center justify-between gap-2 mb-4">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center">
                    <FiBarChart2 className="w-5 h-5 mr-2" aria-hidden="true"/>
                    Resumo por dia
                </h2>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                    Jornada: <strong>{formatHoursMinutes(minutesPerDay)}</strong> •
                    Faltam <strong>{formatHoursMinutes(totalFaltante)}</strong> •
                    A lançar <strong>{formatHoursMinutes(totalALancar)}</strong>
                </p>
            </div>

            <div className="overflow-x-auto max-h-96 overflow-y-auto">
                <table className="min-w-full text-sm">
                    <thead className="text-left text-gray-600 dark:text-gray-400 border-b border-gray-200 dark:border-gray-700">
                        <tr>
                            <th scope="col" className="py-2 pr-4 font-medium">Dia</th>
                            <th scope="col" className="py-2 pr-4 font-medium text-right">Lançado</th>
                            <th scope="col" className="py-2 pr-4 font-medium text-right">Faltante</th>
                            <th scope="col" className="py-2 pr-4 font-medium text-right">A lançar</th>
                            <th scope="col" className="py-2 font-medium">Situação</th>
                        </tr>
                    </thead>
                    <tbody>
                        {days.map(day => (
                            <tr
                                key={day.date}
                                className={clsx(
                                    'border-b border-gray-100 dark:border-gray-800',
                                    day.toLog > 0 && 'bg-primary-50/50 dark:bg-primary-900/10'
                                )}
                            >
                                <td className="py-2 pr-4 text-gray-900 dark:text-white">{formatDate(day.date)}</td>
                                <td className="py-2 pr-4 text-right tabular-nums">{formatHoursMinutes(day.logged)}</td>
                                <td className={clsx(
                                    'py-2 pr-4 text-right tabular-nums',
                                    day.missing > 0 && !day.future && 'text-amber-600 dark:text-amber-400 font-medium'
                                )}>
                                    {formatHoursMinutes(day.missing)}
                                </td>
                                <td className="py-2 pr-4 text-right tabular-nums font-medium">
                                    {day.toLog > 0 ? formatHoursMinutes(day.toLog) : '—'}
                                </td>
                                <td className="py-2 text-gray-600 dark:text-gray-400">
                                    {day.skipped
                                        ? (CAUSAS[day.skipCause ?? ''] ?? day.skipCause)
                                        : day.toLog < day.missing
                                            ? 'Parcial: as tarefas escolhidas não cobrem tudo'
                                            : 'Será completado'}
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
};

export default DaySummaryTable;
