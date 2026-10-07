import {useState} from 'react';
import {toast} from 'react-toastify';
import {FiArrowDown, FiArrowUp, FiPlus, FiSave, FiSliders, FiTrash2} from 'react-icons/fi';
import {SaveAgendaSettings} from '@wailsjs/go/backend/App';
import type {config} from '@wailsjs/go/models';
import {type Dados, paraBinding, type Task} from '../../types/backend';
import {errMsg} from '../../utils/errors';
import TaskSelect from './TaskSelect';

export type AgendaSettings = Dados<config.AgendaSettings>;
export type AgendaRule = Dados<config.AgendaRule>;

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2 dark:bg-gray-700 dark:border-gray-600 dark:text-white';
const labelClass = 'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';

const SEM_TAREFA = {taskId: 0, taskName: '', projectId: 0, projectName: ''};

interface RulesCardProps {
    settings: AgendaSettings;
    savedTasks: readonly Task[];
    onChange: (settings: AgendaSettings) => void;
    // Chamado depois de salvar com sucesso (para recalcular o plano).
    onSaved: () => void;
}

// Regras de mapeamento (a primeira que casa com o título vale) e preferências
// da importação. O estado fica na página, que também cria regras a partir de
// um evento.
const RulesCard = ({settings, savedTasks, onChange, onSaved}: RulesCardProps) => {
    const [saving, setSaving] = useState(false);
    const [ignoreText, setIgnoreText] = useState(() => settings.ignoreWords.join(', '));

    const setRule = (index: number, patch: Partial<AgendaRule>) =>
        onChange({...settings, rules: settings.rules.map((r, i) => (i === index ? {...r, ...patch} : r))});

    const moveRule = (index: number, delta: number) => {
        const destino = index + delta;
        if (destino < 0 || destino >= settings.rules.length) return;
        const rules = [...settings.rules];
        const [regra] = rules.splice(index, 1);
        if (regra) rules.splice(destino, 0, regra);
        onChange({...settings, rules});
    };

    const save = async () => {
        setSaving(true);
        const ignoreWords = ignoreText.split(',').map(w => w.trim()).filter(Boolean);
        const next = {...settings, ignoreWords};
        try {
            await SaveAgendaSettings(paraBinding<config.AgendaSettings>(next));
            onChange(next);
            toast.success('Regras da agenda salvas.');
            onSaved();
        } catch (err) {
            toast.error('Erro ao salvar as regras: ' + errMsg(err));
        } finally {
            setSaving(false);
        }
    };

    return (
        <div className="card">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                <FiSliders className="w-5 h-5 mr-2" aria-hidden="true"/>
                Regras de mapeamento
            </h2>
            <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
                A primeira regra cuja palavra-chave aparece no título do evento define a tarefa. Sem descrição, o
                lançamento usa o título do evento.
            </p>

            <div className="space-y-3 mb-4">
                {settings.rules.length === 0 && (
                    <p className="text-sm text-gray-500 dark:text-gray-400">Nenhuma regra ainda.</p>
                )}
                {settings.rules.map((rule, i) => (
                    <fieldset key={i} className="border border-gray-200 dark:border-gray-700 rounded-lg p-3 space-y-2">
                        <legend className="px-1 text-xs font-medium text-gray-600 dark:text-gray-400">Regra {i + 1}</legend>
                        <div className="flex gap-2 items-center">
                            <input
                                aria-label={`Palavra-chave da regra ${i + 1}`}
                                value={rule.match}
                                onChange={e => setRule(i, {match: e.target.value})}
                                placeholder={rule.isRegex ? 'Expressão regular' : 'Palavra-chave no título'}
                                className={inputClass}
                            />
                            <label className="flex items-center gap-1 text-xs text-gray-700 dark:text-gray-300 shrink-0">
                                <input type="checkbox" checked={rule.isRegex}
                                       onChange={e => setRule(i, {isRegex: e.target.checked})}/>
                                Regex
                            </label>
                        </div>
                        <TaskSelect
                            id={`agenda-regra-${i}-tarefa`}
                            ariaLabel={`Tarefa da regra ${i + 1}`}
                            value={rule.task}
                            savedTasks={savedTasks}
                            onChange={task => setRule(i, {task: task ?? SEM_TAREFA})}
                            emptyLabel="Escolha a tarefa"
                            className={inputClass}
                        />
                        <input
                            aria-label={`Descrição da regra ${i + 1}`}
                            value={rule.description}
                            onChange={e => setRule(i, {description: e.target.value})}
                            placeholder="Descrição (opcional; padrão: título do evento)"
                            className={inputClass}
                        />
                        <div className="flex justify-end gap-1">
                            <button type="button" onClick={() => moveRule(i, -1)} disabled={i === 0}
                                    aria-label={`Subir a regra ${i + 1}`}
                                    className="p-1.5 rounded hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-30">
                                <FiArrowUp className="w-4 h-4" aria-hidden="true"/>
                            </button>
                            <button type="button" onClick={() => moveRule(i, 1)} disabled={i === settings.rules.length - 1}
                                    aria-label={`Descer a regra ${i + 1}`}
                                    className="p-1.5 rounded hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-30">
                                <FiArrowDown className="w-4 h-4" aria-hidden="true"/>
                            </button>
                            <button type="button"
                                    onClick={() => onChange({...settings, rules: settings.rules.filter((_, j) => j !== i)})}
                                    aria-label={`Remover a regra ${i + 1}`}
                                    className="p-1.5 rounded text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20">
                                <FiTrash2 className="w-4 h-4" aria-hidden="true"/>
                            </button>
                        </div>
                    </fieldset>
                ))}
                <button
                    type="button"
                    onClick={() => onChange({
                        ...settings,
                        rules: [...settings.rules, {match: '', isRegex: false, task: SEM_TAREFA, description: ''}]
                    })}
                    className="btn-secondary w-full flex items-center justify-center"
                >
                    <FiPlus className="w-4 h-4 mr-2" aria-hidden="true"/>
                    Nova regra
                </button>
            </div>

            <div className="space-y-3 border-t border-gray-200 dark:border-gray-700 pt-4">
                <div>
                    <label htmlFor="agenda-tarefa-padrao" className={labelClass}>Eventos sem regra</label>
                    <TaskSelect
                        id="agenda-tarefa-padrao"
                        value={settings.defaultTask}
                        savedTasks={savedTasks}
                        onChange={task => onChange({...settings, defaultTask: task ?? SEM_TAREFA})}
                        emptyLabel="Deixar para escolher na lista"
                        className={inputClass}
                    />
                </div>
                <div>
                    <label htmlFor="agenda-ignorar" className={labelClass}>Ignorar títulos com (separe por vírgula)</label>
                    <input id="agenda-ignorar" value={ignoreText} onChange={e => setIgnoreText(e.target.value)}
                           placeholder="almoço, pessoal" className={inputClass}/>
                </div>
                <div className="grid grid-cols-2 gap-3">
                    <div>
                        <label htmlFor="agenda-minimo" className={labelClass}>Duração mínima (min)</label>
                        <input id="agenda-minimo" type="number" min={0} max={1440} value={settings.minMinutes}
                               onChange={e => onChange({...settings, minMinutes: Math.max(0, Number(e.target.value) || 0)})}
                               className={inputClass}/>
                    </div>
                    <div>
                        <label htmlFor="agenda-arredondamento" className={labelClass}>Arredondamento</label>
                        <select id="agenda-arredondamento" value={settings.rounding}
                                onChange={e => onChange({...settings, rounding: e.target.value})} className={inputClass}>
                            <option value="exact">Exato</option>
                            <option value="15">15 minutos</option>
                        </select>
                    </div>
                </div>
                <div>
                    <label htmlFor="agenda-email" className={labelClass}>Seu e-mail na agenda</label>
                    <input id="agenda-email" type="email" value={settings.userEmail}
                           onChange={e => onChange({...settings, userEmail: e.target.value})}
                           placeholder="Para ignorar convites que você recusou" className={inputClass}/>
                </div>
                <label className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                    <input type="checkbox" checked={settings.billable}
                           onChange={e => onChange({...settings, billable: e.target.checked})}/>
                    Lançar como faturável
                </label>
                <label className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                    <input type="checkbox" checked={settings.includeTransparent}
                           onChange={e => onChange({...settings, includeTransparent: e.target.checked})}/>
                    Incluir eventos marcados como “livre”
                </label>
            </div>

            <button type="button" onClick={() => void save()} disabled={saving}
                    className="btn-primary w-full mt-4 flex items-center justify-center disabled:opacity-50">
                <FiSave className="w-4 h-4 mr-2" aria-hidden="true"/>
                {saving ? 'Salvando...' : 'Salvar regras'}
            </button>
        </div>
    );
};

export default RulesCard;
