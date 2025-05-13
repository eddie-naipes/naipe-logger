import React, { useState, useEffect } from 'react';
import { FiClock, FiList, FiHardDrive, FiCalendar } from 'react-icons/fi';
import { format } from 'date-fns';
import { ptBR } from 'date-fns/locale';

// Widgets do Dashboard
const DashboardWidget = ({ title, icon, value, description, className }) => {
    return (
        <div className="card flex items-start space-x-4">
            <div className={`flex items-center justify-center w-12 h-12 rounded-lg ${className}`}>
                {icon}
            </div>
            <div>
                <h3 className="text-gray-500 dark:text-gray-400 text-sm font-medium">{title}</h3>
                <p className="text-3xl font-bold text-gray-900 dark:text-white">{value}</p>
                <p className="text-sm text-gray-500 dark:text-gray-400">{description}</p>
            </div>
        </div>
    );
};

const Dashboard = () => {
    const [isLoading, setIsLoading] = useState(true);
    const [stats, setStats] = useState({
        tarefasPendentes: 0,
        horasLogadas: 0,
        ultimoLancamento: null,
        templates: 0
    });

    useEffect(() => {
        const loadDashboard = async () => {
            try {
                // Obter estatísticas básicas
                // Quando temos acesso ao backend:
                // const tasks = await window.go.backend.App.GetTasks();
                // const templates = await window.go.backend.App.GetTemplates();

                // Simulando dados para exemplo:
                setTimeout(() => {
                    setStats({
                        tarefasPendentes: 12,
                        horasLogadas: 64.5,
                        ultimoLancamento: new Date(),
                        templates: 3
                    });
                    setIsLoading(false);
                }, 1000);
            } catch (error) {
                console.error("Erro ao carregar dashboard:", error);
                setIsLoading(false);
            }
        };

        loadDashboard();
    }, []);

    // Funções auxiliares
    const formatarDataHora = (data) => {
        if (!data) return 'Nenhum lançamento';
        return format(new Date(data), 'dd MMM yyyy, HH:mm', { locale: ptBR });
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
            <div className="mb-6">
                <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">Dashboard</h1>
                <p className="text-gray-600 dark:text-gray-400">Resumo das suas atividades no Teamwork</p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
                <DashboardWidget
                    title="Tarefas Pendentes"
                    icon={<FiList className="w-6 h-6 text-white" />}
                    value={stats.tarefasPendentes}
                    description="Tarefas ativas atribuídas a você"
                    className="bg-blue-500"
                />
                <DashboardWidget
                    title="Horas Logadas"
                    icon={<FiClock className="w-6 h-6 text-white" />}
                    value={`${stats.horasLogadas}h`}
                    description="Total de horas no mês atual"
                    className="bg-green-500"
                />
                <DashboardWidget
                    title="Último Lançamento"
                    icon={<FiCalendar className="w-6 h-6 text-white" />}
                    value={formatarDataHora(stats.ultimoLancamento)}
                    description="Data do último registro de horas"
                    className="bg-purple-500"
                />
                <DashboardWidget
                    title="Templates Salvos"
                    icon={<FiHardDrive className="w-6 h-6 text-white" />}
                    value={stats.templates}
                    description="Templates de lançamento disponíveis"
                    className="bg-amber-500"
                />
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="card">
                    <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Atividade Recente</h2>
                    <p className="text-gray-600 dark:text-gray-400">
                        Seus lançamentos recentes aparecerão aqui quando você começar a registrar horas.
                    </p>
                </div>

                <div className="card">
                    <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Acesso Rápido</h2>
                    <div className="grid grid-cols-2 gap-4">
                        <a
                            href="#/timelog"
                            className="flex items-center p-3 bg-gray-50 hover:bg-gray-100 rounded-lg dark:bg-gray-700 dark:hover:bg-gray-600"
                        >
                            <FiClock className="w-6 h-6 text-primary-600 mr-3" />
                            <span>Lançar Horas</span>
                        </a>
                        <a
                            href="#/tasks"
                            className="flex items-center p-3 bg-gray-50 hover:bg-gray-100 rounded-lg dark:bg-gray-700 dark:hover:bg-gray-600"
                        >
                            <FiList className="w-6 h-6 text-primary-600 mr-3" />
                            <span>Ver Tarefas</span>
                        </a>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default Dashboard;