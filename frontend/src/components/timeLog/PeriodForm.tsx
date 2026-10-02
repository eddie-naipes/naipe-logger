import {FiAlertCircle, FiCalendar, FiLoader, FiPlay} from 'react-icons/fi';
import type {NonWorkingDaysMap} from '../../hooks/useNonWorkingDays';
import type {DateRange} from '../../hooks/useTimeEntries';
import type {NonWorkingDay} from '../../types/backend';

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white';

const describeNonWorkingDay = (day: NonWorkingDay): string => {
    if (day.type === 'holiday') return `Feriado: ${day.name}`;
    if (day.type === 'weekend') return 'Fim de semana';
    return 'Dia não útil';
};

interface DateFieldProps {
    id: string;
    label: string;
    value: string;
    onChange: (value: string) => void;
    nonWorkingDay: NonWorkingDay | undefined;
    min?: string | undefined;
}

const DateField = ({id, label, value, onChange, nonWorkingDay, min}: DateFieldProps) => (
    <div>
        <label htmlFor={id} className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {label}
        </label>
        <input
            type="date"
            id={id}
            value={value}
            min={min}
            onChange={(e) => onChange(e.target.value)}
            className={inputClass}
            aria-describedby={nonWorkingDay ? `${id}-aviso` : undefined}
        />
        {nonWorkingDay && (
            <p id={`${id}-aviso`} className="mt-1 text-xs text-yellow-600 dark:text-yellow-400">
                <FiAlertCircle className="inline-block mr-1" aria-hidden="true"/>
                {describeNonWorkingDay(nonWorkingDay)}
            </p>
        )}
    </div>
);

interface PeriodFormProps {
    dateRange: DateRange;
    onChange: (range: DateRange) => void;
    nonWorkingDays: NonWorkingDaysMap;
    onGenerate: () => void;
    isGenerating: boolean;
    disabled: boolean;
}

// Período do plano e botão de gerar.
const PeriodForm = ({dateRange, onChange, nonWorkingDays, onGenerate, isGenerating, disabled}: PeriodFormProps) => (
    <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
            <FiCalendar className="w-5 h-5 mr-2" aria-hidden="true"/>
            Período
        </h2>

        <div className="space-y-4">
            <DateField
                id="startDate"
                label="Data Inicial"
                value={dateRange.startDate}
                onChange={(startDate) => onChange({...dateRange, startDate})}
                nonWorkingDay={nonWorkingDays[dateRange.startDate]}
            />
            <DateField
                id="endDate"
                label="Data Final"
                value={dateRange.endDate}
                min={dateRange.startDate || undefined}
                onChange={(endDate) => onChange({...dateRange, endDate})}
                nonWorkingDay={nonWorkingDays[dateRange.endDate]}
            />

            <button
                type="button"
                onClick={onGenerate}
                disabled={isGenerating || disabled}
                className="btn-primary w-full flex items-center justify-center disabled:opacity-50"
            >
                {isGenerating ? (
                    <>
                        <FiLoader className="w-5 h-5 mr-2 animate-spin" aria-hidden="true"/>
                        Gerando...
                    </>
                ) : (
                    <>
                        <FiPlay className="w-5 h-5 mr-2" aria-hidden="true"/>
                        Gerar Plano
                    </>
                )}
            </button>
        </div>
    </div>
);

export default PeriodForm;
