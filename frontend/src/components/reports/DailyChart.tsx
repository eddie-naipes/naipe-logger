import {useState} from 'react';
import {ptBR} from 'date-fns/locale';
import type {DayTotal} from '../../types/backend';
import {formatDateBR} from '../../utils/dates';
import {formatMinutes, GRADE, niceMax, REFERENCIA, SERIE_1} from './format';

interface DailyChartProps {
    days: readonly DayTotal[];
}

// Colunas de horas por dia com a jornada esperada como linha de referência
// em degraus (zero nos dias não úteis). Gráfico em CSS puro: barras de até
// 24px com topo arredondado, grade fina e recessiva, tooltip ao passar o
// mouse ou focar a coluna, e tabela com os mesmos dados para leitores de tela.
const DailyChart = ({days}: DailyChartProps) => {
    const [ativo, setAtivo] = useState<string | null>(null);
    const [mostrarTabela, setMostrarTabela] = useState(false);

    if (days.length === 0) return null;

    const max = niceMax(Math.max(...days.map(d => Math.max(d.minutes, d.expectedMinutes))));
    const passos = 4;
    const ticks = Array.from({length: passos + 1}, (_, i) => Math.round((max / passos) * i));
    const pct = (minutes: number) => `${Math.min(100, (minutes / max) * 100)}%`;
    // Rótulos do eixo X só a cada N dias para não colidirem.
    const cadaN = days.length <= 16 ? 1 : days.length <= 35 ? 3 : Math.ceil(days.length / 12);
    const diaAtivo = days.find(d => d.date === ativo);

    return (
        <figure className="m-0">
            <div className="flex items-center gap-4 mb-3 text-xs text-gray-600 dark:text-gray-400" aria-hidden="true">
                <span className="flex items-center"><span className={`inline-block w-3 h-3 rounded-xs mr-1 ${SERIE_1}`}/>Horas lançadas</span>
                <span className="flex items-center"><span className={`inline-block w-4 h-0.5 mr-1 ${REFERENCIA}`}/>Jornada esperada</span>
                <span className="flex items-center"><span className="inline-block w-3 h-3 rounded-xs mr-1 bg-gray-100 dark:bg-gray-700/60"/>Dia não útil</span>
            </div>

            <div className="flex">
                {/* Eixo Y */}
                <div className="relative w-10 h-56 mr-1 text-[10px] text-gray-500 dark:text-gray-400" aria-hidden="true">
                    {ticks.map(t => (
                        <span key={t} className="absolute right-0 -translate-y-1/2" style={{bottom: pct(t)}}>
                            {formatMinutes(t)}
                        </span>
                    ))}
                </div>

                <div className="relative flex-1 min-w-0">
                    <div className="relative h-56">
                        {ticks.map(t => (
                            <div key={t} className={`absolute left-0 right-0 h-px ${GRADE}`} style={{bottom: pct(t)}} aria-hidden="true"/>
                        ))}

                        <div className="absolute inset-0 flex" role="list" aria-label="Horas por dia">
                            {days.map(d => {
                                const rotulo = `${formatDateBR(d.date, "EEEE, dd/MM", d.date, {locale: ptBR})}: ${formatMinutes(d.minutes)} lançadas` +
                                    (d.isWorkingDay ? `, jornada ${formatMinutes(d.expectedMinutes)}` : ', dia não útil');
                                return (
                                    <div key={d.date} role="listitem" className="flex-1 h-full">
                                    <button
                                        type="button"
                                        aria-label={rotulo}
                                        onMouseEnter={() => setAtivo(d.date)}
                                        onMouseLeave={() => setAtivo(a => (a === d.date ? null : a))}
                                        onFocus={() => setAtivo(d.date)}
                                        onBlur={() => setAtivo(a => (a === d.date ? null : a))}
                                        className={`relative block w-full h-full cursor-default outline-hidden focus-visible:ring-2 focus-visible:ring-primary-500 ${d.isWorkingDay ? '' : 'bg-gray-100 dark:bg-gray-700/60'} ${ativo === d.date ? 'bg-gray-50 dark:bg-gray-700' : ''}`}
                                    >
                                        {d.minutes > 0 && (
                                            <div
                                                className={`absolute bottom-0 left-1/2 -translate-x-1/2 rounded-t-[4px] ${SERIE_1}`}
                                                style={{height: pct(d.minutes), width: 'min(24px, 70%)'}}
                                            />
                                        )}
                                        {d.expectedMinutes > 0 && (
                                            <div className={`absolute left-0 right-0 h-0.5 ${REFERENCIA}`}
                                                 style={{bottom: `calc(${pct(d.expectedMinutes)} - 1px)`}}/>
                                        )}
                                    </button>
                                    </div>
                                );
                            })}
                        </div>

                        {diaAtivo && (
                            <div
                                role="status"
                                className="pointer-events-none absolute top-1 right-1 z-10 rounded-md border border-gray-200 bg-white px-3 py-2 text-xs shadow-lg dark:border-gray-600 dark:bg-gray-900"
                            >
                                <p className="font-medium text-gray-900 dark:text-white">
                                    {formatDateBR(diaAtivo.date, "EEEE, dd/MM/yyyy", diaAtivo.date, {locale: ptBR})}
                                </p>
                                <p className="text-gray-700 dark:text-gray-300">Lançado: {formatMinutes(diaAtivo.minutes)}</p>
                                <p className="text-gray-700 dark:text-gray-300">Cobrável: {formatMinutes(diaAtivo.billableMinutes)}</p>
                                <p className="text-gray-500 dark:text-gray-400">
                                    {diaAtivo.isWorkingDay ? `Jornada: ${formatMinutes(diaAtivo.expectedMinutes)}` : 'Dia não útil'}
                                </p>
                            </div>
                        )}
                    </div>

                    {/* Eixo X */}
                    <div className="flex mt-1 text-[10px] text-gray-500 dark:text-gray-400" aria-hidden="true">
                        {days.map((d, i) => (
                            <span key={d.date} className="flex-1 text-center overflow-visible whitespace-nowrap">
                                {i % cadaN === 0 ? formatDateBR(d.date, 'dd/MM') : ''}
                            </span>
                        ))}
                    </div>
                </div>
            </div>

            <figcaption className="mt-2">
                <button
                    type="button"
                    onClick={() => setMostrarTabela(v => !v)}
                    className="text-xs text-primary-700 dark:text-primary-400 hover:underline"
                    aria-expanded={mostrarTabela}
                >
                    {mostrarTabela ? 'Ocultar tabela por dia' : 'Ver dados por dia em tabela'}
                </button>
            </figcaption>

            {mostrarTabela && (
                <div className="mt-2 max-h-64 overflow-y-auto">
                    <table className="w-full text-xs text-left text-gray-700 dark:text-gray-300">
                        <thead className="text-gray-500 dark:text-gray-400">
                        <tr>
                            <th scope="col" className="py-1 pr-2">Dia</th>
                            <th scope="col" className="py-1 pr-2 text-right">Lançado</th>
                            <th scope="col" className="py-1 pr-2 text-right">Cobrável</th>
                            <th scope="col" className="py-1 text-right">Jornada</th>
                        </tr>
                        </thead>
                        <tbody>
                        {days.map(d => (
                            <tr key={d.date} className="border-t border-gray-100 dark:border-gray-700">
                                <td className="py-1 pr-2">{formatDateBR(d.date, 'EEE dd/MM', d.date, {locale: ptBR})}</td>
                                <td className="py-1 pr-2 text-right tabular-nums">{formatMinutes(d.minutes)}</td>
                                <td className="py-1 pr-2 text-right tabular-nums">{formatMinutes(d.billableMinutes)}</td>
                                <td className="py-1 text-right tabular-nums">{d.isWorkingDay ? formatMinutes(d.expectedMinutes) : 'não útil'}</td>
                            </tr>
                        ))}
                        </tbody>
                    </table>
                </div>
            )}
        </figure>
    );
};

export default DailyChart;
