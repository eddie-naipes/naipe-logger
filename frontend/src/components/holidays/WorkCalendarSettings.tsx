import {type FormEvent, useEffect, useState} from 'react';
import {toast} from 'react-toastify';
import {FiInfo, FiLoader, FiMapPin, FiPlus, FiSave, FiSun, FiTrash2} from 'react-icons/fi';
import {GetBrazilianStates, GetWorkCalendarSettings, SaveWorkCalendarSettings} from '@wailsjs/go/backend/App';
import {paraBinding, type Absence, type BrazilianState, type CalendarSettings, type CustomHoliday} from '../../types/backend';
import {formatDateBR} from '../../utils/dates';
import {errMsg} from '../../utils/errors';
import {CUSTOM_HOLIDAY_TYPES, customHolidayTypeLabel, validateAbsence, validateCustomHoliday} from './calendarValidation';

// Seção do HolidayManager que configura o que, além de fins de semana e
// feriados nacionais, não é dia útil: UF (feriados estaduais), feriados
// municipais/pontes e férias. Fica junto do gerenciador de feriados (e não na
// tela de Configurações) para que tudo sobre "dias sem expediente" esteja num
// só lugar; o backend aplica a configuração em planos, lembretes e calendário.

interface WorkCalendarSettingsProps {
    // onSaved avisa quem exibe dias não úteis para recarregar.
    onSaved?: (settings: CalendarSettings) => void;
}

const VAZIO: CalendarSettings = {uf: '', disabledStateHolidays: [], customHolidays: [], absences: []};
const NOVO_FERIADO: CustomHoliday = {date: '', name: '', type: 'municipal', recurring: false};
const NOVA_AUSENCIA: Absence = {start: '', end: '', description: ''};

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2 dark:bg-gray-700 dark:border-gray-600 dark:text-white';
const labelClass = 'block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1';
const sectionClass = 'bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4';

