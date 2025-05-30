import React, { useState, useEffect } from 'react';
import {
    FiClock,
    FiList,
    FiFileText,
    FiCalendar,
    FiAlertCircle,
    FiDownload,
    FiClipboard,
    FiArrowUp,
    FiArrowDown,
    FiLoader,
    FiRefreshCw,
    FiFlag,
    FiTrash2
} from 'react-icons/fi';
import { toast } from 'react-toastify';
import { format, parseISO } from 'date-fns';
import { ptBR } from 'date-fns/locale';
import { useNavigate } from 'react-router-dom';
import ReportPeriodModal from './ReportPeriodModal';
import MonthlyTimeCalendar from '../components/MonthlyTimeCalendar';
import TimeEntryManager from '../components/TimeEntryManager';

const StatCard = ({ title, icon, value, description, change, className }) => {
    return (
        <div className="card flex items-start space-x-4">
            <div className={`flex items-center justify-center w-12 h-12 rounded-lg ${className}`}>
                {icon}
            </div>
            <div className="flex-1">
                <h3 className="text-gray-500 dark:text-gray-400 text-sm font-medium">{title}</h3>
                <div className="flex items-baseline">
                    <p className="text-3xl font-bold text-gray-900 dark:text-white">{value}</p>
                    {change !== undefined && change !== null && (
                        <span className={`ml-2 text-sm ${change >= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'} flex items-center`}>
                           {change >= 0 ? <FiArrowUp className="mr-1" /> : <FiArrowDown className="mr-1" />}
                            {Math.abs(change)}%
                       </span>
                    )}
                </div>
                <p className="text-sm text-gray-500 dark:text-gray-400">{description}</p>
            </div>
        </div>
    );
};

const ActivityItem = ({ activity }) => {
    return (
        <div className="border-b border-gray-200 dark:border-gray-700 py-3 last:border-0">
            <div className="flex items-start">
                <div className="flex-shrink-0 w-10 h-10 flex items-center justify-center bg-blue-100 text-blue-600 rounded-full dark:bg-blue-900 dark:text-blue-300">
                    <FiClock className="w-5 h-5" />
                </div>
                <div className="ml-3 flex-1">
                    <p className="text-sm font-medium text-gray-900 dark:text-white">
                        {activity.description || "Tempo registrado"}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                        {activity.projectName} • {Math.round(activity.minutes / 60 * 10) / 10}h
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                        {format(new Date(activity.date), 'dd/MM/yyyy')}
                    </p>
                </div>
            </div>
        </div>
    );
};

const UpcomingTaskItem = ({ task }) => {
    return (
        <div className="border-b border-gray-200 dark:border-gray-700 py-3 last:border-0">
            <div className="flex items-start">
                <div className="flex-shrink-0 w-10 h-10 flex items-center justify-center bg-amber-100 text-amber-600 rounded-full dark:bg-amber-900 dark:text-amber-300">
                    <FiFlag className="w-5 h-5" />
                </div>
                <div className="ml-3 flex-1">
                    <p className="text-sm font-medium text-gray-900 dark:text-white">
                        {task.name}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                        {task.projectName}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                        Vencimento: {format(new Date(task.dueDate), 'dd/MM/yyyy')}
                    </p>
                </div>
            </div>
        </div>
    );
};

