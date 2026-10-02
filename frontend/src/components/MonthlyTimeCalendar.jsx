import React, {forwardRef, useEffect, useImperativeHandle, useMemo, useState} from 'react';
import {
    FiAlertCircle,
    FiCalendar,
    FiCheckCircle,
    FiChevronLeft,
    FiChevronRight,
    FiClock,
    FiRefreshCw,
    FiX
} from 'react-icons/fi';
import {toast} from 'react-toastify';
import {addMonths, eachDayOfInterval, endOfMonth, format, startOfMonth} from 'date-fns';
import {ptBR} from 'date-fns/locale';
import {
    GetAllNonWorkingDays,
    GetLoggedTimeFromCalendarAPI,
    GetTimeEntriesForPeriod
} from '../../wailsjs/go/backend/App';
import Modal from './Modal';
import useMinutosPorDia from '../hooks/useMinutosPorDia';
import {toYMD, utcTimestampToYMD} from '../utils/dates';
import {errMsg} from '../utils/errors';
import {DIAS_ABREV, formatHoursMinutes} from '../utils/time';

const MONTH_NAMES = [
    'Janeiro', 'Fevereiro', 'Março', 'Abril', 'Maio', 'Junho',
    'Julho', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Dezembro'
];

const STATUS_LABEL = {
    complete: 'Completo',
    incomplete: 'Incompleto',
    missing: 'Sem registros',
    holiday: 'Feriado',
    weekend: 'Fim de semana'
};

const isWeekend = (date) => {
    const day = date.getDay();
    return day === 0 || day === 6;
};

const getDayStatusClass = (status) => {
    switch (status) {
        case 'complete':
            return 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 border-green-300 dark:border-green-700';
        case 'incomplete':
            return 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300 border-yellow-300 dark:border-yellow-700';
        case 'missing':
            return 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 border-red-300 dark:border-red-700';
        case 'holiday':
            return 'bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300 border-purple-300 dark:border-purple-700';
        case 'weekend':
            return 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400 border-gray-300 dark:border-gray-700';
        default:
            return 'bg-white dark:bg-gray-800 text-gray-800 dark:text-gray-200 border-gray-300 dark:border-gray-700';
    }
};

// A API de calendário devolve pares [timestampUTC, horas, minutos] por dia,
// separados em cobráveis e não cobráveis. Consolida tudo por data.
const processCalendarData = (loggedTimeData) => {
    const porDia = {};

    const adicionar = (lista, isBillable) => {
        if (!Array.isArray(lista)) return;
        lista.forEach(entry => {
            if (!entry || entry.length < 3) return;
            const timestamp = parseInt(entry[0], 10);
            const hours = parseFloat(entry[1]) || 0;
            const minutes = parseInt(entry[2], 10) || 0;
            if (Number.isNaN(timestamp) || minutes <= 0) return;

            const date = utcTimestampToYMD(timestamp);
            if (!date) return;

            const item = {
                date,
                minutes,
                hours,
                description: isBillable ? 'Tempo registrado (cobrável)' : 'Tempo registrado (não cobrável)',
                projectName: 'Teamwork',
                isBillable
            };

            if (!porDia[date]) {
                porDia[date] = {...item, isConsolidated: false, entries: [item]};
            } else {
                porDia[date].minutes += minutes;
                porDia[date].hours += hours;
                porDia[date].isConsolidated = true;
                porDia[date].description = 'Consolidado (múltiplas entradas)';
                porDia[date].entries.push(item);
            }
        });
    };

    adicionar(loggedTimeData.user.billable, true);
    adicionar(loggedTimeData.user.nonbillable, false);

    return Object.values(porDia);
};