const WorkCalendarSettings = ({onSaved}: WorkCalendarSettingsProps) => {
    const [settings, setSettings] = useState<CalendarSettings>(VAZIO);
    const [states, setStates] = useState<BrazilianState[]>([]);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [dirty, setDirty] = useState(false);
    const [novoFeriado, setNovoFeriado] = useState<CustomHoliday>(NOVO_FERIADO);
    const [erroFeriado, setErroFeriado] = useState<string | null>(null);
    const [novaAusencia, setNovaAusencia] = useState<Absence>(NOVA_AUSENCIA);
    const [erroAusencia, setErroAusencia] = useState<string | null>(null);

    useEffect(() => {
        let cancelled = false;
        Promise.all([GetWorkCalendarSettings(), GetBrazilianStates()])
            .then(([cfg, ufs]) => {
                if (cancelled) return;
                setSettings({
                    uf: cfg.uf,
                    disabledStateHolidays: cfg.disabledStateHolidays ?? [],
                    customHolidays: cfg.customHolidays ?? [],
                    absences: cfg.absences ?? []
                });
                setStates(ufs ?? []);
            })
            .catch((error: unknown) => {
                console.error('Erro ao carregar calendário de trabalho:', error);
                toast.error('Erro ao carregar calendário de trabalho: ' + errMsg(error));
            })
            .finally(() => {
                if (!cancelled) setLoading(false);
            });
        return () => {
            cancelled = true;
        };
    }, []);

    const update = (patch: Partial<CalendarSettings>) => {
        setSettings(s => ({...s, ...patch}));
        setDirty(true);
    };

    const estado = states.find(s => s.uf === settings.uf);
    const desligados = new Set(settings.disabledStateHolidays);

    const toggleEstadual = (monthDay: string) => {
        const lista = desligados.has(monthDay)
            ? settings.disabledStateHolidays.filter(md => md !== monthDay)
            : [...settings.disabledStateHolidays, monthDay];
        update({disabledStateHolidays: lista});
    };

    const adicionarFeriado = (e: FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        const feriado = {...novoFeriado, name: novoFeriado.name.trim()};
        const erro = validateCustomHoliday(feriado, settings.customHolidays);
        setErroFeriado(erro);
        if (erro) return;
        update({customHolidays: [...settings.customHolidays, feriado].sort((a, b) => a.date.localeCompare(b.date))});
        setNovoFeriado(NOVO_FERIADO);
    };

    const adicionarAusencia = (e: FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        const ausencia = {...novaAusencia, description: novaAusencia.description.trim()};
        const erro = validateAbsence(ausencia);
        setErroAusencia(erro);
        if (erro) return;
        update({absences: [...settings.absences, ausencia].sort((a, b) => a.start.localeCompare(b.start))});
        setNovaAusencia(NOVA_AUSENCIA);
    };

    const salvar = async () => {
        setSaving(true);
        try {
            const salvo = await SaveWorkCalendarSettings(paraBinding(settings));
            const normalizado: CalendarSettings = {
                uf: salvo.uf,
                disabledStateHolidays: salvo.disabledStateHolidays ?? [],
                customHolidays: salvo.customHolidays ?? [],
                absences: salvo.absences ?? []
            };
            setSettings(normalizado);
            setDirty(false);
            toast.success('Calendário de trabalho salvo. Planos e calendário já consideram os novos dias.');
            onSaved?.(normalizado);
        } catch (error) {
            console.error('Erro ao salvar calendário de trabalho:', error);
            toast.error('Não foi possível salvar o calendário: ' + errMsg(error));
        } finally {
            setSaving(false);
        }
    };

    if (loading) {
        return (
            <div className="flex justify-center py-6" role="status" aria-label="Carregando calendário de trabalho">
                <FiLoader className="w-6 h-6 animate-spin text-primary-600" aria-hidden="true"/>
            </div>
        );
    }

    return (
        <section aria-labelledby="calendario-trabalho-titulo" className="mb-6 space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
                <h3 id="calendario-trabalho-titulo" className="text-lg font-medium text-gray-900 dark:text-white flex items-center">
                    <FiMapPin className="w-5 h-5 mr-2" aria-hidden="true"/>
                    Calendário de trabalho
                </h3>
                <button
                    type="button"
                    onClick={() => void salvar()}
                    disabled={saving || !dirty}
                    className="flex items-center px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white text-sm rounded-lg disabled:opacity-50"
                >
                    {saving
                        ? <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/>
                        : <FiSave className="w-4 h-4 mr-2" aria-hidden="true"/>}
                    {dirty ? 'Salvar calendário' : 'Calendário salvo'}
                </button>
            </div>
            <p className="text-sm text-gray-600 dark:text-gray-400">
                Além dos feriados nacionais, estes dias deixam de contar como úteis na distribuição de horas,
                no calendário mensal e nos relatórios.
            </p>

            {/* Feriados estaduais */}
            <div className={sectionClass}>
                <label htmlFor="calendarUF" className={labelClass}>Estado (feriados estaduais)</label>
                <select
                    id="calendarUF"
                    value={settings.uf}
                    onChange={(e) => update({uf: e.target.value, disabledStateHolidays: []})}
                    className={`${inputClass} max-w-xs`}
                >
                    <option value="">Nenhum (só feriados nacionais)</option>
                    {states.map(s => (
                        <option key={s.uf} value={s.uf}>{s.uf} — {s.name}</option>
                    ))}
                </select>

                {estado && (
                    <div className="mt-3">
                        {estado.holidays.length === 0 ? (
                            <p className="text-sm text-gray-600 dark:text-gray-400">
                                Não há feriado estadual de data fixa cadastrado para {estado.name}.
                            </p>
                        ) : (
                            <ul className="space-y-1">
                                {estado.holidays.map(h => (
                                    <li key={h.monthDay}>
                                        <label className="inline-flex items-start text-sm text-gray-800 dark:text-gray-200">
                                            <input
                                                type="checkbox"
                                                className="mt-0.5 mr-2 rounded-sm border-gray-300 text-primary-600 focus:ring-primary-500"
                                                checked={!desligados.has(h.monthDay)}
                                                onChange={() => toggleEstadual(h.monthDay)}
                                            />
                                            <span>
                                                <span className="font-medium">{h.monthDay.slice(3)}/{h.monthDay.slice(0, 2)}</span> — {h.name}
                                                <span className="block text-xs text-gray-500 dark:text-gray-400">{h.source}</span>
                                            </span>
                                        </label>
                                    </li>
                                ))}
                            </ul>
                        )}
                        <p className="mt-2 flex items-start text-xs text-gray-500 dark:text-gray-400">
                            <FiInfo className="w-3.5 h-3.5 mr-1 mt-0.5 shrink-0" aria-hidden="true"/>
                            A lista embutida é conservadora (só feriados estaduais de data fixa amplamente conhecidos).
                            Desmarque o que sua empresa não folga e cadastre abaixo o que faltar.
                        </p>
                    </div>
                )}
            </div>

            {/* Feriados personalizados */}
            <div className={sectionClass}>
                <h4 className="text-sm font-semibold text-gray-900 dark:text-white mb-2">Feriados municipais, pontes e folgas</h4>
                {settings.customHolidays.length === 0 ? (
                    <p className="text-sm text-gray-500 dark:text-gray-400 mb-3">Nenhum feriado personalizado.</p>
                ) : (
                    <ul className="divide-y divide-gray-200 dark:divide-gray-600 mb-3" aria-label="Feriados personalizados">
                        {settings.customHolidays.map((h, i) => (
                            <li key={`${h.date}-${h.name}-${i}`} className="flex items-center justify-between py-1.5 text-sm">
                                <span className="text-gray-800 dark:text-gray-200">
                                    <span className="font-medium">
                                        {h.recurring ? formatDateBR(h.date, 'dd/MM') : formatDateBR(h.date)}
                                    </span>
                                    {' — '}{h.name}
                                    <span className="ml-2 text-xs text-gray-500 dark:text-gray-400">
                                        {customHolidayTypeLabel(h.type)}{h.recurring ? ' · todo ano' : ''}
                                    </span>
                                </span>
                                <button
                                    type="button"
                                    onClick={() => update({customHolidays: settings.customHolidays.filter((_, j) => j !== i)})}
                                    aria-label={`Remover ${h.name}`}
                                    className="p-1 text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/30 rounded-sm"
                                >
                                    <FiTrash2 className="w-4 h-4" aria-hidden="true"/>
                                </button>
                            </li>
                        ))}
                    </ul>
                )}

                <form onSubmit={adicionarFeriado} noValidate className="grid grid-cols-1 md:grid-cols-12 gap-2 items-end">
                    <div className="md:col-span-3">
                        <label htmlFor="feriadoData" className={labelClass}>Data</label>
                        <input id="feriadoData" type="date" className={inputClass} value={novoFeriado.date}
                               onChange={(e) => setNovoFeriado(f => ({...f, date: e.target.value}))}/>
                    </div>
                    <div className="md:col-span-4">
                        <label htmlFor="feriadoNome" className={labelClass}>Nome</label>
                        <input id="feriadoNome" type="text" maxLength={120} className={inputClass} value={novoFeriado.name}
                               placeholder="Ex.: Aniversário da cidade"
                               onChange={(e) => setNovoFeriado(f => ({...f, name: e.target.value}))}/>
                    </div>
                    <div className="md:col-span-2">
                        <label htmlFor="feriadoTipo" className={labelClass}>Tipo</label>
                        <select id="feriadoTipo" className={inputClass} value={novoFeriado.type}
                                onChange={(e) => setNovoFeriado(f => ({...f, type: e.target.value}))}>
                            {CUSTOM_HOLIDAY_TYPES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
                        </select>
                    </div>
                    <div className="md:col-span-2 flex items-center pb-2">
                        <input id="feriadoRecorrente" type="checkbox" checked={novoFeriado.recurring}
                               className="mr-2 rounded-sm border-gray-300 text-primary-600 focus:ring-primary-500"
                               onChange={(e) => setNovoFeriado(f => ({...f, recurring: e.target.checked}))}/>
                        <label htmlFor="feriadoRecorrente" className="text-xs text-gray-700 dark:text-gray-300">Repete todo ano</label>
                    </div>
                    <div className="md:col-span-1">
                        <button type="submit" aria-label="Adicionar feriado"
                                className="w-full flex justify-center p-2 bg-gray-600 hover:bg-gray-700 text-white rounded-lg">
                            <FiPlus className="w-5 h-5" aria-hidden="true"/>
                        </button>
                    </div>
                    {erroFeriado && (
                        <p role="alert" className="md:col-span-12 text-xs text-red-600 dark:text-red-400">{erroFeriado}</p>
                    )}
                </form>
            </div>

            {/* Férias e ausências */}
            <div className={sectionClass}>
                <h4 className="text-sm font-semibold text-gray-900 dark:text-white mb-2 flex items-center">
                    <FiSun className="w-4 h-4 mr-1" aria-hidden="true"/>
                    Férias e ausências
                </h4>
                {settings.absences.length === 0 ? (
                    <p className="text-sm text-gray-500 dark:text-gray-400 mb-3">Nenhum período cadastrado.</p>
                ) : (
                    <ul className="divide-y divide-gray-200 dark:divide-gray-600 mb-3" aria-label="Férias e ausências">
                        {settings.absences.map((a, i) => (
                            <li key={`${a.start}-${a.end}-${i}`} className="flex items-center justify-between py-1.5 text-sm">
                                <span className="text-gray-800 dark:text-gray-200">
                                    <span className="font-medium">{formatDateBR(a.start)} a {formatDateBR(a.end)}</span>
                                    {a.description && <> — {a.description}</>}
                                </span>
                                <button
                                    type="button"
                                    onClick={() => update({absences: settings.absences.filter((_, j) => j !== i)})}
                                    aria-label={`Remover ausência de ${formatDateBR(a.start)} a ${formatDateBR(a.end)}`}
                                    className="p-1 text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/30 rounded-sm"
                                >
                                    <FiTrash2 className="w-4 h-4" aria-hidden="true"/>
                                </button>
                            </li>
                        ))}
                    </ul>
                )}

                <form onSubmit={adicionarAusencia} noValidate className="grid grid-cols-1 md:grid-cols-12 gap-2 items-end">
                    <div className="md:col-span-3">
                        <label htmlFor="ausenciaInicio" className={labelClass}>Início</label>
                        <input id="ausenciaInicio" type="date" className={inputClass} value={novaAusencia.start}
                               onChange={(e) => setNovaAusencia(a => ({...a, start: e.target.value}))}/>
                    </div>
                    <div className="md:col-span-3">
                        <label htmlFor="ausenciaFim" className={labelClass}>Fim</label>
                        <input id="ausenciaFim" type="date" className={inputClass} value={novaAusencia.end}
                               min={novaAusencia.start || undefined}
                               onChange={(e) => setNovaAusencia(a => ({...a, end: e.target.value}))}/>
                    </div>
                    <div className="md:col-span-5">
                        <label htmlFor="ausenciaDescricao" className={labelClass}>Descrição (opcional)</label>
                        <input id="ausenciaDescricao" type="text" maxLength={120} className={inputClass}
                               value={novaAusencia.description} placeholder="Ex.: Férias"
                               onChange={(e) => setNovaAusencia(a => ({...a, description: e.target.value}))}/>
                    </div>
                    <div className="md:col-span-1">
                        <button type="submit" aria-label="Adicionar ausência"
                                className="w-full flex justify-center p-2 bg-gray-600 hover:bg-gray-700 text-white rounded-lg">
                            <FiPlus className="w-5 h-5" aria-hidden="true"/>
                        </button>
                    </div>
                    {erroAusencia && (
                        <p role="alert" className="md:col-span-12 text-xs text-red-600 dark:text-red-400">{erroAusencia}</p>
                    )}
                </form>
            </div>
        </section>
    );
};

export default WorkCalendarSettings;
