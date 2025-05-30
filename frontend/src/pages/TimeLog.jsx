import React, {useEffect, useRef, useState} from 'react';
import {toast} from 'react-toastify';
import {
    FiAlertCircle,
    FiCalendar,
    FiCheck,
    FiCheckCircle,
    FiClock,
    FiInfo,
    FiList,
    FiLoader,
    FiPlay,
    FiRefreshCw
} from 'react-icons/fi';
import {format, parseISO} from 'date-fns';
import {ptBR} from 'date-fns/locale';
import MonthlyTimeCalendar from '../components/MonthlyTimeCalendar';
import {useLocation} from 'react-router-dom';

const TimeLog = () => {
    const location = useLocation();
    const [isLoading, setIsLoading] = useState(true);
    const [savedTasks, setSavedTasks] = useState([]);
    const [selectedTasks, setSelectedTasks] = useState([]);
    const [dateRange, setDateRange] = useState({
        startDate: format(new Date(), 'yyyy-MM-dd'),
        endDate: format(new Date(), 'yyyy-MM-dd')
    });
    const [workDays, setWorkDays] = useState([]);
    const [isGenerating, setIsGenerating] = useState(false);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [results, setResults] = useState([]);
    const [showResults, setShowResults] = useState(false);
    const [error, setError] = useState(null);
    const [processingProgress, setProcessingProgress] = useState(0);
    const [nonWorkingDays, setNonWorkingDays] = useState({});
    const [calendarKey, setCalendarKey] = useState(0);
    const [templatesApplied, setTemplatesApplied] = useState(false);
    const [templateAlertShown, setTemplateAlertShown] = useState(false);

    const calendarRef = useRef(null);

    const loadSavedTasks = async () => {
        try {
            setIsLoading(true);
            const tasks = await window.go.backend.App.GetSavedTasks();
            setSavedTasks(tasks);

            const wasTemplateApplied = localStorage.getItem('templateApplied') === 'true';

            if (tasks.length > 0 && wasTemplateApplied && !templateAlertShown) {
                setSelectedTasks(tasks.map(task => task.taskId));
                setTemplatesApplied(true);
                setTemplateAlertShown(true);
                toast.info(`${tasks.length} tarefas carregadas do template. Clique em "Gerar Plano" para continuar.`);

                localStorage.removeItem('templateApplied');
            }
        } catch (error) {
            console.error('Erro ao carregar tarefas salvas:', error);
            toast.error('Erro ao carregar tarefas salvas.');
            setError("Erro ao carregar tarefas salvas: " + (error.message || error));
        } finally {
            setIsLoading(false);
        }
    };

    useEffect(() => {
        loadSavedTasks();
    }, []);

    useEffect(() => {
        if (location.pathname === '/timelog' && !templateAlertShown) {
            loadSavedTasks();
        }
    }, [location, templateAlertShown]);

    useEffect(() => {
        return () => {
            setTemplateAlertShown(false);
            setTemplatesApplied(false);
        };
    }, []);

    useEffect(() => {
        const loadNonWorkingDays = async () => {
            try {
                if (!dateRange.startDate || !dateRange.endDate) return;

                const startDate = new Date(dateRange.startDate);
                const endDate = new Date(dateRange.endDate);

                const monthsToCheck = new Set();
                let currentDate = new Date(startDate);

                while (currentDate <= endDate) {
                    const yearMonth = `${currentDate.getFullYear()}-${currentDate.getMonth() + 1}`;
                    monthsToCheck.add(yearMonth);
                    currentDate.setDate(currentDate.getDate() + 1);
                }

                const nonWorkingDaysMap = {};

                for (const yearMonth of monthsToCheck) {
                    const [year, month] = yearMonth.split('-').map(Number);
                    const nonWorkingDaysForMonth = await window.go.backend.App.GetAllNonWorkingDays(year, month);

                    nonWorkingDaysForMonth.forEach(day => {
                        nonWorkingDaysMap[day.date] = day;
                    });
                }

                setNonWorkingDays(nonWorkingDaysMap);
            } catch (error) {
                console.error('Erro ao carregar dias não úteis:', error);
            }
        };

        loadNonWorkingDays();
    }, [dateRange]);

    const refreshCalendar = () => {
        setCalendarKey(prev => prev + 1);

        if (calendarRef.current && calendarRef.current.refresh) {
            calendarRef.current.refresh();
        }

        toast.success('Calendário atualizado com os novos lançamentos!');
    };

    const handleDaySelection = async (day, entries) => {
        const formattedDate = day.toISOString().split('T')[0];

        const isNonWorkingDay = nonWorkingDays[formattedDate];
        if (isNonWorkingDay) {
            if (isNonWorkingDay.type === 'holiday') {
                toast.warning(`${formattedDate} é um feriado: ${isNonWorkingDay.name}. Não é possível lançar horas em feriados.`);
            } else if (isNonWorkingDay.type === 'weekend') {
                toast.warning(`${formattedDate} é um fim de semana. Não é possível lançar horas em fins de semana.`);
            }
            return;
        }

        try {
            const isWorkDay = await window.go.backend.App.IsWorkDay(formattedDate);
            if (!isWorkDay) {
                toast.warning(`${formattedDate} não é um dia útil. Não é possível lançar horas.`);
                return;
            }
        } catch (error) {
            console.error("Erro ao verificar dia útil:", error);
        }

        setDateRange({
            startDate: formattedDate,
            endDate: formattedDate
        });

        if (entries && entries.length > 0) {
            const taskIdsFromEntries = entries.map(entry => entry.taskId);
            const existingTasks = savedTasks.filter(task =>
                taskIdsFromEntries.includes(task.taskId)
            );

            if (existingTasks.length > 0) {
                setSelectedTasks(existingTasks.map(task => task.taskId));
            }
        }

        setTimeout(() => {
            generatePlan();
        }, 100);
    };

    const toggleTaskSelection = (taskId) => {
        if (selectedTasks.includes(taskId)) {
            setSelectedTasks(selectedTasks.filter(id => id !== taskId));
        } else {
            setSelectedTasks([...selectedTasks, taskId]);
        }
    };

    const selectAllTasks = () => {
        if (selectedTasks.length === savedTasks.length) {
            setSelectedTasks([]);
        } else {
            setSelectedTasks(savedTasks.map(task => task.taskId));
        }
    };

    const generatePlan = async () => {
        if (selectedTasks.length === 0) {
            toast.warning('Selecione pelo menos uma tarefa para lançar horas.');
            return;
        }

        if (!dateRange.startDate || !dateRange.endDate) {
            toast.warning('Selecione um intervalo de datas válido.');
            return;
        }

        setIsGenerating(true);
        setWorkDays([]);
        setError(null);

        try {
            const workingDays = await window.go.backend.App.GetWorkingDays(
                dateRange.startDate,
                dateRange.endDate
            );

            if (!workingDays || workingDays.length === 0) {
                toast.warning('Não foram encontrados dias úteis no período selecionado.');
                setIsGenerating(false);
                return;
            }

            const filteredTasks = savedTasks.filter(task =>
                selectedTasks.includes(task.taskId)
            );

            // DEBUG: Log das tarefas que serão enviadas
            console.log('Tarefas filtradas para o plano:', filteredTasks.map(t => ({
                taskId: t.taskId,
                taskName: t.taskName,
                workingDays: t.workingDays
            })));

            const plan = await window.go.backend.App.CreateDistributionPlan(
                workingDays,
                filteredTasks
            );

            if (!plan || plan.length === 0) {
                toast.warning('Não foi possível gerar um plano de lançamento.');
                return;
            }

            console.log('Plano gerado:', plan);
            setWorkDays(plan);
            toast.success(`Plano gerado com sucesso para ${plan.length} dias!`);
        } catch (error) {
            console.error('Erro ao gerar plano:', error);
            toast.error('Erro ao gerar plano de lançamento.');
            setError("Erro ao gerar plano: " + (error.message || error));
        } finally {
            setIsGenerating(false);
        }
    };

    const submitPlan = async () => {
        if (workDays.length === 0) {
            toast.warning('Gere um plano antes de lançar horas.');
            return;
        }

        setIsSubmitting(true);
        setResults([]);
        setShowResults(false);
        setError(null);
        setProcessingProgress(0);

        try {
            const totalEntries = workDays.reduce((sum, day) => sum + day.entries.length, 0);

            const toastId = toast.info('Processando lançamentos...', {
                autoClose: false,
                closeButton: false
            });

            const results = await window.go.backend.App.LogMultipleTimes(workDays);

            toast.dismiss(toastId);

            if (!results || results.length === 0) {
                toast.error('Não foram recebidos resultados do lançamento.');
                setError("Não foram recebidos resultados do lançamento.");
                return;
            }

            setResults(results);
            setShowResults(true);

            const successes = results.filter(r => r.success).length;
            const failures = results.length - successes;

            if (failures === 0) {
                toast.success(`${successes} lançamentos realizados com sucesso!`);

                setTimeout(() => {
                    refreshCalendar();
                }, 1000);

            } else if (successes === 0) {
                toast.error(`Falha em todos os ${failures} lançamentos.`);
            } else {
                toast.warning(`${successes} lançamentos com sucesso e ${failures} falhas.`);

                setTimeout(() => {
                    refreshCalendar();
                }, 1000);
            }
        } catch (error) {
            console.error('Erro ao lançar horas:', error);
            toast.error('Erro ao lançar horas no Teamwork.');
            setError("Erro ao lançar horas: " + (error.message || error));
        } finally {
            setIsSubmitting(false);
            setProcessingProgress(100);
        }
    };

    const formatDate = (dateString) => {
        try {
            return format(parseISO(dateString), 'dd/MM/yyyy (EEEE)', {locale: ptBR});
        } catch (error) {
            return dateString || "Data inválida";
        }
    };

    const calculateTotalHours = () => {
        if (!workDays || workDays.length === 0) return {days: 0, hours: 0, minutes: 0, totalMinutes: 0};

        const totalMinutes = workDays.reduce((sum, day) => sum + (day.totalMin || 0), 0);
        const hours = Math.floor(totalMinutes / 60);
        const minutes = totalMinutes % 60;

        return {
            days: workDays.length,
            hours,
            minutes,
            totalMinutes
        };
    };

    const isDateNonWorkingDay = (dateStr) => {
        return nonWorkingDays[dateStr] !== undefined;
    };

    const getNonWorkingDayInfo = (dateStr) => {
        if (!nonWorkingDays[dateStr]) return null;

        const day = nonWorkingDays[dateStr];
        if (day.type === 'holiday') {
            return `Feriado: ${day.name}`;
        } else if (day.type === 'weekend') {
            return 'Fim de semana';
        }
        return 'Dia não útil';
    };

    const getTaskDaysSummary = () => {
        if (!workDays || workDays.length === 0) return '';

        const taskDays = {};
        workDays.forEach(day => {
            day.entries?.forEach(entry => {
                const taskId = entry.taskId;
                if (!taskDays[taskId]) {
                    taskDays[taskId] = new Set();
                }
                taskDays[taskId].add(day.date);
            });
        });

        return Object.entries(taskDays).map(([taskId, daysSet]) => {
            const task = savedTasks.find(t => t.taskId === parseInt(taskId));
            return `${task?.taskName || `Tarefa ${taskId}`}: ${daysSet.size} dias`;
        }).join(' • ');
    };

    const formatWorkingDays = (workingDays) => {
        if (!workingDays || workingDays.length === 0) return 'Todos os dias';

        const diasNomes = {
            0: 'Dom', 1: 'Seg', 2: 'Ter', 3: 'Qua',
            4: 'Qui', 5: 'Sex', 6: 'Sáb'
        };

        if (workingDays.length === 7) return 'Todos os dias';
        if (workingDays.length === 5 &&
            [1, 2, 3, 4, 5].every(day => workingDays.includes(day))) {
            return 'Dias úteis';
        }

        return workingDays
            .sort()
            .map(day => diasNomes[day])
            .join(', ');
    };

    const totals = calculateTotalHours();

    if (isLoading) {
        return (
            <div className="flex justify-center items-center h-full">
                <div
                    className="animate-spin-slow w-12 h-12 border-4 border-primary-600 border-t-transparent rounded-full"></div>
            </div>
        );
    }

    return (
        <div>
            <div className="mb-6 flex justify-between items-center">
                <div>
                    <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">Lançar Horas</h1>
                    <p className="text-gray-600 dark:text-gray-400">Registre horas trabalhadas em múltiplos dias</p>
                </div>
                <button
                    onClick={refreshCalendar}
                    className="flex items-center px-3 py-2 text-sm bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 rounded-lg transition-colors"
                    title="Atualizar calendário"
                >
                    <FiRefreshCw className="w-4 h-4 mr-2"/>
                    Atualizar Calendário
                </button>
            </div>
            <MonthlyTimeCalendar
                key={calendarKey}
                ref={calendarRef}
                onDayClick={handleDaySelection}
            />

            {savedTasks.length > 0 && templatesApplied && (
                <div
                    className="mb-6 bg-blue-50 border-l-4 border-blue-500 p-4 dark:bg-blue-900/20 dark:border-blue-700">
                    <div className="flex items-start">
                        <FiInfo className="mt-0.5 w-5 h-5 text-blue-500 dark:text-blue-600 mr-2"/>
                        <div>
                            <h3 className="text-sm font-medium text-blue-800 dark:text-blue-400">
                                Template Aplicado
                            </h3>
                            <p className="mt-1 text-sm text-blue-700 dark:text-blue-200">
                                {savedTasks.length} tarefas foram carregadas do template. Configure o período e clique
                                em "Gerar Plano" para continuar.
                            </p>
                        </div>
                    </div>
                </div>
            )}

            {error && (
                <div className="mb-6 bg-red-50 border-l-4 border-red-500 p-4 dark:bg-red-900/20 dark:border-red-700">
                    <div className="flex items-start">
                        <FiAlertCircle className="mt-0.5 w-5 h-5 text-red-500 dark:text-red-600 mr-2"/>
                        <div>
                            <h3 className="text-sm font-medium text-red-800 dark:text-red-400">Erro</h3>
                            <p className="mt-1 text-sm text-red-700 dark:text-red-200">{error}</p>
                        </div>
                    </div>
                </div>
            )}

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
                <div className="card lg:col-span-2">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                        <FiList className="w-5 h-5 mr-2"/>
                        Tarefas para Lançamento
                    </h2>

                    {savedTasks.length === 0 ? (
                        <div
                            className="bg-yellow-50 border-l-4 border-yellow-400 p-4 dark:bg-yellow-900/20 dark:border-yellow-600">
                            <div className="flex items-start">
                                <FiAlertCircle className="mt-0.5 w-5 h-5 text-yellow-500 dark:text-yellow-600 mr-2"/>
                                <div>
                                    <h3 className="text-sm font-medium text-yellow-800 dark:text-yellow-400">
                                        Nenhuma tarefa salva
                                    </h3>
                                    <p className="mt-1 text-sm text-yellow-700 dark:text-yellow-200">
                                        Adicione tarefas na seção "Tarefas" ou aplique um template da seção "Templates".
                                    </p>
                                </div>
                            </div>
                        </div>
                    ) : (
                        <>
                            <div className="mb-4">
                                <button
                                    type="button"
                                    onClick={selectAllTasks}
                                    className="text-sm text-primary-600 hover:text-primary-700 dark:text-primary-500 dark:hover:text-primary-400"
                                >
                                    {selectedTasks.length === savedTasks.length ? 'Desmarcar todas' : 'Selecionar todas'}
                                </button>
                            </div>

                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                {savedTasks.map(task => (
                                    <div
                                        key={task.taskId}
                                        className={`border rounded-lg p-3 ${
                                            selectedTasks.includes(task.taskId)
                                                ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20 dark:border-primary-700'
                                                : 'border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600'
                                        }`}
                                    >
                                        <div className="flex items-start">
                                            <div className="flex-shrink-0">
                                                <input
                                                    type="checkbox"
                                                    checked={selectedTasks.includes(task.taskId)}
                                                    onChange={() => toggleTaskSelection(task.taskId)}
                                                    className="w-4 h-4 text-primary-600 bg-gray-100 border-gray-300 rounded focus:ring-primary-500 dark:focus:ring-primary-600 dark:ring-offset-gray-800 dark:focus:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                                                />
                                            </div>
                                            <div className="ml-3">
                                                <h3 className="text-sm font-medium text-gray-900 dark:text-white">
                                                    {task.taskName}
                                                </h3>
                                                <p className="text-xs text-gray-500 dark:text-gray-400">
                                                    {task.projectName}
                                                </p>
                                                <div className="mt-1 text-xs">
                                                    {task.entries.length} entradas •
                                                    {task.entries.reduce((sum, e) => sum + e.minutes, 0)} min
                                                </div>
                                                {task.workingDays && (
                                                    <div className="mt-1 text-xs text-blue-600 dark:text-blue-400">
                                                        <FiClock className="inline w-3 h-3 mr-1"/>
                                                        {formatWorkingDays(task.workingDays)}
                                                    </div>
                                                )}
                                            </div>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </>
                    )}
                </div>

                <div className="card">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                        <FiCalendar className="w-5 h-5 mr-2"/>
                        Período
                    </h2>

                    <div className="space-y-4">
                        <div>
                            <label htmlFor="startDate"
                                   className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Data Inicial
                            </label>
                            <input
                                type="date"
                                id="startDate"
                                value={dateRange.startDate}
                                onChange={(e) => setDateRange({...dateRange, startDate: e.target.value})}
                                className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                            />
                            {isDateNonWorkingDay(dateRange.startDate) && (
                                <p className="mt-1 text-xs text-yellow-600 dark:text-yellow-400">
                                    <FiAlertCircle className="inline-block mr-1"/>
                                    {getNonWorkingDayInfo(dateRange.startDate)}
                                </p>
                            )}
                        </div>

                        <div>
                            <label htmlFor="endDate"
                                   className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Data Final
                            </label>
                            <input
                                type="date"
                                id="endDate"
                                value={dateRange.endDate}
                                onChange={(e) => setDateRange({...dateRange, endDate: e.target.value})}
                                className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                            />
                            {isDateNonWorkingDay(dateRange.endDate) && (
                                <p className="mt-1 text-xs text-yellow-600 dark:text-yellow-400">
                                    <FiAlertCircle className="inline-block mr-1"/>
                                    {getNonWorkingDayInfo(dateRange.endDate)}
                                </p>
                            )}
                        </div>

                        <button
                            type="button"
                            onClick={generatePlan}
                            disabled={isGenerating || savedTasks.length === 0}
                            className="btn-primary w-full flex items-center justify-center"
                        >
                            {isGenerating ? (
                                <>
                                    <FiLoader className="w-5 h-5 mr-2 animate-spin"/>
                                    Gerando...
                                </>
                            ) : (
                                <>
                                    <FiPlay className="w-5 h-5 mr-2"/>
                                    Gerar Plano
                                </>
                            )}
                        </button>
                    </div>
                </div>
            </div>

            {workDays && workDays.length > 0 && (
                <div className="card mb-6">
                    <div className="flex items-center justify-between mb-4">
                        <h2 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center">
                            <FiClock className="w-5 h-5 mr-2"/>
                            Plano de Lançamento
                        </h2>

                        <div className="text-sm text-gray-600 dark:text-gray-400">
                            <span className="font-medium">{totals.days} dias</span> •
                            <span className="font-medium">{totals.hours}h {totals.minutes}min</span> •
                            <span className="font-medium">
                              {workDays.reduce((sum, day) => sum + (day.entries ? day.entries.length : 0), 0)} entradas
                          </span>
                        </div>
                    </div>

                    {getTaskDaysSummary() && (
                        <div className="mb-4 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                            <p className="text-sm text-blue-700 dark:text-blue-300">
                                <strong>Distribuição por tarefa:</strong> {getTaskDaysSummary()}
                            </p>
                        </div>
                    )}

                    <div className="overflow-y-auto max-h-96">
                        <div className="space-y-4">
                            {workDays.map((day, dayIndex) => (
                                <div key={dayIndex}
                                     className="border border-gray-200 rounded-lg p-4 dark:border-gray-700">
                                    <h3 className="text-md font-medium text-gray-900 dark:text-white mb-2">
                                        {formatDate(day.date)}
                                    </h3>

                                    {day.entries && day.entries.length > 0 ? (
                                        <div className="space-y-2">
                                            {day.entries.map((entry, entryIndex) => (
                                                <div
                                                    key={`${dayIndex}-${entryIndex}`}
                                                    className="bg-gray-50 rounded-md p-3 text-sm dark:bg-gray-800"
                                                >
                                                    <div className="flex justify-between">
                                                        <div>
                                                          <span className="font-medium">
                                                              {entry.entry.description}
                                                          </span>
                                                            <span className="text-gray-600 dark:text-gray-400 ml-2">
                                                              ({entry.entry.minutes} min • {entry.entry.time ? entry.entry.time.substring(0, 5) : "00:00"})
                                                          </span>
                                                        </div>
                                                        <div className="text-gray-700 dark:text-gray-300">
                                                            TaskID: {entry.taskId}
                                                        </div>
                                                    </div>
                                                </div>
                                            ))}
                                        </div>
                                    ) : (
                                        <p className="text-sm text-gray-500 dark:text-gray-400 py-2">
                                            Nenhuma entrada para este dia.
                                        </p>
                                    )}

                                    <div className="mt-2 text-right text-sm">
                                        <span className="text-gray-600 dark:text-gray-400">Total do dia:</span>
                                        <span className="font-medium ml-1">{day.totalMin || 0} minutos</span>
                                        <span className="text-gray-600 dark:text-gray-400 ml-1">
                                          ({((day.totalMin || 0) / 60).toFixed(1)}h)
                                      </span>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>

                    <div className="mt-4">
                        {isSubmitting && (
                            <div className="mb-4">
                                <div className="w-full bg-gray-200 rounded-full h-2.5 dark:bg-gray-700 mb-2">
                                    <div className="bg-primary-600 h-2.5 rounded-full dark:bg-primary-500"
                                         style={{width: `${processingProgress}%`}}></div>
                                </div>
                                <p className="text-sm text-gray-500 dark:text-gray-400 text-center">
                                    Processando lançamentos...
                                </p>
                            </div>
                        )}

                        <button
                            type="button"
                            onClick={submitPlan}
                            disabled={isSubmitting}
                            className="btn-success flex items-center justify-center"
                        >
                            {isSubmitting ? (
                                <>
                                    <FiLoader className="w-5 h-5 mr-2 animate-spin"/>
                                    Enviando...
                                </>
                            ) : (
                                <>
                                    <FiCheck className="w-5 h-5 mr-2"/>
                                    Executar Lançamento
                                </>
                            )}
                        </button>
                    </div>
                </div>
            )}

            {showResults && results && results.length > 0 && (
                <div className="card">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                        <FiCheckCircle className="w-5 h-5 mr-2"/>
                        Resultados do Lançamento
                    </h2>

                    <div className="overflow-y-auto max-h-64">
                        <div className="space-y-2">
                            {results.map((result, index) => (
                                <div
                                    key={index}
                                    className={`p-3 rounded-md ${
                                        result.success
                                            ? 'bg-green-50 dark:bg-green-900/20'
                                            : 'bg-red-50 dark:bg-red-900/20'
                                    }`}
                                >
                                    <div className="flex items-start">
                                        <div
                                            className={`flex-shrink-0 w-5 h-5 rounded-full flex items-center justify-center ${
                                                result.success
                                                    ? 'bg-green-100 text-green-600 dark:bg-green-800 dark:text-green-200'
                                                    : 'bg-red-100 text-red-600 dark:bg-red-800 dark:text-red-200'
                                            }`}
                                        >
                                            {result.success ? <FiCheck size={12}/> : <FiAlertCircle size={12}/>}
                                        </div>
                                        <div className="ml-3">
                                            <p className={`text-sm ${
                                                result.success
                                                    ? 'text-green-800 dark:text-green-200'
                                                    : 'text-red-800 dark:text-red-200'
                                            }`}>
                                              <span className="font-medium">
                                                  {result.success ? 'Sucesso' : 'Falha'}:
                                              </span> {result.message || 'Sem mensagem'}
                                            </p>
                                            <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                                                Tarefa: {result.taskId} • Data: {result.date || 'N/A'}
                                            </p>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>

                    <div className="mt-4 bg-gray-50 p-3 rounded-md dark:bg-gray-800">
                        <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">Resumo</h3>
                        <div className="grid grid-cols-2 gap-2">
                            <div className="bg-green-50 p-2 rounded dark:bg-green-900/20">
                                <p className="text-xs text-green-800 dark:text-green-200">
                                  <span
                                      className="font-medium">Sucessos:</span> {results.filter(r => r.success).length}
                                </p>
                            </div>
                            <div className="bg-red-50 p-2 rounded dark:bg-red-900/20">
                                <p className="text-xs text-red-800 dark:text-red-200">
                                  <span
                                      className="font-medium">Falhas:</span> {results.filter(r => !r.success).length}
                                </p>
                            </div>
                        </div>

                        <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-600">
                            <button
                                onClick={refreshCalendar}
                                className="w-full flex items-center justify-center px-3 py-2 text-sm bg-primary-100 hover:bg-primary-200 dark:bg-primary-900/20 dark:hover:bg-primary-900/40 text-primary-700 dark:text-primary-300 rounded-lg transition-colors"
                            >
                                <FiRefreshCw className="w-4 h-4 mr-2"/>
                                Atualizar Calendário
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default TimeLog;