import React, { useState, useEffect } from 'react';
import { FiChevronLeft, FiChevronRight, FiClock, FiAlertCircle, FiCheckCircle, FiX, FiLoader } from 'react-icons/fi';

const MonthlyTimeCalendar = ({ onDayClick }) => {
    const [currentMonth, setCurrentMonth] = useState(new Date());
    const [timeEntries, setTimeEntries] = useState([]);
    const [loading, setLoading] = useState(true);
    const [selectedDay, setSelectedDay] = useState(null);
    const [dayDetails, setDayDetails] = useState(null);
    const [showModal, setShowModal] = useState(false);
    const [error, setError] = useState(null);

    // Definindo o objetivo diário como 8h (480 minutos)
    const dailyGoal = 480;

    const formatDate = (date) => {
        return date.toISOString().split('T')[0];
    };

    const isWeekend = (date) => {
        const day = date.getDay();
        return day === 0 || day === 6;
    };

    const isSameDay = (date1, date2) => {
        return formatDate(date1) === formatDate(date2);
    };

    const startOfMonth = (date) => {
        return new Date(date.getFullYear(), date.getMonth(), 1);
    };

    const endOfMonth = (date) => {
        return new Date(date.getFullYear(), date.getMonth() + 1, 0);
    };

    const addMonths = (date, months) => {
        const newDate = new Date(date);
        newDate.setMonth(newDate.getMonth() + months);
        return newDate;
    };

    const eachDayOfInterval = (start, end) => {
        const days = [];
        let current = new Date(start);

        while (current <= end) {
            days.push(new Date(current));
            current.setDate(current.getDate() + 1);
        }

        return days;
    };

    useEffect(() => {
        loadTimeEntries();
    }, [currentMonth]);

    const loadTimeEntries = async () => {
        setLoading(true);
        setError(null);
        try {
            const startDate = formatDate(startOfMonth(currentMonth));
            const endDate = formatDate(endOfMonth(currentMonth));

            console.log(`Obtendo dados de tempo para o período: ${startDate} a ${endDate}`);

            // Primeiro - Vamos pegar os dados de horas registradas do GetTimeTotalsForPeriod
            const timeReport = await window.go.backend.App.GetTimeTotalsForPeriod(startDate, endDate);

            // Vamos tentar usar o formato LoggedTimeResponse se possível
            let loggedTimeData = null;
            try {
                // Vamos tentar o endpoint do calendário
                const monthNum = currentMonth.getMonth() + 1; // JavaScript meses começam em 0
                const yearNum = currentMonth.getFullYear();

                // Primeiro tentamos usar o método específico
                // Essa será uma tentativa que pode falhar e vamos ter fallback
                loggedTimeData = await window.go.backend.App.GetLoggedTimeFromCalendarAPI(monthNum, yearNum);
                console.log('Dados do calendário obtidos com sucesso:', loggedTimeData);
            } catch (calendarErr) {
                console.log('Endpoint de calendário não disponível, usando dados alternativos:', calendarErr);
            }

            // Agora, baseado nos dados que temos, vamos criar as entradas de tempo
            let processedEntries = [];

            // Opção 1: Se temos dados do LoggedTimeResponse, usamos eles
            if (loggedTimeData && loggedTimeData.user &&
                (loggedTimeData.user.billable || loggedTimeData.user.nonbillable)) {

                // Processar entradas "billable"
                if (loggedTimeData.user.billable && Array.isArray(loggedTimeData.user.billable)) {
                    loggedTimeData.user.billable.forEach(entry => {
                        if (entry.length >= 3) {
                            const timestamp = entry[0];
                            const hours = parseFloat(entry[1]) || 0;
                            const minutes = parseInt(entry[2]) || 0;

                            if (timestamp && !isNaN(minutes) && minutes > 0) {
                                const date = new Date(parseInt(timestamp));
                                const dateStr = formatDate(date);

                                processedEntries.push({
                                    date: dateStr,
                                    minutes,
                                    hours,
                                    description: "Tempo registrado (cobrável)",
                                    projectName: "Teamwork",
                                    isBillable: true
                                });
                            }
                        }
                    });
                }

                // Processar entradas "nonbillable" se houver minutos registrados
                if (loggedTimeData.user.nonbillable && Array.isArray(loggedTimeData.user.nonbillable)) {
                    loggedTimeData.user.nonbillable.forEach(entry => {
                        if (entry.length >= 3) {
                            const timestamp = entry[0];
                            const hours = parseFloat(entry[1]) || 0;
                            const minutes = parseInt(entry[2]) || 0;

                            if (timestamp && minutes > 0) {
                                const date = new Date(parseInt(timestamp));
                                const dateStr = formatDate(date);

                                processedEntries.push({
                                    date: dateStr,
                                    minutes,
                                    hours,
                                    description: "Tempo registrado (não cobrável)",
                                    projectName: "Teamwork",
                                    isBillable: false
                                });
                            }
                        }
                    });
                }
            }
            // Opção 2: Se não temos dados LoggedTime, mas temos TimeTotal, geramos dados sintéticos
            else if (timeReport && timeReport["time-totals"] && timeReport["time-totals"].minutes > 0) {
                try {
                    // Obter dias úteis do mês para distribuição
                    const workingDays = await window.go.backend.App.GetWorkingDays(startDate, endDate);

                    if (workingDays && workingDays.length > 0) {
                        // Verificar se temos dados de GetTimeEntriesForPeriod
                        let timeEntries = [];
                        try {
                            timeEntries = await window.go.backend.App.GetTimeEntriesForPeriod(startDate, endDate);
                            if (timeEntries && timeEntries.length > 0) {
                                // Se temos entradas reais, usamos elas
                                timeEntries.forEach(entry => {
                                    processedEntries.push({
                                        date: entry.date,
                                        minutes: entry.minutes || 0,
                                        hours: (entry.minutes || 0) / 60,
                                        description: entry.description || "Tempo registrado",
                                        projectName: entry.projectName || "Teamwork",
                                        isBillable: entry.isBillable !== undefined ? entry.isBillable : true
                                    });
                                });
                            } else {
                                throw new Error("Sem entradas detalhadas disponíveis");
                            }
                        } catch (entriesErr) {
                            console.log('Sem dados detalhados, usando distribuição por dias úteis:', entriesErr);

                            // Usamos atividades recentes para ajudar na distribuição se disponíveis
                            const recentActivities = await window.go.backend.App.GetRecentActivities();
                            const minutesByDay = {};

                            // Primeiro, distribuir com base nas atividades recentes
                            if (recentActivities && recentActivities.length > 0) {
                                recentActivities.forEach(activity => {
                                    if (activity.date && activity.date >= startDate && activity.date <= endDate) {
                                        if (!minutesByDay[activity.date]) {
                                            minutesByDay[activity.date] = 0;
                                        }
                                        minutesByDay[activity.date] += activity.minutes || 0;
                                    }
                                });
                            }

                            // Calcular minutos restantes
                            const totalRecentMinutes = Object.values(minutesByDay).reduce((sum, min) => sum + min, 0);
                            const totalReportMinutes = timeReport["time-totals"].minutes;
                            const remainingMinutes = totalReportMinutes - totalRecentMinutes;

                            // Se ainda temos minutos para distribuir
                            if (remainingMinutes > 0) {
                                // Distribuir o restante igualmente pelos dias úteis
                                const minutesPerWorkDay = Math.floor(remainingMinutes / workingDays.length);

                                workingDays.forEach(workDay => {
                                    if (!minutesByDay[workDay]) {
                                        minutesByDay[workDay] = 0;
                                    }
                                    minutesByDay[workDay] += minutesPerWorkDay;
                                });
                            }

                            // Converter para o formato de entradas
                            Object.entries(minutesByDay).forEach(([date, minutes]) => {
                                if (minutes > 0) {
                                    processedEntries.push({
                                        date,
                                        minutes,
                                        hours: minutes / 60,
                                        description: "Tempo registrado",
                                        projectName: "Teamwork",
                                        isBillable: true
                                    });
                                }
                            });
                        }
                    }
                } catch (workDaysErr) {
                    console.error('Erro ao obter dias úteis:', workDaysErr);

                    // Sem dias úteis, distribuímos igualmente pelo mês
                    const daysInMonth = endOfMonth(currentMonth).getDate();
                    const minutesPerDay = Math.floor(timeReport["time-totals"].minutes / daysInMonth);

                    // Criar entradas para cada dia do mês
                    for (let day = 1; day <= daysInMonth; day++) {
                        const date = new Date(currentMonth.getFullYear(), currentMonth.getMonth(), day);

                        // Pular fins de semana
                        if (isWeekend(date)) continue;

                        const dateStr = formatDate(date);
                        processedEntries.push({
                            date: dateStr,
                            minutes: minutesPerDay,
                            hours: minutesPerDay / 60,
                            description: "Tempo registrado (estimado)",
                            projectName: "Teamwork",
                            isBillable: true
                        });
                    }
                }
            } else {
                // Sem dados de tempo registrado
                console.log('Sem tempo registrado para este período');
            }

            // Consolidar entradas por dia (somar minutos de múltiplas entradas no mesmo dia)
            const consolidatedEntries = [];
            const entriesByDay = {};

            processedEntries.forEach(entry => {
                const { date, minutes, hours, description, projectName, isBillable } = entry;

                if (!entriesByDay[date]) {
                    entriesByDay[date] = {
                        date,
                        minutes,
                        hours,
                        description,
                        projectName,
                        isBillable,
                        isConsolidated: false
                    };
                } else {
                    // Se já existe uma entrada para este dia, somamos os minutos
                    entriesByDay[date].minutes += minutes;
                    entriesByDay[date].hours += hours;
                    entriesByDay[date].isConsolidated = true;
                    if (entriesByDay[date].description.indexOf('múltiplas entradas') === -1) {
                        entriesByDay[date].description = 'Consolidado (múltiplas entradas)';
                    }
                }
            });

            // Converter o objeto em array
            Object.values(entriesByDay).forEach(entry => {
                consolidatedEntries.push(entry);
            });

            setTimeEntries(consolidatedEntries);
            console.log('Entradas de tempo processadas:', consolidatedEntries);

        } catch (error) {
            console.error('Erro ao carregar entradas de tempo:', error);
            setError(`Erro ao carregar dados: ${error.message || 'Erro desconhecido'}`);
            setTimeEntries([]);
        } finally {
            setLoading(false);
        }
    };

    const getMinutesForDay = (day) => {
        const dayStr = formatDate(day);
        return timeEntries
            .filter(entry => entry.date === dayStr)
            .reduce((total, entry) => total + entry.minutes, 0);
    };

    const getDayStatus = (day) => {
        if (isWeekend(day)) return 'weekend';

        const minutes = getMinutesForDay(day);

        if (minutes === 0) return 'missing';
        if (minutes < dailyGoal) return 'incomplete';
        return 'complete';
    };

    const getDayStatusClass = (status) => {
        switch (status) {
            case 'complete': return 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 border-green-300 dark:border-green-700';
            case 'incomplete': return 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300 border-yellow-300 dark:border-yellow-700';
            case 'missing': return 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 border-red-300 dark:border-red-700';
            case 'weekend': return 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400 border-gray-300 dark:border-gray-700';
            default: return 'bg-white dark:bg-gray-800 text-gray-800 dark:text-gray-200 border-gray-300 dark:border-gray-700';
        }
    };

    const handlePrevMonth = () => {
        setCurrentMonth(addMonths(currentMonth, -1));
    };

    const handleNextMonth = () => {
        setCurrentMonth(addMonths(currentMonth, 1));
    };

    const handleDayClick = (day) => {
        if (isWeekend(day)) return;

        setSelectedDay(day);

        const dayStr = formatDate(day);
        const dayEntries = timeEntries.filter(entry => entry.date === dayStr);

        const minutesLogged = dayEntries.reduce((total, entry) => total + entry.minutes, 0);
        const hoursLogged = (minutesLogged / 60).toFixed(1);

        setDayDetails({
            date: day,
            entries: dayEntries,
            totalMinutes: minutesLogged,
            totalHours: hoursLogged,
            status: getDayStatus(day)
        });

        setShowModal(true);

        if (onDayClick) {
            onDayClick(day, dayEntries);
        }
    };

    const closeModal = () => {
        setShowModal(false);
        setSelectedDay(null);
        setDayDetails(null);
    };

    const renderDayCell = (day) => {
        const dayNum = day.getDate();
        const isToday = isSameDay(day, new Date());
        const status = getDayStatus(day);
        const statusClass = getDayStatusClass(status);
        const minutes = getMinutesForDay(day);
        const hours = (minutes / 60).toFixed(1);

        const isSelected = selectedDay && isSameDay(day, selectedDay);

        return (
            <div
                key={day.toString()}
                onClick={() => handleDayClick(day)}
                className={`relative p-2 border ${statusClass} ${isToday ? 'ring-2 ring-primary-500 dark:ring-primary-400' : ''} 
                ${isSelected ? 'ring-2 ring-blue-500 dark:ring-blue-400' : ''} 
                ${!isWeekend(day) ? 'cursor-pointer hover:shadow-md' : ''} rounded-md h-20 flex flex-col`}
            >
                <div className="text-sm font-medium mb-1">
                    {dayNum}
                </div>

                {!isWeekend(day) && (
                    <>
                        <div className="text-xs mt-auto">
                            {minutes > 0 ? (
                                <div className="flex items-center">
                                    <FiClock className="mr-1 w-3 h-3" />
                                    <span>{hours}h</span>
                                </div>
                            ) : (
                                <span className="text-gray-400 dark:text-gray-500">Sem registros</span>
                            )}
                        </div>

                        {status === 'complete' && (
                            <div className="absolute top-1 right-1">
                                <FiCheckCircle className="w-4 h-4 text-green-500 dark:text-green-400" />
                            </div>
                        )}

                        {status === 'incomplete' && (
                            <div className="absolute top-1 right-1">
                                <FiAlertCircle className="w-4 h-4 text-yellow-500 dark:text-yellow-400" />
                            </div>
                        )}

                        {status === 'missing' && (
                            <div className="absolute top-1 right-1">
                                <FiX className="w-4 h-4 text-red-500 dark:text-red-400" />
                            </div>
                        )}
                    </>
                )}
            </div>
        );
    };

    const monthStart = startOfMonth(currentMonth);
    const monthEnd = endOfMonth(currentMonth);
    const days = eachDayOfInterval(monthStart, monthEnd);

    const weeks = [];
    let week = [];

    const firstDayOfMonth = monthStart.getDay();
    for (let i = 0; i < firstDayOfMonth; i++) {
        week.push(null);
    }

    days.forEach(day => {
        week.push(day);
        if (week.length === 7) {
            weeks.push(week);
            week = [];
        }
    });

    if (week.length > 0) {
        for (let i = week.length; i < 7; i++) {
            week.push(null);
        }
        weeks.push(week);
    }

    const monthNames = [
        'Janeiro', 'Fevereiro', 'Março', 'Abril', 'Maio', 'Junho',
        'Julho', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Dezembro'
    ];

    const weekDays = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb'];

    return (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md p-4 mb-6">
            <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                    {monthNames[currentMonth.getMonth()]} {currentMonth.getFullYear()}
                </h2>

                <div className="flex space-x-2">
                    <button
                        onClick={handlePrevMonth}
                        className="p-1 rounded-full hover:bg-gray-100 dark:hover:bg-gray-700"
                    >
                        <FiChevronLeft className="w-5 h-5 text-gray-600 dark:text-gray-400" />
                    </button>
                    <button
                        onClick={() => setCurrentMonth(new Date())}
                        className="px-2 py-1 text-xs text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                    >
                        Hoje
                    </button>
                    <button
                        onClick={handleNextMonth}
                        className="p-1 rounded-full hover:bg-gray-100 dark:hover:bg-gray-700"
                    >
                        <FiChevronRight className="w-5 h-5 text-gray-600 dark:text-gray-400" />
                    </button>
                </div>
            </div>

            <div className="flex flex-wrap gap-2 mb-4">
                <div className="flex items-center">
                    <div className="w-3 h-3 bg-green-500 dark:bg-green-400 rounded-full mr-1"></div>
                    <span className="text-xs text-gray-600 dark:text-gray-400">Completo (8h+)</span>
                </div>
                <div className="flex items-center">
                    <div className="w-3 h-3 bg-yellow-500 dark:bg-yellow-400 rounded-full mr-1"></div>
                    <span className="text-xs text-gray-600 dark:text-gray-400">Incompleto (&lt;8h)</span>
                </div>
                <div className="flex items-center">
                    <div className="w-3 h-3 bg-red-500 dark:bg-red-400 rounded-full mr-1"></div>
                    <span className="text-xs text-gray-600 dark:text-gray-400">Sem Registro</span>
                </div>
                <div className="flex items-center">
                    <div className="w-3 h-3 bg-gray-400 dark:bg-gray-500 rounded-full mr-1"></div>
                    <span className="text-xs text-gray-600 dark:text-gray-400">Fim de semana</span>
                </div>
            </div>

            {loading && (
                <div className="flex justify-center items-center py-8">
                    <div className="animate-spin w-6 h-6 border-2 border-primary-600 border-t-transparent rounded-full"></div>
                </div>
            )}

            {error && !loading && (
                <div className="bg-red-50 dark:bg-red-900/20 border-l-4 border-red-500 p-4 mb-4">
                    <div className="flex">
                        <FiAlertCircle className="h-5 w-5 text-red-500 dark:text-red-400 flex-shrink-0" />
                        <div className="ml-3">
                            <p className="text-sm text-red-700 dark:text-red-200">{error}</p>
                        </div>
                    </div>
                </div>
            )}

            {!loading && !error && (
                <>
                    <div className="grid grid-cols-7 gap-1 mb-1">
                        {weekDays.map((day, index) => (
                            <div key={index} className="p-1 text-center text-xs font-medium text-gray-500 dark:text-gray-400">
                                {day}
                            </div>
                        ))}
                    </div>

                    <div className="grid grid-cols-7 gap-1">
                        {weeks.flat().map((day, index) => (
                            day ? renderDayCell(day) : <div key={`empty-${index}`} className="border border-gray-200 dark:border-gray-700 rounded-md h-20"></div>
                        ))}
                    </div>
                </>
            )}

            {showModal && dayDetails && (
                <div className="fixed inset-0 bg-black bg-opacity-30 dark:bg-opacity-60 flex items-center justify-center z-50">
                    <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md mx-4">
                        <div className="p-5 border-b border-gray-200 dark:border-gray-700 flex justify-between items-center">
                            <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                                {new Date(dayDetails.date).toLocaleDateString('pt-BR', {weekday: 'long', day: 'numeric', month: 'long'})}
                            </h3>
                            <button
                                onClick={closeModal}
                                className="rounded-full p-1 hover:bg-gray-100 dark:hover:bg-gray-700"
                            >
                                <FiX className="w-5 h-5 text-gray-500 dark:text-gray-400" />
                            </button>
                        </div>

                        <div className="p-5">
                            <div className={`mb-4 p-3 rounded-lg ${getDayStatusClass(dayDetails.status)} border`}>
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center">
                                        {dayDetails.status === 'complete' && <FiCheckCircle className="w-5 h-5 mr-2" />}
                                        {dayDetails.status === 'incomplete' && <FiAlertCircle className="w-5 h-5 mr-2" />}
                                        {dayDetails.status === 'missing' && <FiX className="w-5 h-5 mr-2" />}
                                        <span className="font-medium">
                                            {dayDetails.status === 'complete' && 'Completo'}
                                            {dayDetails.status === 'incomplete' && 'Incompleto'}
                                            {dayDetails.status === 'missing' && 'Sem Registros'}
                                        </span>
                                    </div>
                                    <div>
                                        <FiClock className="w-4 h-4 inline mr-1" />
                                        <span>{dayDetails.totalHours}h</span>
                                    </div>
                                </div>
                            </div>

                            {dayDetails.entries.length > 0 ? (
                                <div className="space-y-3 max-h-60 overflow-y-auto">
                                    <h4 className="font-medium text-gray-900 dark:text-white mb-2">Lançamentos</h4>
                                    {dayDetails.entries.map((entry, idx) => (
                                        <div key={idx} className="p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                                            <div className="flex justify-between mb-1">
                                                <span className="font-medium text-sm">{entry.description || 'Sem descrição'}</span>
                                                <span className="text-sm">{(entry.minutes / 60).toFixed(1)}h</span>
                                            </div>
                                            <div className="text-xs text-gray-500 dark:text-gray-400">
                                                <div>Projeto: {entry.projectName || 'N/A'}</div>
                                                <div>Horário: {entry.startTime ? entry.startTime.substring(0, 5) : 'N/A'}</div>
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
                        </div>

                        <div className="px-5 py-3 border-t border-gray-200 dark:border-gray-700 flex justify-end">
                            <button
                                onClick={closeModal}
                                className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md"
                            >
                                Fechar
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default MonthlyTimeCalendar;