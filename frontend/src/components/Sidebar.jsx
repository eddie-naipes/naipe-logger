import React, { useContext, useState } from 'react';
import { NavLink } from 'react-router-dom';
import {
    FiHome,
    FiSettings,
    FiList,
    FiClock,
    FiSave,
    FiX,
    FiSun,
    FiMoon,
    FiFileText,
    FiLoader,
    FiDownload
} from 'react-icons/fi';
import { toast } from 'react-toastify';
import { ThemeContext } from '../contexts/ThemeContext';
import clsx from 'clsx';

const Sidebar = ({ isOpen, onClose, isConfigured }) => {
    const { darkMode, toggleDarkMode } = useContext(ThemeContext);
    const [isExporting, setIsExporting] = useState(false);

    const navItems = [
        {
            to: '/',
            icon: <FiHome className="w-5 h-5" />,
            label: 'Dashboard',
            requiresConfig: true
        },
        {
            to: '/tasks',
            icon: <FiList className="w-5 h-5" />,
            label: 'Tarefas',
            requiresConfig: true
        },
        {
            to: '/timelog',
            icon: <FiClock className="w-5 h-5" />,
            label: 'Lançar Horas',
            requiresConfig: true
        },
        {
            to: '/templates',
            icon: <FiSave className="w-5 h-5" />,
            label: 'Templates',
            requiresConfig: true
        },
        {
            to: '/config',
            icon: <FiSettings className="w-5 h-5" />,
            label: 'Configurações',
            requiresConfig: false
        },
    ];

    const filteredNavItems = isConfigured
        ? navItems
        : navItems.filter(item => !item.requiresConfig);

    const handleExportReport = async () => {
        if (!isConfigured) {
            toast.warning("Configure sua conta antes de exportar relatórios.");
            return;
        }

        setIsExporting(true);

        try {
            const filePath = await window.go.backend.App.DownloadCurrentMonthTimeReport();
            toast.success(`Relatório exportado com sucesso para: ${filePath}`);
            await window.go.backend.App.OpenDirectoryPath(filePath);
        } catch (error) {
            console.error("Erro ao exportar relatório:", error);
            toast.error("Não foi possível exportar o relatório: " + (error.message || "Erro desconhecido"));
        } finally {
            setIsExporting(false);
        }
    };

    return (
        <>
            {isOpen && (
                <div
                    className="fixed inset-0 z-10 bg-gray-900 bg-opacity-50 lg:hidden"
                    onClick={onClose}
                />
            )}

            <aside
                className={clsx(
                    "fixed top-0 left-0 z-20 h-full w-64 transform bg-white border-r border-gray-200 transition-transform duration-300 ease-in-out lg:translate-x-0 lg:static lg:h-full dark:bg-gray-800 dark:border-gray-700",
                    isOpen ? 'translate-x-0' : '-translate-x-full'
                )}
            >
                <div className="flex flex-col h-full">
                    <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
                        <h1 className="text-xl font-semibold text-gray-800 dark:text-white">
                            Teamwork Logger
                        </h1>
                        <button
                            onClick={onClose}
                            className="p-1 rounded-full hover:bg-gray-100 dark:hover:bg-gray-700 lg:hidden"
                        >
                            <FiX className="w-5 h-5 text-gray-500 dark:text-gray-400" />
                        </button>
                    </div>

                    <nav className="flex-1 p-4 space-y-1 overflow-y-auto">
                        {filteredNavItems.map((item) => (
                            <NavLink
                                key={item.to}
                                to={item.to}
                                className={({ isActive }) => clsx(
                                    "flex items-center px-4 py-3 rounded-lg transition-colors",
                                    isActive
                                        ? "bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-100"
                                        : "text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700"
                                )}
                            >
                                {item.icon}
                                <span className="ml-3">{item.label}</span>
                            </NavLink>
                        ))}

                        {isConfigured && (
                            <button
                                onClick={handleExportReport}
                                disabled={isExporting}
                                className="flex items-center w-full px-4 py-3 rounded-lg transition-colors text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700"
                            >
                                {isExporting ? (
                                    <FiLoader className="w-5 h-5 animate-spin" />
                                ) : (
                                    <FiDownload className="w-5 h-5" />
                                )}
                                <span className="ml-3">
                                    {isExporting ? "Exportando..." : "Relatório Mensal"}
                                </span>
                            </button>
                        )}
                    </nav>

                    <div className="p-4 border-t border-gray-200 dark:border-gray-700">
                        <button
                            onClick={toggleDarkMode}
                            className="flex items-center w-full px-4 py-2 text-gray-700 rounded-lg hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700"
                        >
                            {darkMode ? (
                                <>
                                    <FiSun className="w-5 h-5" />
                                    <span className="ml-3">Modo Claro</span>
                                </>
                            ) : (
                                <>
                                    <FiMoon className="w-5 h-5" />
                                    <span className="ml-3">Modo Escuro</span>
                                </>
                            )}
                        </button>
                    </div>
                </div>
            </aside>
        </>
    );
};

export default Sidebar;