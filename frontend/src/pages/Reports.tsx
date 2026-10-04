import {type ReactNode, useMemo, useState} from 'react';
import {toast} from 'react-toastify';
import {FiAlertCircle, FiBarChart2, FiDownload, FiFileText, FiLoader, FiRefreshCw} from 'react-icons/fi';
import {DownloadTimeReport, ExportTimeReportCSV, OpenDirectoryPath} from '@wailsjs/go/backend/App';
import PeriodSelector from '../components/reports/PeriodSelector';
import SummaryCards from '../components/reports/SummaryCards';
import DailyChart from '../components/reports/DailyChart';
import RankingBars, {type RankingItem} from '../components/reports/RankingBars';
import TaskTable from '../components/reports/TaskTable';
import {formatMinutes} from '../components/reports/format';
import useTimeReport from '../hooks/useTimeReport';
import {formatDateBR} from '../utils/dates';
import {errMsg} from '../utils/errors';
import {presetRange, rangeError, type ReportPreset, type ReportRange} from '../utils/reportPeriods';

type Exportacao = 'csv-detalhado' | 'csv-resumido' | 'pdf';

// Página de Relatórios: totais do período, horas por dia contra a jornada,
// rankings por projeto e tarefa e exportação (CSV para Excel e o PDF do
// Teamwork). Só lê lançamentos; nada aqui altera dados no Teamwork.
const Reports = () => {
    const [preset, setPreset] = useState<ReportPreset>('thisMonth');
    const [range, setRange] = useState<ReportRange>(() => presetRange('thisMonth') ?? {startDate: '', endDate: ''});
    const [exportando, setExportando] = useState<Exportacao | null>(null);
    const {report, loading, error, reload} = useTimeReport(range);

    const erroPeriodo = rangeError(range);

    const projetos = useMemo<RankingItem[]>(() => (report?.byProject ?? []).map(p => ({
        key: `${p.projectId}-${p.projectName}`, label: p.projectName, minutes: p.minutes
    })), [report]);

    const tarefas = useMemo<RankingItem[]>(() => (report?.byTask ?? []).map(t => ({
        key: `${t.projectId}-${t.taskId}-${t.taskName}`, label: t.taskName, sublabel: t.projectName, minutes: t.minutes
    })), [report]);

    const exportar = async (tipo: Exportacao) => {
        if (erroPeriodo) {
            toast.warning(erroPeriodo);
            return;
        }
        setExportando(tipo);
        try {
            const caminho = tipo === 'pdf'
                ? await DownloadTimeReport(range.startDate, range.endDate)
                : await ExportTimeReportCSV(range.startDate, range.endDate, tipo === 'csv-detalhado');
            toast.success(
                <div>
                    <p>Relatório salvo em:</p>
                    <p className="text-xs break-all">{caminho}</p>
                    <button type="button" className="mt-1 text-sm underline"
                            onClick={() => {
                                OpenDirectoryPath(caminho).catch((err: unknown) =>
                                    toast.error('Não foi possível abrir a pasta: ' + errMsg(err)));
                            }}>
                        Abrir pasta
                    </button>
                </div>,
                {autoClose: 10000}
            );
        } catch (err) {
            console.error('Erro ao exportar relatório:', err);
            toast.error('Não foi possível exportar o relatório: ' + errMsg(err));
        } finally {
            setExportando(null);
        }
    };

    const botaoExportar = (tipo: Exportacao, rotulo: string, icone: ReactNode) => (
        <button
            type="button"
            onClick={() => void exportar(tipo)}
            disabled={exportando !== null || Boolean(erroPeriodo)}
            className="btn-secondary !px-3 !py-2 inline-flex items-center disabled:opacity-50"
        >
            {exportando === tipo ? <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/> : icone}
            {rotulo}
        </button>
    );

    return (
        <div className="space-y-6">
            <div className="flex flex-wrap items-center justify-between gap-3">
                <h1 className="text-2xl font-semibold text-gray-900 dark:text-white flex items-center">
                    <FiBarChart2 className="w-6 h-6 mr-2" aria-hidden="true"/>
                    Relatórios
                </h1>
                <div className="flex flex-wrap gap-2">
                    {botaoExportar('csv-detalhado', 'CSV detalhado', <FiDownload className="w-4 h-4 mr-2" aria-hidden="true"/>)}
                    {botaoExportar('csv-resumido', 'CSV resumido', <FiDownload className="w-4 h-4 mr-2" aria-hidden="true"/>)}
                    {botaoExportar('pdf', 'PDF', <FiFileText className="w-4 h-4 mr-2" aria-hidden="true"/>)}
                </div>
            </div>

            <div className="card !p-4 space-y-2">
                <PeriodSelector preset={preset} range={range} error={erroPeriodo}
                                onChange={(p, r) => {
                                    setPreset(p);
                                    setRange(r);
                                }}/>
                {!erroPeriodo && (
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                        {formatDateBR(range.startDate)} a {formatDateBR(range.endDate)}
                    </p>
                )}
            </div>

            {error && (
                <div role="alert" className="flex items-start p-4 bg-red-50 dark:bg-red-900/20 border-l-4 border-red-500 rounded">
                    <FiAlertCircle className="w-5 h-5 mr-2 text-red-500 flex-shrink-0" aria-hidden="true"/>
                    <div>
                        <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
                        {!erroPeriodo && (
                            <button type="button" onClick={reload}
                                    className="mt-1 inline-flex items-center text-sm text-red-700 dark:text-red-300 hover:underline">
                                <FiRefreshCw className="w-4 h-4 mr-1" aria-hidden="true"/>Tentar novamente
                            </button>
                        )}
                    </div>
                </div>
            )}

            {loading && !report && (
                <div className="flex justify-center py-12" role="status" aria-label="Carregando relatório">
                    <FiLoader className="w-8 h-8 animate-spin text-primary-600" aria-hidden="true"/>
                </div>
            )}

            {report && (
                <div className={`space-y-6 ${loading ? 'opacity-60' : ''}`} aria-busy={loading}>
                    <SummaryCards report={report}/>

                    <section className="card !p-4" aria-labelledby="horas-por-dia">
                        <h2 id="horas-por-dia" className="text-sm font-semibold text-gray-900 dark:text-white mb-1">
                            Horas por dia
                        </h2>
                        <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
                            Média por dia útil: {formatMinutes(report.workingDays > 0 ? Math.round(report.totalMinutes / report.workingDays) : 0)}
                        </p>
                        <DailyChart days={report.byDay ?? []}/>
                    </section>

                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                        <RankingBars title="Por projeto" items={projetos} totalMinutes={report.totalMinutes}/>
                        <RankingBars title="Por tarefa" items={tarefas} totalMinutes={report.totalMinutes}/>
                    </div>

                    {(report.byWeek ?? []).length > 1 && (
                        <section className="card !p-4" aria-labelledby="por-semana">
                            <h2 id="por-semana" className="text-sm font-semibold text-gray-900 dark:text-white mb-3">Por semana</h2>
                            <div className="overflow-x-auto">
                                <table className="w-full text-sm text-left text-gray-700 dark:text-gray-300">
                                    <thead className="text-xs uppercase text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-700">
                                    <tr>
                                        <th scope="col" className="py-2 px-2">Semana</th>
                                        <th scope="col" className="py-2 px-2 text-right">Lançado</th>
                                        <th scope="col" className="py-2 px-2 text-right">Esperado</th>
                                        <th scope="col" className="py-2 px-2 text-right">Dias úteis</th>
                                    </tr>
                                    </thead>
                                    <tbody>
                                    {(report.byWeek ?? []).map(w => (
                                        <tr key={w.weekStart} className="border-b border-gray-100 dark:border-gray-700">
                                            <td className="py-1.5 px-2">{formatDateBR(w.weekStart, 'dd/MM')} a {formatDateBR(w.weekEnd, 'dd/MM')}</td>
                                            <td className="py-1.5 px-2 text-right tabular-nums">{formatMinutes(w.minutes)}</td>
                                            <td className="py-1.5 px-2 text-right tabular-nums">{formatMinutes(w.expectedMinutes)}</td>
                                            <td className="py-1.5 px-2 text-right tabular-nums">{w.workingDays}</td>
                                        </tr>
                                    ))}
                                    </tbody>
                                </table>
                            </div>
                        </section>
                    )}

                    <section className="card !p-4" aria-labelledby="por-tarefa">
                        <h2 id="por-tarefa" className="text-sm font-semibold text-gray-900 dark:text-white mb-3">Detalhe por tarefa</h2>
                        <TaskTable tasks={report.byTask ?? []} totalMinutes={report.totalMinutes}/>
                    </section>
                </div>
            )}
        </div>
    );
};

export default Reports;
