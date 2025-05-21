import React, { useState } from 'react';
import { FiCalendar, FiDownload, FiX, FiLoader } from 'react-icons/fi';
import { toast } from 'react-toastify';

const ReportPeriodModal = ({ isOpen, onClose, onExport }) => {
    const [startDate, setStartDate] = useState('');
    const [endDate, setEndDate] = useState('');
    const [isExporting, setIsExporting] = useState(false);

    const handleSubmit = async (e) => {
        e.preventDefault();
        if (!startDate || !endDate) {
            toast.warning("Por favor, selecione as datas inicial e final");
            return;
        }

        setIsExporting(true);
        try {
            await onExport(startDate, endDate);
            onClose();
        } catch (error) {
        } finally {
            setIsExporting(false);
        }
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md p-6">
                <div className="flex justify-between items-center mb-4">
                    <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                        Exportar Relatório de Período
                    </h3>
                    <button
                        onClick={onClose}
                        className="text-gray-400 hover:text-gray-500 dark:hover:text-gray-300"
                    >
                        <FiX className="w-5 h-5" />
                    </button>
                </div>

                <form onSubmit={handleSubmit}>
                    <div className="space-y-4">
                        <div>
                            <label htmlFor="startDate" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                Data Inicial
                            </label>
                            <div className="mt-1 relative rounded-md shadow-sm">
                                <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                                    <FiCalendar className="text-gray-400" />
                                </div>
                                <input
                                    type="date"
                                    id="startDate"
                                    value={startDate}
                                    onChange={(e) => setStartDate(e.target.value)}
                                    className="pl-10 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                                    required
                                />
                            </div>
                        </div>

                        <div>
                            <label htmlFor="endDate" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                Data Final
                            </label>
                            <div className="mt-1 relative rounded-md shadow-sm">
                                <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                                    <FiCalendar className="text-gray-400" />
                                </div>
                                <input
                                    type="date"
                                    id="endDate"
                                    value={endDate}
                                    onChange={(e) => setEndDate(e.target.value)}
                                    className="pl-10 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                                    required
                                />
                            </div>
                        </div>
                    </div>

                    <div className="mt-6 flex justify-end space-x-3">
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 dark:bg-gray-700 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-600"
                        >
                            Cancelar
                        </button>
                        <button
                            type="submit"
                            disabled={isExporting}
                            className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500 disabled:opacity-70 disabled:cursor-not-allowed dark:bg-primary-700 dark:hover:bg-primary-800"
                        >
                            {isExporting ? (
                                <>
                                    <FiLoader className="w-4 h-4 mr-2 inline animate-spin" />
                                    Exportando...
                                </>
                            ) : (
                                <>
                                    <FiDownload className="w-4 h-4 mr-2 inline" />
                                    Exportar
                                </>
                            )}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default ReportPeriodModal;