// Busca o tempo do mês: primeiro a API de calendário (agregada por dia) e, se
// ela não estiver disponível, as entradas detalhadas do período. Se as duas
// falharem, o erro sobe — não inventamos horas estimadas.
const fetchMonthEntries = async (month) => {
    const startDate = toYMD(startOfMonth(month));
    const endDate = toYMD(endOfMonth(month));

    let calendarError = null;
    try {
        const loggedTimeData = await GetLoggedTimeFromCalendarAPI(month.getMonth() + 1, month.getFullYear());
        if (loggedTimeData && loggedTimeData.STATUS === 'OK' && loggedTimeData.user) {
            return processCalendarData(loggedTimeData);
        }
    } catch (error) {
        calendarError = error;
    }

    try {
        const timeEntries = await GetTimeEntriesForPeriod(startDate, endDate);
        return (timeEntries || []).map(entry => ({
            date: entry.date,
            minutes: entry.minutes || 0,
            hours: (entry.minutes || 0) / 60,
            description: entry.description || 'Tempo registrado',
            projectName: entry.projectName || 'Teamwork',
            isBillable: entry.isBillable !== undefined ? entry.isBillable : true
        }));
    } catch (entriesError) {
        console.error('Erro ao obter o tempo do mês:', calendarError, entriesError);
        throw entriesError;
    }
};

