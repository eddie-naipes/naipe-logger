import {type FormEvent, useEffect, useState} from 'react';
import {FiAlertCircle, FiCalendar, FiDownload, FiLoader} from 'react-icons/fi';
import {toast} from 'react-toastify';
import Modal from '../components/Modal';
import {errMsg} from '../utils/errors';

// onExport(startDate, endDate) deve lançar em caso de falha: o modal só fecha
// quando a exportação termina com sucesso, e mostra o erro sem perder as datas.
interface ReportPeriodModalProps {
    isOpen: boolean;
    onClose: () => void;
    onExport: (startDate: string, endDate: string) => Promise<void>;
}

const ReportPeriodModal = ({isOpen, onClose, onExport}: ReportPeriodModalProps) => {
    const [startDate, setStartDate] = useState('');
    const [endDate, setEndDate] = useState('');
    const [isExporting, setIsExporting] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (isOpen) setError(null);
    }, [isOpen]);

    // Datas 'YYYY-MM-DD' comparam corretamente como texto.
    const invalidRange = Boolean(startDate && endDate && startDate > endDate);

    const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        setError(null);

        if (!startDate || !endDate) {
            toast.warning('Por favor, selecione as datas inicial e final');
            return;
        }

        if (invalidRange) {
            toast.warning('A data inicial deve ser anterior ou igual à data final.');
            return;
        }

        setIsExporting(true);
        try {
            await onExport(startDate, endDate);
            onClose();
        } catch (err) {
            const msg = errMsg(err);
            console.error('Erro ao exportar relatório do período:', err);
            setError(msg);
            toast.error('Não foi possível exportar o relatório: ' + msg);
        } finally {
            setIsExporting(false);
        }
    };

    const inputClass = 'pl-10 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white';

    return (
        <Modal
            isOpen={isOpen}
            onClose={onClose}
            size="sm"
            title="Exportar Relatório de Período"
            closeDisabled={isExporting}
        >
            <form onSubmit={(e) => void handleSubmit(e)} noValidate>
                <div className="space-y-4">
                    <div>
                        <label htmlFor="reportStartDate"
                               className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                            Data Inicial
                        </label>
                        <div className="mt-1 relative rounded-md shadow-sm">
                            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                                <FiCalendar className="text-gray-400" aria-hidden="true"/>
                            </div>
                            <input
                                type="date"
                                id="reportStartDate"
                                value={startDate}
                                onChange={(e) => setStartDate(e.target.value)}
                                className={inputClass}
                                required
                            />
                        </div>
                    </div>

                    <div>
                        <label htmlFor="reportEndDate"
                               className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                            Data Final
                        </label>
                        <div className="mt-1 relative rounded-md shadow-sm">
                            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                                <FiCalendar className="text-gray-400" aria-hidden="true"/>
                            </div>
                            <input
                                type="date"
                                id="reportEndDate"
                                value={endDate}
                                min={startDate || undefined}
                                onChange={(e) => setEndDate(e.target.value)}
                                className={inputClass}
                                aria-invalid={invalidRange}
                                aria-describedby={invalidRange ? 'reportRangeError' : undefined}
                                required
                            />
                        </div>
                        {invalidRange && (
                            <p id="reportRangeError" className="mt-1 text-xs text-red-600 dark:text-red-400">
                                A data final não pode ser anterior à data inicial.
                            </p>
                        )}
                    </div>

                    {error && (
                        <div role="alert"
                             className="flex items-start p-3 bg-red-50 dark:bg-red-900/20 border-l-4 border-red-500 rounded">
                            <FiAlertCircle className="w-4 h-4 mt-0.5 mr-2 text-red-500 flex-shrink-0"
                                           aria-hidden="true"/>
                            <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
                        </div>
                    )}
                </div>

                <div className="mt-6 flex justify-end space-x-3">
                    <button
                        type="button"
                        onClick={onClose}
                        disabled={isExporting}
                        className="px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 dark:bg-gray-700 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-600 disabled:opacity-50"
                    >
                        Cancelar
                    </button>
                    <button
                        type="submit"
                        disabled={isExporting || invalidRange}
                        className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500 disabled:opacity-70 disabled:cursor-not-allowed dark:bg-primary-700 dark:hover:bg-primary-800"
                    >
                        {isExporting ? (
                            <>
                                <FiLoader className="w-4 h-4 mr-2 inline animate-spin" aria-hidden="true"/>
                                Exportando...
                            </>
                        ) : (
                            <>
                                <FiDownload className="w-4 h-4 mr-2 inline" aria-hidden="true"/>
                                Exportar
                            </>
                        )}
                    </button>
                </div>
            </form>
        </Modal>
    );
};

export default ReportPeriodModal;
