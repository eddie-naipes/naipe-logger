// src/components/TimeEntryManager.jsx - atualizar para usar novos endpoints
import React, { useState, useEffect } from 'react';
import { toast } from 'react-toastify';
import {
    FiTrash2,
    FiLoader,
    FiAlertCircle,
    FiCheck,
    FiClock,
    FiCalendar,
    FiRefreshCw,
    FiFilter,
    FiX,
    FiEye,
    FiEyeOff
} from 'react-icons/fi';
import { format, parseISO } from 'date-fns';
import { ptBR } from 'date-fns/locale';

const TimeEntryManager = ({ isOpen, onClose, onEntriesDeleted }) => {
    const [entries, setEntries] = useState([]);
    const [filteredEntries, setFilteredEntries] = useState([]);
    const [selectedEntries, setSelectedEntries] = useState([]);
    const [loading, setLoading] = useState(false);
    const [deleting, setDeleting] = useState(false);
    const [dateRange, setDateRange] = useState({
        startDate: format(new Date(), 'yyyy-MM-dd'),
        endDate: format(new Date(), 'yyyy-MM-dd')
    });
    const [filters, setFilters] = useState({
        projectName: '',
        taskName: '',
        minHours: '',
        maxHours: '',
        isBillable: 'all',
        status: 'active'
    });
    const [deleteResults, setDeleteResults] = useState([]);
    const [showResults, setShowResults] = useState(false);
    const [showDeleted, setShowDeleted] = useState(false);

    useEffect(() => {
        if (isOpen) {
            loadTimeEntries();
        }
    }, [isOpen, dateRange, showDeleted]);

    useEffect(() => {
        applyFilters();
    }, [entries, filters]);

    const loadTimeEntries = async () => {
        setLoading(true);
        try {
            const timeEntries = await window.go.backend.App.GetTimeEntriesForPeriodV2(
                dateRange.startDate,
                dateRange.endDate,
                showDeleted
            );

            setEntries(timeEntries || []);
            setSelectedEntries([]);
            setShowResults(false);
        } catch (error) {
            console.error('Erro ao carregar entradas de tempo:', error);
            toast.error('Erro ao carregar entradas de tempo: ' + (error.message || 'Erro desconhecido'));
        } finally {
            setLoading(false);
        }
    };

    const applyFilters = () => {
        let filtered = [...entries];

        if (filters.projectName) {
            filtered = filtered.filter(entry =>
                entry.projectName?.toLowerCase().includes(filters.projectName.toLowerCase())
            );
        }

        if (filters.taskName) {
            filtered = filtered.filter(entry =>
                entry.taskName?.toLowerCase().includes(filters.taskName.toLowerCase())
            );
        }

        if (filters.minHours) {
            const minMinutes = parseFloat(filters.minHours) * 60;
            filtered = filtered.filter(entry => entry.minutes >= minMinutes);
        }

        if (filters.maxHours) {
            const maxMinutes = parseFloat(filters.maxHours) * 60;
            filtered = filtered.filter(entry => entry.minutes <= maxMinutes);
        }

        if (filters.isBillable !== 'all') {
            const isBillable = filters.isBillable === 'true';
            filtered = filtered.filter(entry => entry.isBillable === isBillable);
        }

        if (filters.status !== 'all') {
            if (filters.status === 'active') {
                filtered = filtered.filter(entry => !entry.status || entry.status !== 'deleted');
            } else if (filters.status === 'deleted') {
                filtered = filtered.filter(entry => entry.status === 'deleted');
            }
        }

        setFilteredEntries(filtered);
    };

    const toggleEntrySelection = (entryId) => {
        if (selectedEntries.includes(entryId)) {
            setSelectedEntries(selectedEntries.filter(id => id !== entryId));
        } else {
            setSelectedEntries([...selectedEntries, entryId]);
        }
    };

    const selectAllEntries = () => {
        const selectableEntries = filteredEntries.filter(entry => !entry.status || entry.status !== 'deleted');
        if (selectedEntries.length === selectableEntries.length) {
            setSelectedEntries([]);
        } else {
            setSelectedEntries(selectableEntries.map(entry => entry.id));
        }
    };

    const deleteSelectedEntries = async () => {
        const activeEntries = selectedEntries.filter(id => {
            const entry = filteredEntries.find(e => e.id === id);
            return entry && (!entry.status || entry.status !== 'deleted');
        });

        if (activeEntries.length === 0) {
            toast.warning('Selecione pelo menos uma entrada ativa para deletar.');
            return;
        }

        const confirmMessage = `Tem certeza que deseja deletar ${activeEntries.length} entrada(s) de tempo?\n\nEsta ação não pode ser desfeita.`;

        if (!window.confirm(confirmMessage)) {
            return;
        }

        setDeleting(true);
        setShowResults(false);

        try {
            const results = [];

            for (const entryId of activeEntries) {
                try {
                    await window.go.backend.App.DeleteTimeEntryV2(entryId);
                    results.push({
                        entryId: entryId,
                        success: true,
                        message: 'Entrada deletada com sucesso'
                    });
                } catch (error) {
                    results.push({
                        entryId: entryId,
                        success: false,
                        message: error.message || 'Erro ao deletar entrada'
                    });
                }

                await new Promise(resolve => setTimeout(resolve, 300));
            }

            setDeleteResults(results);
            setShowResults(true);

            const successCount = results.filter(r => r.success).length;
            const failureCount = results.length - successCount;

            if (failureCount === 0) {
                toast.success(`${successCount} entradas deletadas com sucesso!`);
            } else if (successCount === 0) {
                toast.error(`Falha ao deletar todas as ${failureCount} entradas.`);
            } else {
                toast.warning(`${successCount} entradas deletadas, ${failureCount} falharam.`);
            }

            await loadTimeEntries();

            if (onEntriesDeleted) {
                onEntriesDeleted();
            }

        } catch (error) {
            console.error('Erro ao deletar entradas:', error);
            toast.error('Erro ao deletar entradas: ' + (error.message || 'Erro desconhecido'));
        } finally {
            setDeleting(false);
        }
    };

    const formatDate = (dateString) => {
        try {
            return format(parseISO(dateString), 'dd/MM/yyyy', { locale: ptBR });
        } catch {
            return dateString;
        }
    };

    const formatTime = (timeString) => {
        if (!timeString) return '';
        try {
            return timeString.substring(0, 5);
        } catch {
            return timeString;
        }
    };

    const getTotalHours = () => {
        return filteredEntries
            .filter(entry => !entry.status || entry.status !== 'deleted')
            .reduce((sum, entry) => sum + (entry.minutes || 0), 0) / 60;
    };

    const getTotalBillableHours = () => {
        return filteredEntries
            .filter(entry => entry.isBillable && (!entry.status || entry.status !== 'deleted'))
            .reduce((sum, entry) => sum + (entry.minutes || 0), 0) / 60;
    };

    const getDeletedCount = () => {
        return filteredEntries.filter(entry => entry.status === 'deleted').length;
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-7xl mx-4 max-h-[90vh] overflow-hidden">
                <div className="flex justify-between items-center p-6 border-b border-gray-200 dark:border-gray-700">
                    <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                        Gerenciar Apontamentos de Horas
                    </h2>
                    <button
                        onClick={onClose}
                        className="text-gray-400 hover:text-gray-500 dark:hover:text-gray-300"
                    >
                        <FiX className="w-6 h-6" />
                    </button>
                </div>

                <div className="p-6">
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Data Inicial
                            </label>
                            <input
                                type="date"
                                value={dateRange.startDate}
                                onChange={(e) => setDateRange({...dateRange, startDate: e.target.value})}
                                className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                            />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Data Final
                            </label>
                            <input
                                type="date"
                                value={dateRange.endDate}
                                onChange={(e) => setDateRange({...dateRange, endDate: e.target.value})}
                                className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                            />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Incluir Deletados
                            </label>
                            <button
                                onClick={() => setShowDeleted(!showDeleted)}
                                className={`flex items-center justify-center w-full p-2.5 rounded-lg border text-sm ${
                                    showDeleted
                                        ? 'bg-red-50 border-red-300 text-red-700 dark:bg-red-900/20 dark:border-red-700 dark:text-red-300'
                                        : 'bg-gray-50 border-gray-300 text-gray-900 dark:bg-gray-700 dark:border-gray-600 dark:text-white'
                                }`}
                            >
                                {showDeleted ? <FiEye className="w-4 h-4 mr-2" /> : <FiEyeOff className="w-4 h-4 mr-2" />}
                                {showDeleted ? 'Mostrando Deletados' : 'Ocultar Deletados'}
                            </button>
                        </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-5 gap-4 mb-6">
                        <input
                            type="text"
                            placeholder="Filtrar por projeto..."
                            value={filters.projectName}
                            onChange={(e) => setFilters({...filters, projectName: e.target.value})}
                            className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                        />
                        <input
                            type="text"
                            placeholder="Filtrar por tarefa..."
                            value={filters.taskName}
                            onChange={(e) => setFilters({...filters, taskName: e.target.value})}
                            className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                        />
                        <input
                            type="number"
                            placeholder="Min horas"
                            step="0.5"
                            value={filters.minHours}
                            onChange={(e) => setFilters({...filters, minHours: e.target.value})}
                            className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                        />
                        <input
                            type="number"
                            placeholder="Max horas"
                            step="0.5"
                            value={filters.maxHours}
                            onChange={(e) => setFilters({...filters, maxHours: e.target.value})}
                            className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                        />
                        <select
                            value={filters.isBillable}
                            onChange={(e) => setFilters({...filters, isBillable: e.target.value})}
                            className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                        >
                            <option value="all">Todos</option>
                            <option value="true">Contabilizável</option>
                            <option value="false">Não Contabilizável</option>
                        </select>
                    </div>

                    <div className="flex justify-between items-center mb-4">
                        <div className="flex items-center space-x-4">
                            <button
                                onClick={loadTimeEntries}
                                disabled={loading}
                                className="flex items-center px-3 py-2 text-sm bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 rounded-lg"
                            >
                                <FiRefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
                                Atualizar
                            </button>

                            <div className="text-sm text-gray-600 dark:text-gray-400">
                                <span className="font-medium">{filteredEntries.length}</span> entradas •
                                <span className="font-medium">{getTotalHours().toFixed(1)}h</span> total •
                                <span className="font-medium">{getTotalBillableHours().toFixed(1)}h</span> contabilizável
                                {showDeleted && getDeletedCount() > 0 && (
                                    <span className="text-red-600 dark:text-red-400 ml-2">
                                        • <span className="font-medium">{getDeletedCount()}</span> deletadas
                                    </span>
                                )}
                            </div>
                        </div>

                        <div className="flex items-center space-x-2">
                            {filteredEntries.filter(e => !e.status || e.status !== 'deleted').length > 0 && (
                                <button
                                    onClick={selectAllEntries}
                                    className="text-sm text-primary-600 hover:text-primary-700 dark:text-primary-500"
                                >
                                    {selectedEntries.length === filteredEntries.filter(e => !e.status || e.status !== 'deleted').length ? 'Desmarcar todas' : 'Selecionar todas'}
                                </button>
                            )}

                            {selectedEntries.length > 0 && (
                                <button
                                    onClick={deleteSelectedEntries}
                                    disabled={deleting}
                                    className="flex items-center px-3 py-2 text-sm bg-red-600 hover:bg-red-700 text-white rounded-lg disabled:opacity-50"
                                >
                                    {deleting ? (
                                        <FiLoader className="w-4 h-4 mr-2 animate-spin" />
                                    ) : (
                                        <FiTrash2 className="w-4 h-4 mr-2" />
                                    )}
                                    Deletar ({selectedEntries.length})
                                </button>
                            )}
                        </div>
                    </div>

                    {loading ? (
                        <div className="flex justify-center items-center py-12">
                            <div className="animate-spin w-8 h-8 border-4 border-primary-600 border-t-transparent rounded-full"></div>
                        </div>
                    ) : (
                        <div className="overflow-y-auto max-h-96 border border-gray-200 rounded-lg dark:border-gray-700">
                            {filteredEntries.length === 0 ? (
                                <div className="text-center py-8">
                                    <p className="text-gray-500 dark:text-gray-400">
                                        Nenhuma entrada de tempo encontrada no período selecionado.
                                    </p>
                                </div>
                            ) : (
                                <table className="w-full">
                                    <thead className="bg-gray-50 dark:bg-gray-700">
                                    <tr>
                                        <th className="w-8 px-3 py-2">
                                            <input
                                                type="checkbox"
                                                checked={selectedEntries.length === filteredEntries.filter(e => !e.status || e.status !== 'deleted').length && filteredEntries.filter(e => !e.status || e.status !== 'deleted').length > 0}
                                                onChange={selectAllEntries}
                                                className="w-4 h-4 text-primary-600 rounded"
                                            />
                                        </th>
                                        <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Status</th>
                                        <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Data</th>
                                        <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Projeto</th>
                                        <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Tarefa</th>
                                        <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Descrição</th>
                                        <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Tempo</th>
                                        <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Horas</th>
                                        <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Contabilizável</th>
                                    </tr>
                                    </thead>
                                    <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                                    {filteredEntries.map((entry) => {
                                        const isDeleted = entry.status === 'deleted';
                                        const isSelected = selectedEntries.includes(entry.id);

                                        return (
                                            <tr
                                                key={entry.id}
                                                className={`${
                                                    isDeleted
                                                        ? 'bg-red-50 dark:bg-red-900/10 opacity-75'
                                                        : 'hover:bg-gray-50 dark:hover:bg-gray-700'
                                                } ${
                                                    isSelected && !isDeleted ? 'bg-primary-50 dark:bg-primary-900/20' : ''
                                                }`}
                                            >
                                                <td className="px-3 py-2">
                                                    {!isDeleted && (
                                                        <input
                                                            type="checkbox"
                                                            checked={isSelected}
                                                            onChange={() => toggleEntrySelection(entry.id)}
                                                            className="w-4 h-4 text-primary-600 rounded"
                                                        />
                                                    )}
                                                </td>
                                                <td className="px-3 py-2">
                                                       <span className={`inline-flex px-2 py-1 text-xs rounded-full ${
                                                           isDeleted
                                                               ? 'bg-red-100 text-red-800 dark:bg-red-800 dark:text-red-100'
                                                               : 'bg-green-100 text-green-800 dark:bg-green-800 dark:text-green-100'
                                                       }`}>
                                                           {isDeleted ? 'Deletado' : 'Ativo'}
                                                       </span>
                                                </td>
                                                <td className="px-3 py-2 text-sm text-gray-900 dark:text-white">
                                                    {formatDate(entry.date)}
                                                </td>
                                                <td className="px-3 py-2 text-sm text-gray-900 dark:text-white">
                                                    {entry.projectName || 'N/A'}
                                                </td>
                                                <td className="px-3 py-2 text-sm text-gray-900 dark:text-white">
                                                    {entry.taskName || 'N/A'}
                                                </td>
                                                <td className="px-3 py-2 text-sm text-gray-900 dark:text-white max-w-xs truncate">
                                                    {entry.description || 'Sem descrição'}
                                                </td>
                                                <td className="px-3 py-2 text-sm text-gray-900 dark:text-white">
                                                    {entry.minutes || 0} min
                                                </td>
                                                <td className="px-3 py-2 text-sm text-gray-900 dark:text-white">
                                                    {((entry.minutes || 0) / 60).toFixed(2)}h
                                                </td>
                                                <td className="px-3 py-2">
                                                       <span className={`inline-flex px-2 py-1 text-xs rounded-full ${
                                                           entry.isBillable
                                                               ? 'bg-green-100 text-green-800 dark:bg-green-800 dark:text-green-100'
                                                               : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
                                                       }`}>
                                                           {entry.isBillable ? 'Sim' : 'Não'}
                                                       </span>
                                                </td>
                                            </tr>
                                        );
                                    })}
                                    </tbody>
                                </table>
                            )}
                        </div>
                    )}

                    {showResults && deleteResults.length > 0 && (
                        <div className="mt-6 bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
                            <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">
                                Resultados da Deleção
                            </h3>
                            <div className="space-y-2 max-h-32 overflow-y-auto">
                                {deleteResults.map((result, index) => (
                                    <div key={index} className={`flex items-center text-sm ${
                                        result.success
                                            ? 'text-green-700 dark:text-green-300'
                                            : 'text-red-700 dark:text-red-300'
                                    }`}>
                                        {result.success ? (
                                            <FiCheck className="w-4 h-4 mr-2" />
                                        ) : (
                                            <FiAlertCircle className="w-4 h-4 mr-2" />
                                        )}
                                        <span>
                                           Entrada {result.entryId}: {result.message}
                                       </span>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </div>

                <div className="px-6 py-3 border-t border-gray-200 dark:border-gray-700 flex justify-end">
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

export default TimeEntryManager;