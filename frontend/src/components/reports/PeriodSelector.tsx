import {presetRange, REPORT_PRESETS, type ReportPreset, type ReportRange} from '../../utils/reportPeriods';

interface PeriodSelectorProps {
    preset: ReportPreset;
    range: ReportRange;
    onChange: (preset: ReportPreset, range: ReportRange) => void;
    error?: string | null;
}

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block p-2 dark:bg-gray-700 dark:border-gray-600 dark:text-white';

// Seletor de período: atalhos em uma linha e, em "Personalizado", as datas.
const PeriodSelector = ({preset, range, onChange, error}: PeriodSelectorProps) => (
    <div className="flex flex-wrap items-end gap-3">
        <div role="group" aria-label="Período" className="inline-flex flex-wrap rounded-lg border border-gray-300 dark:border-gray-600 overflow-hidden">
            {REPORT_PRESETS.map(p => (
                <button
                    key={p.value}
                    type="button"
                    aria-pressed={preset === p.value}
                    onClick={() => onChange(p.value, presetRange(p.value) ?? range)}
                    className={`px-3 py-2 text-sm border-r last:border-r-0 border-gray-300 dark:border-gray-600 ${preset === p.value
                        ? 'bg-primary-600 text-white dark:bg-primary-700'
                        : 'bg-white text-gray-700 hover:bg-gray-100 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700'}`}
                >
                    {p.label}
                </button>
            ))}
        </div>

        {preset === 'custom' && (
            <>
                <div>
                    <label htmlFor="relatorioInicio" className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Início</label>
                    <input id="relatorioInicio" type="date" className={inputClass} value={range.startDate}
                           onChange={(e) => onChange('custom', {...range, startDate: e.target.value})}/>
                </div>
                <div>
                    <label htmlFor="relatorioFim" className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Fim</label>
                    <input id="relatorioFim" type="date" className={inputClass} value={range.endDate}
                           min={range.startDate || undefined}
                           aria-invalid={Boolean(error)}
                           onChange={(e) => onChange('custom', {...range, endDate: e.target.value})}/>
                </div>
            </>
        )}
    </div>
);

export default PeriodSelector;
