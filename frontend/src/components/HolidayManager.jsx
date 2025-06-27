import React, { useState, useEffect } from 'react';
import { toast } from 'react-toastify';
import {
    FiCalendar,
    FiRefreshCw,
    FiDatabase,
    FiInfo,
    FiTrash2,
    FiDownload,
    FiCheck,
    FiAlertCircle
} from 'react-icons/fi';

const HolidayManager = ({ isOpen, onClose }) => {
    const [loading, setLoading] = useState(false);
    const [cacheStats, setCacheStats] = useState(null);
    const [selectedYear, setSelectedYear] = useState(new Date().getFullYear());
    const [holidays, setHolidays] = useState([]);
    const [preloading, setPreloading] = useState(false);

    useEffect(() => {
        if (isOpen) {
            loadCacheStats();
        }
    }, [isOpen]);

    const loadCacheStats = async () => {
        try {
            setLoading(true);
            const stats = await window.go.backend.App.GetHolidayCacheStats();
            setCacheStats(stats);
        } catch (error) {
            console.error('Erro ao carregar estatísticas do cache:', error);
            toast.error('Erro ao carregar estatísticas do cache');
        } finally {
            setLoading(false);
        }
    };

    const loadHolidaysForYear = async (year) => {
        try {
            setLoading(true);
            const holidaysData = await window.go.backend.App.GetBrazilianHolidays(year);

            // Converter map para array e ordenar por data
            const holidaysArray = Object.values(holidaysData).sort((a, b) =>
                new Date(a.date) - new Date(b.date)
            );

            setHolidays(holidaysArray);
            toast.success(`${holidaysArray.length} feriados carregados para ${year}`);
        } catch (error) {
            console.error('Erro ao carregar feriados:', error);
            toast.error('Erro ao carregar feriados: ' + error.message);
        } finally {
            setLoading(false);
        }
    };

    const refreshYear = async (year) => {
        try {
            setLoading(true);
            const holidaysData = await window.go.backend.App.RefreshHolidaysForYear(year);

            const holidaysArray = Object.values(holidaysData).sort((a, b) =>
                new Date(a.date) - new Date(b.date)
            );

            setHolidays(holidaysArray);
            await loadCacheStats();
            toast.success(`Feriados atualizados para ${year}`);
        } catch (error) {
            console.error('Erro ao atualizar feriados:', error);
            toast.error('Erro ao atualizar feriados: ' + error.message);
        } finally {
            setLoading(false);
        }
    };

    const clearCache = async () => {
        if (!window.confirm('Tem certeza que deseja limpar todo o cache de feriados?')) {
            return;
        }

        try {
            await window.go.backend.App.ClearHolidayCache();
            await loadCacheStats();
            setHolidays([]);
            toast.success('Cache de feriados limpo com sucesso');
        } catch (error) {
            console.error('Erro ao limpar cache:', error);
            toast.error('Erro ao limpar cache');
        }
    };

    const preloadHolidays = async () => {
        try {
            setPreloading(true);
            await window.go.backend.App.PreloadHolidays();
            await loadCacheStats();
            toast.success('Feriados pré-carregados com sucesso');
        } catch (error) {
            console.error('Erro ao pré-carregar feriados:', error);
            toast.error('Erro ao pré-carregar feriados');
        } finally {
            setPreloading(false);
        }
    };

    const formatDate = (dateStr) => {
        try {
            return new Date(dateStr).toLocaleDateString('pt-BR', {
                day: '2-digit',
                month: 'long',
                weekday: 'long'
            });
        } catch {
            return dateStr;
        }
    };

    const getStatusColor = (isExpired) => {
        return isExpired
            ? 'text-red-600 dark:text-red-400'
            : 'text-green-600 dark:text-green-400';
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-6xl mx-4 max-h-[90vh] overflow-hidden">
                {/* Header */}
                <div className="flex justify-between items-center p-6 border-b border-gray-200 dark:border-gray-700">
                    <h2 className="text-xl font-semibold text-gray-900 dark:text-white flex items-center">
                        <FiCalendar className="w-6 h-6 mr-2" />
                        Gerenciamento de Feriados
                    </h2>
                    <button
                        onClick={onClose}
                        className="text-gray-400 hover:text-gray-500 dark:hover:text-gray-300"
                    >
                        <FiX className="w-6 h-6" />
                    </button>
                </div>

                <div className="p-6 overflow-y-auto max-h-[calc(90vh-8rem)]">
                    {/* Controles principais */}
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                    Selecionar Ano
                                </label>
                                <select
                                    value={selectedYear}
                                    onChange={(e) => setSelectedYear(parseInt(e.target.value))}
                                    className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                                >
                                    {Array.from({ length: 10 }, (_, i) => {
                                        const year = new Date().getFullYear() - 2 + i;
                                        return (
                                            <option key={year} value={year}>
                                                {year}
                                            </option>
                                        );
                                    })}
                                </select>
                            </div>

                            <button
                                onClick={() => loadHolidaysForYear(selectedYear)}
                                disabled={loading}
                                className="w-full flex items-center justify-center px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg disabled:opacity-50"
                            >
                                {loading ? (
                                    <FiRefreshCw className="w-4 h-4 mr-2 animate-spin" />
                                ) : (
                                    <FiDownload className="w-4 h-4 mr-2" />
                                )}
                                Carregar Feriados
                            </button>
                        </div>

                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                    Ações de Cache
                                </label>
                                <div className="space-y-2">
                                    <button
                                        onClick={() => refreshYear(selectedYear)}
                                        disabled={loading}
                                        className="w-full flex items-center justify-center px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg disabled:opacity-50"
                                    >
                                        <FiRefreshCw className="w-4 h-4 mr-2" />
                                        Atualizar Ano
                                    </button>

                                    <button
                                        onClick={preloadHolidays}
                                        disabled={preloading}
                                        className="w-full flex items-center justify-center px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg disabled:opacity-50"
                                    >
                                        {preloading ? (
                                            <FiRefreshCw className="w-4 h-4 mr-2 animate-spin" />
                                        ) : (
                                            <FiDatabase className="w-4 h-4 mr-2" />
                                        )}
                                        Pré-carregar
                                    </button>
                                </div>
                            </div>
                        </div>

                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                    Limpeza
                                </label>
                                <button
                                    onClick={clearCache}
                                    className="w-full flex items-center justify-center px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg"
                                >
                                    <FiTrash2 className="w-4 h-4 mr-2" />
                                    Limpar Cache
                                </button>
                            </div>

                            <button
                                onClick={loadCacheStats}
                                disabled={loading}
                                className="w-full flex items-center justify-center px-4 py-2 bg-gray-600 hover:bg-gray-700 text-white rounded-lg disabled:opacity-50"
                            >
                                <FiInfo className="w-4 h-4 mr-2" />
                                Atualizar Stats
                            </button>
                        </div>
                    </div>

                    {/* Estatísticas do Cache */}
                    {cacheStats && (
                        <div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-4 mb-6">
                            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-3 flex items-center">
                                <FiDatabase className="w-5 h-5 mr-2" />
                                Estatísticas do Cache
                            </h3>

                            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
                                <div className="bg-white dark:bg-gray-800 p-3 rounded">
                                    <p className="text-sm text-gray-500 dark:text-gray-400">Anos em Cache</p>
                                    <p className="text-2xl font-bold text-primary-600 dark:text-primary-400">
                                        {cacheStats.cached_years}
                                    </p>
                                </div>

                                <div className="bg-white dark:bg-gray-800 p-3 rounded">
                                    <p className="text-sm text-gray-500 dark:text-gray-400">Anos Disponíveis</p>
                                    <p className="text-sm text-gray-900 dark:text-white">
                                        {cacheStats.years?.join(', ') || 'Nenhum'}
                                    </p>
                                </div>

                                <div className="bg-white dark:bg-gray-800 p-3 rounded">
                                    <p className="text-sm text-gray-500 dark:text-gray-400">Total de Feriados</p>
                                    <p className="text-2xl font-bold text-green-600 dark:text-green-400">
                                        {Object.values(cacheStats.cache_details || {}).reduce((total, details) =>
                                            total + (details.holidays_count || 0), 0
                                        )}
                                    </p>
                                </div>
                            </div>

                            {/* Detalhes por ano */}
                            {cacheStats.cache_details && Object.keys(cacheStats.cache_details).length > 0 && (
                                <div className="overflow-x-auto">
                                    <table className="w-full text-sm">
                                        <thead>
                                        <tr className="border-b border-gray-200 dark:border-gray-600">
                                            <th className="text-left py-2 px-3">Ano</th>
                                            <th className="text-left py-2 px-3">Feriados</th>
                                            <th className="text-left py-2 px-3">Fontes</th>
                                            <th className="text-left py-2 px-3">Cachado em</th>
                                            <th className="text-left py-2 px-3">Expira em</th>
                                            <th className="text-left py-2 px-3">Status</th>
                                        </tr>
                                        </thead>
                                        <tbody>
                                        {Object.entries(cacheStats.cache_details).map(([year, details]) => (
                                            <tr key={year} className="border-b border-gray-100 dark:border-gray-700">
                                                <td className="py-2 px-3 font-medium">{year}</td>
                                                <td className="py-2 px-3">{details.holidays_count}</td>
                                                <td className="py-2 px-3">
                                                        <span className="text-xs bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 px-2 py-1 rounded">
                                                            {details.sources?.join(', ') || 'N/A'}
                                                        </span>
                                                </td>
                                                <td className="py-2 px-3 text-xs">
                                                    {new Date(details.cached_at).toLocaleString('pt-BR')}
                                                </td>
                                                <td className="py-2 px-3 text-xs">
                                                    {new Date(details.expires_at).toLocaleString('pt-BR')}
                                                </td>
                                                <td className="py-2 px-3">
                                                        <span className={`inline-flex items-center text-xs ${getStatusColor(details.is_expired)}`}>
                                                            {details.is_expired ? (
                                                                <>
                                                                    <FiAlertCircle className="w-3 h-3 mr-1" />
                                                                    Expirado
                                                                </>
                                                            ) : (
                                                                <>
                                                                    <FiCheck className="w-3 h-3 mr-1" />
                                                                    Válido
                                                                </>
                                                            )}
                                                        </span>
                                                </td>
                                            </tr>
                                        ))}
                                        </tbody>
                                    </table>
                                </div>
                            )}
                        </div>
                    )}

                    {/* Lista de Feriados */}
                    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
                        <div className="p-4 border-b border-gray-200 dark:border-gray-700">
                            <h3 className="text-lg font-medium text-gray-900 dark:text-white flex items-center">
                                <FiCalendar className="w-5 h-5 mr-2" />
                                Feriados de {selectedYear}
                                {holidays.length > 0 && (
                                    <span className="ml-2 px-2 py-1 text-xs bg-primary-100 dark:bg-primary-900 text-primary-800 dark:text-primary-200 rounded-full">
                                        {holidays.length} feriados
                                    </span>
                                )}
                            </h3>
                        </div>

                        <div className="p-4">
                            {loading ? (
                                <div className="flex justify-center items-center py-8">
                                    <div className="animate-spin w-8 h-8 border-4 border-primary-600 border-t-transparent rounded-full"></div>
                                </div>
                            ) : holidays.length === 0 ? (
                                <div className="text-center py-8">
                                    <FiCalendar className="w-12 h-12 mx-auto text-gray-400 dark:text-gray-600 mb-4" />
                                    <p className="text-gray-500 dark:text-gray-400">
                                        Nenhum feriado carregado. Selecione um ano e clique em "Carregar Feriados".
                                    </p>
                                </div>
                            ) : (
                                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                                    {holidays.map((holiday, index) => (
                                        <div
                                            key={index}
                                            className="border border-gray-200 dark:border-gray-600 rounded-lg p-4 hover:shadow-md transition-shadow"
                                        >
                                            <div className="flex items-start justify-between mb-2">
                                                <h4 className="font-medium text-gray-900 dark:text-white">
                                                    {holiday.name}
                                                </h4>
                                                {holiday.isOptional && (
                                                    <span className="text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 px-2 py-1 rounded">
                                                        Opcional
                                                    </span>
                                                )}
                                            </div>

                                            <div className="space-y-1 text-sm text-gray-600 dark:text-gray-400">
                                                <p className="flex items-center">
                                                    <FiCalendar className="w-4 h-4 mr-2" />
                                                    {formatDate(holiday.date)}
                                                </p>

                                                <p className="flex items-center">
                                                    <FiInfo className="w-4 h-4 mr-2" />
                                                    Tipo: {holiday.type}
                                                </p>

                                                {holiday.source && (
                                                    <p className="flex items-center">
                                                        <FiDatabase className="w-4 h-4 mr-2" />
                                                        Fonte: {holiday.source}
                                                    </p>
                                                )}

                                                {holiday.description && (
                                                    <p className="text-xs text-gray-500 dark:text-gray-500 mt-2">
                                                        {holiday.description}
                                                    </p>
                                                )}
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>
                    </div>
                </div>

                {/* Footer */}
                <div className="px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex justify-between items-center">
                    <div className="text-sm text-gray-500 dark:text-gray-400">
                        <p>Sistema de feriados com cache inteligente e múltiplas fontes de dados</p>
                    </div>
                    <button
                        onClick={onClose}
                        className="px-4 py-2 bg-gray-600 hover:bg-gray-700 text-white rounded-md"
                    >
                        Fechar
                    </button>
                </div>
            </div>
        </div>
    );
};

export default HolidayManager;