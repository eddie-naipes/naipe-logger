import {FiAlertTriangle, FiCheckCircle} from 'react-icons/fi';
import type {TimeReport} from '../../types/backend';
import {formatMinutes, formatSigned, percent, SERIE_1, SERIE_2} from './format';

interface SummaryCardsProps {
    report: TimeReport;
}

// Cartões de totais. O total é o número de destaque; cobrável × não cobrável
// vira uma barra de proporção com legenda; o saldo contra a jornada usa cor
// de status sempre acompanhada de ícone e texto.
const SummaryCards = ({report}: SummaryCardsProps) => {
    const pctCobravel = percent(report.billableMinutes, report.totalMinutes);
    const saldoOk = report.balanceMinutes >= 0;

    return (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
            <div className="card !p-4">
                <p className="text-sm text-gray-600 dark:text-gray-400">Total lançado</p>
                <p className="text-4xl font-semibold text-gray-900 dark:text-white tabular-nums" data-testid="total-lancado">
                    {formatMinutes(report.totalMinutes)}
                </p>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {report.entryCount} lançamentos em {report.daysWithEntries} dias
                </p>
            </div>

            <div className="card !p-4">
                <p className="text-sm text-gray-600 dark:text-gray-400">Cobrável × não cobrável</p>
                <p className="text-2xl font-semibold text-gray-900 dark:text-white tabular-nums">{pctCobravel}% cobrável</p>
                <div className="flex h-3 mt-2 gap-0.5" aria-hidden="true">
                    {report.billableMinutes > 0 && (
                        <div className={`h-3 rounded-l-[4px] ${report.nonBillableMinutes === 0 ? 'rounded-r-[4px]' : ''} ${SERIE_1}`}
                             style={{width: `${pctCobravel}%`}}/>
                    )}
                    {report.nonBillableMinutes > 0 && (
                        <div className={`h-3 rounded-r-[4px] ${report.billableMinutes === 0 ? 'rounded-l-[4px]' : ''} ${SERIE_2}`}
                             style={{width: `${100 - pctCobravel}%`}}/>
                    )}
                    {report.totalMinutes === 0 && <div className="h-3 w-full rounded-[4px] bg-gray-100 dark:bg-gray-700"/>}
                </div>
                <ul className="mt-2 space-y-0.5 text-xs text-gray-700 dark:text-gray-300">
                    <li className="flex items-center"><span className={`inline-block w-2.5 h-2.5 rounded-sm mr-1.5 ${SERIE_1}`} aria-hidden="true"/>
                        Cobrável: <span className="ml-1 tabular-nums">{formatMinutes(report.billableMinutes)}</span></li>
                    <li className="flex items-center"><span className={`inline-block w-2.5 h-2.5 rounded-sm mr-1.5 ${SERIE_2}`} aria-hidden="true"/>
                        Não cobrável: <span className="ml-1 tabular-nums">{formatMinutes(report.nonBillableMinutes)}</span></li>
                </ul>
            </div>

            <div className="card !p-4">
                <p className="text-sm text-gray-600 dark:text-gray-400">Jornada esperada</p>
                <p className="text-2xl font-semibold text-gray-900 dark:text-white tabular-nums">{formatMinutes(report.expectedMinutes)}</p>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {report.workingDays} dias úteis × {formatMinutes(report.minutesPerDay)}
                </p>
                {report.workingDaysWithoutEntries > 0 && (
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                        {report.workingDaysWithoutEntries} dias úteis sem lançamento
                    </p>
                )}
            </div>

            <div className="card !p-4">
                <p className="text-sm text-gray-600 dark:text-gray-400">Saldo do período</p>
                <p className="text-2xl font-semibold text-gray-900 dark:text-white tabular-nums">{formatSigned(report.balanceMinutes)}</p>
                <p className={`text-xs mt-1 flex items-center ${saldoOk ? 'text-green-700 dark:text-green-400' : 'text-amber-700 dark:text-amber-400'}`}>
                    {saldoOk
                        ? <><FiCheckCircle className="w-3.5 h-3.5 mr-1" aria-hidden="true"/>Jornada cumprida</>
                        : <><FiAlertTriangle className="w-3.5 h-3.5 mr-1" aria-hidden="true"/>Faltam {formatMinutes(-report.balanceMinutes)}</>}
                </p>
            </div>
        </div>
    );
};

export default SummaryCards;