const Dashboard = () => {
    const navigate = useNavigate();
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState(null);
    const [dashboardData, setDashboardData] = useState({
        horasLogadas: 0,
        horasLogadasChange: 0,
        tarefasPendentes: 0,
        projetos: 0,
        diasUteisMes: 0,
        diasUteisRestantes: 0,
        diasUteisPassados: 0
    });
    const [recentActivities, setRecentActivities] = useState([]);
    const [upcomingTasks, setUpcomingTasks] = useState([]);
    const [isExporting, setIsExporting] = useState(false);
    const [isRefreshing, setIsRefreshing] = useState(false);
    const [lastUpdate, setLastUpdate] = useState(new Date());
    const [projectsData, setProjectsData] = useState([]);
    const [tasksData, setTasksData] = useState([]);
    const [isReportModalOpen, setIsReportModalOpen] = useState(false);
    const [isTimeManagerOpen, setIsTimeManagerOpen] = useState(false);

    const loadProjectsAndTasks = async () => {
        try {
            const projects = await window.go.backend.App.GetProjects();
            setProjectsData(projects || []);

            if (projects && projects.length > 0) {
                const projectID = projects[0].id;
                const tasks = await window.go.backend.App.GetTasksByProject(projectID);
                setTasksData(tasks || []);

                setDashboardData(prevData => ({
                    ...prevData,
                    projetos: projects.length,
                    tarefasPendentes: tasks.length
                }));
            }
        } catch (error) {
            console.error("Erro ao carregar projetos e tarefas:", error);
        }
    };

    const loadDashboard = async () => {
        try {
            setIsLoading(true);
            setError(null);

            const now = new Date();
            const startDate = format(new Date(now.getFullYear(), now.getMonth(), 1), 'yyyy-MM-dd');
            const endDate = format(new Date(now.getFullYear(), now.getMonth() + 1, 0), 'yyyy-MM-dd');

            const [dashboardStats, recentActivitiesData, upcomingTasksData, timeReport] = await Promise.all([
                window.go.backend.App.GetDashboardStats(),
                window.go.backend.App.GetRecentActivities(),
                window.go.backend.App.GetTasksWithUpcomingDeadlines(),
                window.go.backend.App.GetTimeTotalsForPeriod(startDate, endDate)
            ]);

            let horasLogadas = dashboardStats.horasLogadas || 0;

            if (timeReport && timeReport["time-totals"] && timeReport["time-totals"].minutes) {
                horasLogadas = timeReport["time-totals"].minutes / 60;
            }

            setDashboardData(prevData => ({
                ...prevData,
                ...dashboardStats,
                horasLogadas: horasLogadas,
                projetos: (dashboardStats.projetos === 0 && prevData.projetos > 0) ? prevData.projetos : dashboardStats.projetos,
                tarefasPendentes: (dashboardStats.tarefasPendentes === 0 && prevData.tarefasPendentes > 0) ? prevData.tarefasPendentes : dashboardStats.tarefasPendentes
            }));

            setRecentActivities(recentActivitiesData || []);
            setUpcomingTasks(upcomingTasksData || []);
            setLastUpdate(new Date());

            if (dashboardStats.projetos === 0 || dashboardStats.tarefasPendentes === 0) {
                await loadProjectsAndTasks();
            }
        } catch (error) {
            console.error("Erro ao carregar dashboard:", error);
            setError("Não foi possível carregar os dados do dashboard. Verifique sua conexão com o Teamwork.");

            await loadProjectsAndTasks();
        } finally {
            setIsLoading(false);
        }
    };

    useEffect(() => {
        loadDashboard();
    }, []);

    const refreshDashboard = async () => {
        if (isRefreshing) return;

        setIsRefreshing(true);
        try {
            await loadDashboard();
            toast.success("Dashboard atualizado com sucesso!");
        } catch (error) {
            toast.error("Erro ao atualizar o dashboard");
        } finally {
            setIsRefreshing(false);
        }
    };

    const navigateTo = (path) => {
        navigate(path);
    };

    const handleExportReport = async () => {
        try {
            setIsExporting(true);
            const filePath = await window.go.backend.App.DownloadCurrentMonthReport();
            toast.success(`Relatório exportado com sucesso para: ${filePath}`);
            await window.go.backend.App.OpenDirectoryPath(filePath);
        } catch (error) {
            console.error("Erro ao exportar relatório:", error);
            toast.error("Erro ao exportar relatório: " + (error.message || "Erro desconhecido"));
        } finally {
            setIsExporting(false);
        }
    };

    const handleExportCustomPeriodReport = async (startDate, endDate) => {
        try {
            setIsExporting(true);
            const filePath = await window.go.backend.App.DownloadTimeReport(startDate, endDate);
            toast.success(`Relatório exportado com sucesso para: ${filePath}`);
            await window.go.backend.App.OpenDirectoryPath(filePath);
        } catch (error) {
            console.error("Erro ao exportar relatório do período:", error);
            toast.error("Não foi possível exportar o relatório: " + (error.message || "Erro desconhecido"));
        } finally {
            setIsExporting(false);
        }
    };

    const handleTimeManagerClose = () => {
        setIsTimeManagerOpen(false);
        loadDashboard();
    };

    const formatarHoras = (horas) => {
        if (typeof horas !== 'number') return '0h';
        return `${horas.toFixed(1)}h`;
    };

    if (isLoading) {
        return (
            <div className="flex justify-center items-center h-full">
                <div className="animate-spin-slow w-12 h-12 border-4 border-primary-600 border-t-transparent rounded-full"></div>
            </div>
        );
    }

    return (
        <div>
            <div className="mb-6 flex justify-between items-center">
                <div>
                    <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">Dashboard</h1>
                    <p className="text-gray-600 dark:text-gray-400">Visão geral das suas atividades no Teamwork</p>
                </div>
                <button
                    onClick={refreshDashboard}
                    disabled={isRefreshing}
                    className="p-2 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700">
                    <FiRefreshCw className={`w-5 h-5 text-gray-500 dark:text-gray-400 ${isRefreshing ? 'animate-spin' : ''}`} />
                </button>
            </div>

            <MonthlyTimeCalendar
                onDayClick={(day, entries) => {
                    console.log('Dia clicado:', day, 'Entradas:', entries);
                }}
            />

            {error && (
                <div className="mb-6 bg-red-50 border-l-4 border-red-500 p-4 dark:bg-red-900/20 dark:border-red-700">
                    <div className="flex">
                        <div className="flex-shrink-0">
                            <FiAlertCircle className="h-5 w-5 text-red-500" />
                        </div>
                        <div className="ml-3">
                            <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
                        </div>
                    </div>
                </div>
            )}

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
                <StatCard
                    title="Horas Logadas (Mês Atual)"
                    icon={<FiClock className="w-6 h-6 text-white" />}
                    value={formatarHoras(dashboardData.horasLogadas)}
                    description="Comparado ao mês anterior"
                    change={dashboardData.horasLogadasChange}
                    className="bg-blue-500"
                />
                <StatCard
                    title="Tarefas Pendentes"
                    icon={<FiList className="w-6 h-6 text-white" />}
                    value={dashboardData.tarefasPendentes || tasksData.length}
                    description="Tarefas ativas"
                    className="bg-amber-500"
                />
                <StatCard
                    title="Projetos Ativos"
                    icon={<FiFileText className="w-6 h-6 text-white" />}
                    value={dashboardData.projetos || projectsData.length}
                    description="Projetos com atividade"
                    className="bg-purple-500"
                />
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
                <div className="card">
                    <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Meta do Mês</h2>
                    <div className="p-4 bg-blue-50 rounded-lg dark:bg-blue-900/20">
                        <div className="mb-4">
                            <div className="flex justify-between text-sm text-blue-700 dark:text-blue-400 mb-2">
                                <span className="font-medium">Progresso: {formatarHoras(dashboardData.horasLogadas)}</span>
                                <span className="font-medium">Meta: 168h</span>
                            </div>
                            <div className="w-full bg-blue-200 rounded-full h-3 dark:bg-blue-700">
                                <div
                                    className="bg-blue-600 h-3 rounded-full dark:bg-blue-500"
                                    style={{ width: `${Math.min(100, (dashboardData.horasLogadas / 168) * 100)}%` }}
                                ></div>
                            </div>
                        </div>
                        <p className="text-sm text-blue-700 dark:text-blue-400 text-center font-medium">
                            {Math.floor((dashboardData.horasLogadas / 168) * 100)}% da meta mensal atingida
                        </p>
                    </div>

                    <div className="mt-4 p-4 bg-gray-50 rounded-lg dark:bg-gray-700">
                        <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">Status do Mês</h3>
                        <div className="grid grid-cols-3 gap-4">
                            <div className="text-center">
                                <p className="text-sm text-gray-500 dark:text-gray-400">Dias úteis</p>
                                <p className="text-xl font-bold text-gray-900 dark:text-white">{dashboardData.diasUteisMes || 0}</p>
                            </div>
                            <div className="text-center">
                                <p className="text-sm text-gray-500 dark:text-gray-400">Dias passados</p>
                                <p className="text-xl font-bold text-gray-900 dark:text-white">{dashboardData.diasUteisPassados || 0}</p>
                            </div>
                            <div className="text-center">
                                <p className="text-sm text-gray-500 dark:text-gray-400">Dias restantes</p>
                                <p className="text-xl font-bold text-gray-900 dark:text-white">{dashboardData.diasUteisRestantes || 0}</p>
                            </div>
                        </div>
                    </div>
                </div>

                <div className="card">
                    <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Acesso Rápido</h2>
                    <div className="grid grid-cols-2 gap-4">
                        <button
                            onClick={() => navigateTo('/timelog')}
                            className="flex items-center p-4 bg-gray-50 hover:bg-gray-100 rounded-lg dark:bg-gray-700 dark:hover:bg-gray-600"
                        >
                            <FiClock className="w-6 h-6 text-primary-600 mr-3" />
                            <span className="font-medium">Lançar Horas</span>
                        </button>
                        <button
                            onClick={() => navigateTo('/tasks')}
                            className="flex items-center p-4 bg-gray-50 hover:bg-gray-100 rounded-lg dark:bg-gray-700 dark:hover:bg-gray-600"
                        >
                            <FiList className="w-6 h-6 text-primary-600 mr-3" />
                            <span className="font-medium">Ver Tarefas</span>
                        </button>
                        <button
                            onClick={() => navigateTo('/templates')}
                            className="flex items-center p-4 bg-gray-50 hover:bg-gray-100 rounded-lg dark:bg-gray-700 dark:hover:bg-gray-600"
                        >
                            <FiClipboard className="w-6 h-6 text-primary-600 mr-3" />
                            <span className="font-medium">Templates</span>
                        </button>
                        <button
                            onClick={() => setIsTimeManagerOpen(true)}
                            className="flex items-center p-4 bg-gray-50 hover:bg-gray-100 rounded-lg dark:bg-gray-700 dark:hover:bg-gray-600"
                        >
                            <FiTrash2 className="w-6 h-6 text-red-600 mr-3" />
                            <span className="font-medium">Gerenciar Horas</span>
                        </button>
                        <button
                            onClick={handleExportReport}
                            disabled={isExporting}
                            className="flex items-center p-4 bg-gray-50 hover:bg-gray-100 rounded-lg dark:bg-gray-700 dark:hover:bg-gray-600 disabled:opacity-50"
                        >
                            {isExporting ? (
                                <FiLoader className="w-6 h-6 text-primary-600 mr-3 animate-spin" />
                            ) : (
                                <FiDownload className="w-6 h-6 text-primary-600 mr-3" />
                            )}
                            <span className="font-medium">{isExporting ? "Exportando..." : "Relatório Mensal"}</span>
                        </button>
                        <button
                            onClick={() => setIsReportModalOpen(true)}
                            disabled={isExporting}
                            className="flex items-center p-4 bg-gray-50 hover:bg-gray-100 rounded-lg dark:bg-gray-700 dark:hover:bg-gray-600 disabled:opacity-50"
                        >
                            <FiCalendar className="w-6 h-6 text-primary-600 mr-3" />
                            <span className="font-medium">Relatório Personalizado</span>
                        </button>
                    </div>

                    <div className="mt-6 text-center text-sm text-gray-500 dark:text-gray-400">
                        Última atualização: {lastUpdate.toLocaleString()}
                    </div>
                </div>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="card">
                    <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                        <FiClock className="w-5 h-5 mr-2" />
                        Atividades Recentes
                    </h2>
                    {recentActivities.length === 0 ? (
                        <p className="text-center py-6 text-gray-500 dark:text-gray-400">
                            Nenhuma atividade recente encontrada.
                        </p>
                    ) : (
                        <div className="divide-y divide-gray-200 dark:divide-gray-700">
                            {recentActivities.map((activity, index) => (
                                <ActivityItem key={index} activity={activity} />
                            ))}
                        </div>
                    )}
                </div>

                <div className="card">
                    <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                        <FiCalendar className="w-5 h-5 mr-2" />
                        Tarefas com Vencimento Próximo
                    </h2>
                    {upcomingTasks.length === 0 ? (
                        <p className="text-center py-6 text-gray-500 dark:text-gray-400">
                            Nenhuma tarefa com vencimento próximo.
                        </p>
                    ) : (
                        <div className="divide-y divide-gray-200 dark:divide-gray-700">
                            {upcomingTasks.map((task, index) => (
                                <UpcomingTaskItem key={index} task={task} />
                            ))}
                        </div>
                    )}
                </div>
            </div>

            <ReportPeriodModal
                isOpen={isReportModalOpen}
                onClose={() => setIsReportModalOpen(false)}
                onExport={handleExportCustomPeriodReport}
            />

            <TimeEntryManager
                isOpen={isTimeManagerOpen}
                onClose={handleTimeManagerClose}
                onEntriesDeleted={() => {
                    loadDashboard();
                    toast.success('Dados atualizados após deleção das entradas.');
                }}
            />
        </div>
    );
};

export default Dashboard;