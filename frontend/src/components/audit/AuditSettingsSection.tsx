import {useEffect, useId, useState} from 'react';
import {toast} from 'react-toastify';
import {FiLoader, FiSave} from 'react-icons/fi';
import {GetAuditSettings, SaveAuditSettings} from '@wailsjs/go/backend/App';
import {paraBinding, type AuditSettings} from '../../types/backend';
import {errMsg} from '../../utils/errors';

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white';
const labelClass = 'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';

interface AuditSettingsSectionProps {
    // Chamado depois de salvar, para reexecutar a auditoria.
    onSaved: () => void;
}

// Configurações da auditoria: ficam na própria página do fechamento porque só
// valem para ela, e assim o efeito aparece na hora (a auditoria roda de novo
// ao salvar).
const AuditSettingsSection = ({onSaved}: AuditSettingsSectionProps) => {
    const limiteId = useId();
    const genericasId = useId();
    const [settings, setSettings] = useState<AuditSettings | null>(null);
    const [limitHours, setLimitHours] = useState('10');
    const [generic, setGeneric] = useState('');
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        let cancelled = false;
        GetAuditSettings()
            .then(s => {
                if (cancelled) return;
                setSettings(s);
                setLimitHours(String(s.dailyLimitMinutes / 60));
                setGeneric((s.genericDescriptions ?? []).join('\n'));
            })
            .catch((err: unknown) => {
                if (!cancelled) toast.error('Erro ao carregar as configurações da auditoria: ' + errMsg(err));
            });
        return () => {
            cancelled = true;
        };
    }, []);

    const save = async () => {
        if (!settings) return;
        const horas = Number(limitHours.replace(',', '.'));
        if (!Number.isFinite(horas) || horas < 0 || horas > 24) {
            toast.warning('Informe um limite diário entre 0 e 24 horas (0 desliga).');
            return;
        }
        setSaving(true);
        try {
            const novo: AuditSettings = {
                ...settings,
                dailyLimitMinutes: Math.round(horas * 60),
                genericDescriptions: generic.split('\n').map(l => l.trim()).filter(Boolean)
            };
            await SaveAuditSettings(paraBinding(novo));
            setSettings(novo);
            toast.success('Configurações da auditoria salvas.');
            onSaved();
        } catch (err) {
            toast.error('Erro ao salvar as configurações da auditoria: ' + errMsg(err));
        } finally {
            setSaving(false);
        }
    };

    if (!settings) {
        return (
            <div className="flex justify-center py-4" role="status" aria-label="Carregando configurações">
                <FiLoader className="w-5 h-5 animate-spin text-primary-600" aria-hidden="true"/>
            </div>
        );
    }

    return (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
                <label htmlFor={limiteId} className={labelClass}>Limite diário (horas)</label>
                <input id={limiteId} type="number" min={0} max={24} step={0.5} className={inputClass}
                       value={limitHours} onChange={e => setLimitHours(e.target.value)}/>
                <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    Dias acima disso viram aviso. 0 desliga a verificação.
                </p>
            </div>
            <div>
                <label htmlFor={genericasId} className={labelClass}>Descrições genéricas (uma por linha)</label>
                <textarea id={genericasId} rows={4} className={inputClass} value={generic}
                          placeholder={'reunião\najustes'}
                          onChange={e => setGeneric(e.target.value)}/>
                <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    Comparação sem diferenciar maiúsculas. Descrições iguais ao nome da tarefa já são apontadas.
                </p>
            </div>
            <div className="md:col-span-2 flex justify-end">
                <button type="button" onClick={() => void save()} disabled={saving}
                        className="btn-primary inline-flex items-center disabled:opacity-50">
                    {saving ? <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/> : <FiSave className="w-4 h-4 mr-2" aria-hidden="true"/>}
                    Salvar configurações
                </button>
            </div>
        </div>
    );
};

export default AuditSettingsSection;
