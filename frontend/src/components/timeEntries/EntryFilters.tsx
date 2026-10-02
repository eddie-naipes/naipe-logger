import {FiEye, FiEyeOff} from 'react-icons/fi';
import type {DateRange} from '../../hooks/useTimeEntries';

// Filtros locais aplicados sobre a lista já carregada (valores dos campos do
// formulário, por isso tudo texto).
export interface EntryFiltersState {
    projectName: string;
    taskName: string;
    minHours: string;
    maxHours: string;
    isBillable: 'all' | 'true' | 'false';
    status: 'all' | 'active' | 'deleted';
}

interface EntryFiltersProps {
    dateRange: DateRange;
    onDateRangeChange: (range: DateRange) => void;
    showDeleted: boolean;
    onToggleDeleted: () => void;
    filters: EntryFiltersState;
    onFiltersChange: (filters: EntryFiltersState) => void;
}

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white';
const labelClass = 'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';

// Filtros do Gerenciador de Apontamentos: período e "incluir deletados" (vão ao
// backend) e filtros locais sobre a lista já carregada.
const EntryFilters = ({dateRange, onDateRangeChange, showDeleted, onToggleDeleted, filters, onFiltersChange}: EntryFiltersProps) => {
    const setFilter = <K extends keyof EntryFiltersState>(field: K, value: EntryFiltersState[K]) =>
        onFiltersChange({...filters, [field]: value});

    return (
        <>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
                <div>
                    <label htmlFor="tem-start-date" className={labelClass}>Data Inicial</label>
                    <input
                        id="tem-start-date"
                        type="date"
                        value={dateRange.startDate}
                        onChange={(e) => onDateRangeChange({...dateRange, startDate: e.target.value})}
                        className={inputClass}
                    />
                </div>
                <div>
                    <label htmlFor="tem-end-date" className={labelClass}>Data Final</label>
                    <input
                        id="tem-end-date"
                        type="date"
                        value={dateRange.endDate}
                        min={dateRange.startDate || undefined}
                        onChange={(e) => onDateRangeChange({...dateRange, endDate: e.target.value})}
                        className={inputClass}
                    />
                </div>
                <div>
                    <span id="tem-deleted-label" className={labelClass}>Incluir Deletados</span>
                    <button
                        type="button"
                        onClick={onToggleDeleted}
                        aria-pressed={showDeleted}
                        aria-labelledby="tem-deleted-label"
                        className={`flex items-center justify-center w-full p-2.5 rounded-lg border text-sm ${
                            showDeleted
                                ? 'bg-red-50 border-red-300 text-red-700 dark:bg-red-900/20 dark:border-red-700 dark:text-red-300'
                                : 'bg-gray-50 border-gray-300 text-gray-900 dark:bg-gray-700 dark:border-gray-600 dark:text-white'
                        }`}
                    >
                        {showDeleted
                            ? <FiEye className="w-4 h-4 mr-2" aria-hidden="true"/>
                            : <FiEyeOff className="w-4 h-4 mr-2" aria-hidden="true"/>}
                        {showDeleted ? 'Mostrando Deletados' : 'Deletados Ocultos'}
                    </button>
                </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-6 gap-4 mb-6">
                <div>
                    <label htmlFor="tem-filter-project" className="sr-only">Filtrar por projeto</label>
                    <input
                        id="tem-filter-project"
                        type="text"
                        placeholder="Filtrar por projeto..."
                        value={filters.projectName}
                        onChange={(e) => setFilter('projectName', e.target.value)}
                        className={inputClass}
                    />
                </div>
                <div>
                    <label htmlFor="tem-filter-task" className="sr-only">Filtrar por tarefa</label>
                    <input
                        id="tem-filter-task"
                        type="text"
                        placeholder="Filtrar por tarefa..."
                        value={filters.taskName}
                        onChange={(e) => setFilter('taskName', e.target.value)}
                        className={inputClass}
                    />
                </div>
                <div>
                    <label htmlFor="tem-filter-min" className="sr-only">Mínimo de horas</label>
                    <input
                        id="tem-filter-min"
                        type="number"
                        placeholder="Min horas"
                        step="0.5"
                        min="0"
                        value={filters.minHours}
                        onChange={(e) => setFilter('minHours', e.target.value)}
                        className={inputClass}
                    />
                </div>
                <div>
                    <label htmlFor="tem-filter-max" className="sr-only">Máximo de horas</label>
                    <input
                        id="tem-filter-max"
                        type="number"
                        placeholder="Max horas"
                        step="0.5"
                        min="0"
                        value={filters.maxHours}
                        onChange={(e) => setFilter('maxHours', e.target.value)}
                        className={inputClass}
                    />
                </div>
                <div>
                    <label htmlFor="tem-filter-billable" className="sr-only">Contabilizável</label>
                    <select
                        id="tem-filter-billable"
                        value={filters.isBillable}
                        onChange={(e) => setFilter('isBillable', e.target.value as EntryFiltersState['isBillable'])}
                        className={inputClass}
                    >
                        <option value="all">Todos</option>
                        <option value="true">Contabilizável</option>
                        <option value="false">Não Contabilizável</option>
                    </select>
                </div>
                <div>
                    <label htmlFor="tem-filter-status" className="sr-only">Situação</label>
                    <select
                        id="tem-filter-status"
                        value={filters.status}
                        onChange={(e) => setFilter('status', e.target.value as EntryFiltersState['status'])}
                        disabled={!showDeleted}
                        title={showDeleted ? undefined : 'Ative "Incluir Deletados" para ver entradas deletadas'}
                        className={`${inputClass} disabled:opacity-60`}
                    >
                        <option value="all">Ativas e deletadas</option>
                        <option value="active">Só ativas</option>
                        <option value="deleted">Só deletadas</option>
                    </select>
                </div>
            </div>
        </>
    );
};

export default EntryFilters;