// onDayClick(dataYMD) é chamado ao clicar num dia útil.
const MonthlyTimeCalendar = forwardRef(({onDayClick}, ref) => {
    const dailyGoal = useMinutosPorDia();
    const [currentMonth, setCurrentMonth] = useState(() => startOfMonth(new Date()));
    const [timeEntries, setTimeEntries] = useState([]);
    const [holidays, setHolidays] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [reloadToken, setReloadToken] = useState(0);
    const [dayDetails, setDayDetails] = useState(null);

    // refresh recarrega o mês exibido, sem voltar ao mês atual.
    useImperativeHandle(ref, () => ({
        refresh: () => setReloadToken(t => t + 1)
    }), []);

    useEffect(() => {
        // Ao trocar de mês rápido, a resposta do mês anterior pode chegar por
        // último; o flag descarta respostas de efeitos já substituídos.
        let cancelled = false;

        const load = async () => {
            setLoading(true);
            setError(null);

            const [entriesResult, holidaysResult] = await Promise.allSettled([
                fetchMonthEntries(currentMonth),
                GetAllNonWorkingDays(currentMonth.getFullYear(), currentMonth.getMonth() + 1)
            ]);

            if (cancelled) return;

            if (entriesResult.status === 'fulfilled') {
                setTimeEntries(entriesResult.value);
            } else {
                setTimeEntries([]);
                setError(`Não foi possível carregar as horas do mês: ${errMsg(entriesResult.reason)}`);
            }

            if (holidaysResult.status === 'fulfilled') {
                setHolidays((holidaysResult.value || []).filter(day => day.type === 'holiday'));
            } else {
                console.error('Erro ao carregar feriados:', holidaysResult.reason);
                setHolidays([]);
            }

            setLoading(false);
        };

        load();
        return () => {
            cancelled = true;
        };
    }, [currentMonth, reloadToken]);

    // Índices por data calculados uma vez por carga, em vez de filtrar todas as
    // entradas para cada célula do mês.
    const entriesByDate = useMemo(() => {
        const map = new Map();
        timeEntries.forEach(entry => {
            const lista = map.get(entry.date);
            if (lista) lista.push(entry);
            else map.set(entry.date, [entry]);
        });
        return map;
    }, [timeEntries]);

    const minutesByDate = useMemo(() => {
        const map = new Map();
        entriesByDate.forEach((lista, date) => {
            map.set(date, lista.reduce((total, entry) => total + (entry.minutes || 0), 0));
        });
        return map;
    }, [entriesByDate]);

    const holidaysByDate = useMemo(() => {
        const map = new Map();
        holidays.forEach(h => map.set(h.date, h));
        return map;
    }, [holidays]);

    const getDayStatus = (day, dayStr) => {
        if (isWeekend(day)) return 'weekend';
        if (holidaysByDate.has(dayStr)) return 'holiday';

        const minutes = minutesByDate.get(dayStr) || 0;
        if (minutes === 0) return 'missing';
        if (minutes < dailyGoal) return 'incomplete';
        return 'complete';
    };

    const handleDayClick = (day) => {
        if (isWeekend(day)) return;

        const dayStr = toYMD(day);
        const holiday = holidaysByDate.get(dayStr);
        if (holiday) {
            toast.info(`Feriado: ${holiday.name}. Não é possível lançar horas em feriados.`);
            return;
        }

        const dayEntries = entriesByDate.get(dayStr) || [];
        const minutesLogged = minutesByDate.get(dayStr) || 0;

        setDayDetails({
            date: day,
            dateStr: dayStr,
            entries: dayEntries,
            totalMinutes: minutesLogged,
            totalHours: (minutesLogged / 60).toFixed(1),
            status: getDayStatus(day, dayStr)
        });

        if (onDayClick) {
            onDayClick(dayStr);
        }
    };

    const closeModal = () => setDayDetails(null);

    const todayStr = toYMD(new Date());

    const renderDayCell = (day) => {
        const dayStr = toYMD(day);
        const dayNum = day.getDate();
        const isToday = dayStr === todayStr;
        const status = getDayStatus(day, dayStr);
        const statusClass = getDayStatusClass(status);
        const minutes = minutesByDate.get(dayStr) || 0;
        const hours = (minutes / 60).toFixed(1);
        const isSelected = dayDetails?.dateStr === dayStr;
        const holiday = holidaysByDate.get(dayStr);
        const weekend = status === 'weekend';

        const descricao = [
            format(day, "EEEE, d 'de' MMMM", {locale: ptBR}),
            holiday ? `Feriado: ${holiday.name}` : STATUS_LABEL[status],
            !weekend && !holiday ? `${hours} horas registradas` : null
        ].filter(Boolean).join('. ');

        const className = `relative p-2 border text-left ${statusClass} ${isToday ? 'ring-2 ring-primary-500 dark:ring-primary-400' : ''}
                ${isSelected ? 'ring-2 ring-blue-500 dark:ring-blue-400' : ''}
                ${!weekend && !holiday ? 'cursor-pointer hover:shadow-md' : ''} rounded-md h-20 flex flex-col w-full focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-600`;

        const conteudo = (
            <>
                <span className="text-sm font-medium mb-1" aria-hidden="true">
                    {dayNum}
                </span>

                {holiday && (
                    <span className="text-xs text-purple-700 dark:text-purple-300 mt-auto overflow-hidden text-ellipsis" aria-hidden="true">
                        <FiCalendar className="inline mr-1 w-3 h-3"/>
                        <span className="whitespace-nowrap overflow-hidden text-ellipsis">
                            {holiday.name}
                        </span>
                    </span>
                )}

                {!weekend && !holiday && (
                    <>
                        <span className="text-xs mt-auto" aria-hidden="true">
                            {minutes > 0 ? (
                                <span className="flex items-center">
                                    <FiClock className="mr-1 w-3 h-3"/>
                                    <span>{hours}h</span>
                                </span>
                            ) : (
                                <span className="text-gray-400 dark:text-gray-500">Sem registros</span>
                            )}
                        </span>

                        <span className="absolute top-1 right-1" aria-hidden="true">
                            {status === 'complete' && <FiCheckCircle className="w-4 h-4 text-green-500 dark:text-green-400"/>}
                            {status === 'incomplete' && <FiAlertCircle className="w-4 h-4 text-yellow-500 dark:text-yellow-400"/>}
                            {status === 'missing' && <FiX className="w-4 h-4 text-red-500 dark:text-red-400"/>}
                        </span>
                    </>
                )}
            </>
        );

        if (weekend) {
            return (
                <div key={dayStr} className={className} role="gridcell" aria-label={descricao}>
                    {conteudo}
                </div>
            );
        }

        return (
            <div key={dayStr} role="gridcell">
                <button
                    type="button"
                    onClick={() => handleDayClick(day)}
                    className={className}
                    aria-label={descricao}
                    aria-current={isToday ? 'date' : undefined}
                >
                    {conteudo}
                </button>
            </div>
        );
    };

    const days = eachDayOfInterval({start: startOfMonth(currentMonth), end: endOfMonth(currentMonth)});
    const cells = [
        ...Array.from({length: startOfMonth(currentMonth).getDay()}, () => null),
        ...days
    ];
    while (cells.length % 7 !== 0) cells.push(null);

    const weeks = [];
    for (let i = 0; i < cells.length; i += 7) {
        weeks.push(cells.slice(i, i + 7));
    }

    const metaLabel = formatHoursMinutes(dailyGoal);
    const monthTitle = `${MONTH_NAMES[currentMonth.getMonth()]} ${currentMonth.getFullYear()}`;

    return (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md p-4 mb-6">
            <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-white" aria-live="polite">
                    {monthTitle}
                </h2>

                <div className="flex space-x-2">
                    <button
                        type="button"
                        onClick={() => setCurrentMonth(m => addMonths(m, -1))}
                        aria-label="Mês anterior"
                        title="Mês anterior"
                        className="p-1 rounded-full hover:bg-gray-100 dark:hover:bg-gray-700"
                    >
                        <FiChevronLeft className="w-5 h-5 text-gray-600 dark:text-gray-400" aria-hidden="true"/>
                    </button>
                    <button
                        type="button"
                        onClick={() => setCurrentMonth(startOfMonth(new Date()))}
                        className="px-2 py-1 text-xs text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                    >
                        Hoje
                    </button>
                    <button
                        type="button"
                        onClick={() => setCurrentMonth(m => addMonths(m, 1))}
                        aria-label="Próximo mês"
                        title="Próximo mês"
                        className="p-1 rounded-full hover:bg-gray-100 dark:hover:bg-gray-700"
                    >
                        <FiChevronRight className="w-5 h-5 text-gray-600 dark:text-gray-400" aria-hidden="true"/>
                    </button>
                </div>
            </div>

            <div className="flex flex-wrap gap-2 mb-4">
                <div className="flex items-center">
                    <div className="w-3 h-3 bg-green-500 dark:bg-green-400 rounded-full mr-1"></div>
                    <span className="text-xs text-gray-600 dark:text-gray-400">Completo ({metaLabel}+)</span>
                </div>
                <div className="flex items-center">
                    <div className="w-3 h-3 bg-yellow-500 dark:bg-yellow-400 rounded-full mr-1"></div>
                    <span className="text-xs text-gray-600 dark:text-gray-400">Incompleto (&lt;{metaLabel})</span>
                </div>
                <div className="flex items-center">
                    <div className="w-3 h-3 bg-red-500 dark:bg-red-400 rounded-full mr-1"></div>
                    <span className="text-xs text-gray-600 dark:text-gray-400">Sem Registro</span>
                </div>
                <div className="flex items-center">
                    <div className="w-3 h-3 bg-purple-500 dark:bg-purple-400 rounded-full mr-1"></div>
                    <span className="text-xs text-gray-600 dark:text-gray-400">Feriado</span>
                </div>
                <div className="flex items-center">
                    <div className="w-3 h-3 bg-gray-400 dark:bg-gray-500 rounded-full mr-1"></div>
                    <span className="text-xs text-gray-600 dark:text-gray-400">Fim de semana</span>
                </div>
            </div>

            {loading && (
                <div className="flex justify-center items-center py-8" role="status" aria-label="Carregando calendário">
                    <div className="animate-spin w-6 h-6 border-2 border-primary-600 border-t-transparent rounded-full"></div>
                </div>
            )}

            {error && !loading && (
                <div role="alert" className="bg-red-50 dark:bg-red-900/20 border-l-4 border-red-500 p-4 mb-4">
                    <div className="flex items-start">
                        <FiAlertCircle className="h-5 w-5 text-red-500 dark:text-red-400 flex-shrink-0" aria-hidden="true"/>
                        <div className="ml-3 flex-1">
                            <p className="text-sm text-red-700 dark:text-red-200">{error}</p>
                            <button
                                type="button"
                                onClick={() => setReloadToken(t => t + 1)}
                                className="mt-2 inline-flex items-center text-sm text-red-700 dark:text-red-300 hover:underline"
                            >
                                <FiRefreshCw className="w-4 h-4 mr-1" aria-hidden="true"/>
                                Tentar novamente
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {!loading && !error && (
                <div role="grid" aria-label={`Calendário de ${monthTitle}`}>
                    <div role="row" className="grid grid-cols-7 gap-1 mb-1">
                        {DIAS_ABREV.map((day) => (
                            <div key={day} role="columnheader"
                                 className="p-1 text-center text-xs font-medium text-gray-500 dark:text-gray-400">
                                {day}
                            </div>
                        ))}
                    </div>

                    {weeks.map((week, weekIndex) => (
                        <div key={`semana-${weekIndex}`} role="row" className="grid grid-cols-7 gap-1 mb-1">
                            {week.map((day, index) => (
                                day ? renderDayCell(day) : (
                                    <div key={`vazio-${weekIndex}-${index}`} role="gridcell"
                                         className="border border-gray-200 dark:border-gray-700 rounded-md h-20"></div>
                                )
                            ))}
                        </div>
                    ))}
                </div>
            )}

            <Modal
                isOpen={Boolean(dayDetails)}
                onClose={closeModal}
                size="sm"
                title={dayDetails ? format(dayDetails.date, "EEEE, d 'de' MMMM", {locale: ptBR}) : ''}
                footer={
                    <button
                        type="button"
                        onClick={closeModal}
                        className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md"
                    >
                        Fechar
                    </button>
                }
            >
                {dayDetails && (
                    <>
                        <div className={`mb-4 p-3 rounded-lg ${getDayStatusClass(dayDetails.status)} border`}>
                            <div className="flex items-center justify-between">
                                <div className="flex items-center">
                                    {dayDetails.status === 'complete' && <FiCheckCircle className="w-5 h-5 mr-2" aria-hidden="true"/>}
                                    {dayDetails.status === 'incomplete' && <FiAlertCircle className="w-5 h-5 mr-2" aria-hidden="true"/>}
                                    {dayDetails.status === 'missing' && <FiX className="w-5 h-5 mr-2" aria-hidden="true"/>}
                                    <span className="font-medium">{STATUS_LABEL[dayDetails.status]}</span>
                                </div>
                                <div>
                                    <FiClock className="w-4 h-4 inline mr-1" aria-hidden="true"/>
                                    <span>{dayDetails.totalHours}h de {metaLabel}</span>
                                </div>
                            </div>
                        </div>

                        {dayDetails.entries.length > 0 ? (
                            <div className="space-y-3 max-h-60 overflow-y-auto">
                                <h3 className="font-medium text-gray-900 dark:text-white mb-2">Lançamentos</h3>
                                {dayDetails.entries.map((entry, idx) => (
                                    <div key={`${entry.date}-${idx}`} className="p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                                        <div className="flex justify-between mb-1">
                                            <span className="font-medium text-sm">{entry.description || 'Sem descrição'}</span>
                                            <span className="text-sm">{(entry.minutes / 60).toFixed(1)}h</span>
                                        </div>
                                        <div className="text-xs text-gray-500 dark:text-gray-400">
                                            <div>Projeto: {entry.projectName || 'N/A'}</div>
                                            <div>Cobrável: {entry.isBillable ? 'Sim' : 'Não'}</div>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        ) : (
                            <div className="text-center py-6">
                                <p className="text-gray-500 dark:text-gray-400">Nenhum lançamento para este dia.</p>
                            </div>
                        )}
                    </>
                )}
            </Modal>
        </div>
    );
});

MonthlyTimeCalendar.displayName = 'MonthlyTimeCalendar';

export default MonthlyTimeCalendar;
