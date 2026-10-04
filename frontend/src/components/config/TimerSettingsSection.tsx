import {useEffect, useState} from 'react';
import {toast} from 'react-toastify';
import {FiSave, FiWatch} from 'react-icons/fi';
import {GetTimerSettings, SaveTimerSettings} from '@wailsjs/go/backend/App';
import type {config} from '@wailsjs/go/models';
import {type Dados, paraBinding} from '../../types/backend';
import {errMsg} from '../../utils/errors';

type TimerSettings = Dados<config.TimerSettings>;

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white';

// Seção "Cronômetro": arredondamento ao lançar e aviso de cronômetro esquecido.
const TimerSettingsSection = () => {
    const [settings, setSettings] = useState<TimerSettings | null>(null);
    const [salvando, setSalvando] = useState(false);

    useEffect(() => {
        GetTimerSettings()
            .then(setSettings)
            .catch((error: unknown) => toast.error('Erro ao carregar o cronômetro: ' + errMsg(error)));
    }, []);

    if (!settings) return null;

    const salvar = async () => {
        setSalvando(true);
        try {
            await SaveTimerSettings(paraBinding<config.TimerSettings>(settings));
            toast.success('Preferências do cronômetro salvas.');
        } catch (error) {
            toast.error('Erro ao salvar o cronômetro: ' + errMsg(error));
        } finally {
            setSalvando(false);
        }
    };

    return (
        <div className="card max-w-md mx-auto mt-6">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                <FiWatch className="w-5 h-5 mr-2" aria-hidden="true"/>
                Cronômetro
            </h2>
            <div className="space-y-3">
                <div>
                    <label htmlFor="cronometro-arredondamento" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        Arredondamento ao lançar
                    </label>
                    <select id="cronometro-arredondamento" value={settings.rounding}
                            onChange={e => setSettings({...settings, rounding: e.target.value})} className={inputClass}>
                        <option value="exact">Exato, por minuto</option>
                        <option value="15min">Múltiplos de 15 minutos (para cima)</option>
                    </select>
                </div>
                <div>
                    <label htmlFor="cronometro-limite" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        Avisar se rodar por mais de
                    </label>
                    <select id="cronometro-limite" value={settings.longRunningHours}
                            onChange={e => setSettings({...settings, longRunningHours: Number(e.target.value)})} className={inputClass}>
                        <option value={0}>Nunca avisar</option>
                        {[2, 3, 4, 6, 8, 10].map(h => <option key={h} value={h}>{h} horas</option>)}
                    </select>
                </div>
            </div>
            <button type="button" onClick={() => void salvar()} disabled={salvando}
                    className="w-full mt-4 btn-primary flex items-center justify-center disabled:opacity-50">
                <FiSave className="w-4 h-4 mr-2" aria-hidden="true"/>
                {salvando ? 'Salvando...' : 'Salvar cronômetro'}
            </button>
        </div>
    );
};

export default TimerSettingsSection;
