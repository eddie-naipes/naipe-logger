import {useEffect, useState} from 'react';
import {toast} from 'react-toastify';
import {FiAlertTriangle, FiBell, FiSave, FiSend} from 'react-icons/fi';
import {GetNotificationStatus, GetReminderSettings, SaveReminderSettings, SendTestReminder} from '@wailsjs/go/backend/App';
import type {backend, config} from '@wailsjs/go/models';
import {type Dados, paraBinding} from '../../types/backend';
import {errMsg} from '../../utils/errors';

type ReminderSettings = Dados<config.ReminderSettings>;
type NotificationStatus = Dados<backend.NotificationStatus>;

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white';

// Seção "Lembretes": lembrete diário de horas pendentes e de fim de mês, via
// notificações do sistema. "Testar lembrete" envia uma notificação real.
const RemindersSection = () => {
    const [settings, setSettings] = useState<ReminderSettings | null>(null);
    const [status, setStatus] = useState<NotificationStatus | null>(null);
    const [salvando, setSalvando] = useState(false);
    const [testando, setTestando] = useState(false);

    useEffect(() => {
        GetReminderSettings()
            .then(setSettings)
            .catch((error: unknown) => toast.error('Erro ao carregar lembretes: ' + errMsg(error)));
        GetNotificationStatus()
            .then(setStatus)
            .catch((error: unknown) => console.error('Erro ao consultar notificações:', error));
    }, []);

    if (!settings) return null;

    const alterar = (patch: Partial<ReminderSettings>) => setSettings({...settings, ...patch});

    const salvar = async () => {
        setSalvando(true);
        try {
            await SaveReminderSettings(paraBinding<config.ReminderSettings>(settings));
            toast.success('Lembretes salvos.');
        } catch (error) {
            toast.error('Erro ao salvar lembretes: ' + errMsg(error));
        } finally {
            setSalvando(false);
        }
    };

    const testar = async () => {
        setTestando(true);
        try {
            await SendTestReminder();
            toast.info('Notificação de teste enviada.');
        } catch (error) {
            toast.error('Não foi possível enviar a notificação: ' + errMsg(error));
        } finally {
            setTestando(false);
        }
    };

    return (
        <div className="card max-w-md mx-auto mt-6">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-1 flex items-center">
                <FiBell className="w-5 h-5 mr-2" aria-hidden="true"/>
                Lembretes
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                Avisa por notificação do sistema quando o dia ainda não fechou a jornada e, no fim do mês, quais dias
                úteis ficaram incompletos.
            </p>

            {status && !status.available && (
                <div className="flex items-start text-sm text-amber-800 bg-amber-50 dark:text-amber-200 dark:bg-amber-900/30 p-3 rounded-sm mb-4" role="alert">
                    <FiAlertTriangle className="w-4 h-4 mr-2 mt-0.5 shrink-0" aria-hidden="true"/>
                    <span>
                        As notificações do sistema não estão disponíveis{status.error ? ` (${status.error})` : ''}.
                        Verifique as permissões de notificação do sistema e reinicie o app.
                    </span>
                </div>
            )}

            <label className="flex items-center text-sm text-gray-700 dark:text-gray-300 mb-3">
                <input type="checkbox" className="mr-2" checked={settings.enabled}
                       onChange={e => alterar({enabled: e.target.checked})}/>
                Ativar lembretes
            </label>

            <fieldset disabled={!settings.enabled} className="space-y-3 disabled:opacity-60">
                <div>
                    <label htmlFor="lembrete-horario" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        Horário do lembrete diário
                    </label>
                    <input id="lembrete-horario" type="time" value={settings.dailyTime}
                           onChange={e => alterar({dailyTime: e.target.value})} className={inputClass}/>
                </div>
                <label className="flex items-center text-sm text-gray-700 dark:text-gray-300">
                    <input type="checkbox" className="mr-2" checked={settings.workDaysOnly}
                           onChange={e => alterar({workDaysOnly: e.target.checked})}/>
                    Só em dias úteis (sem fins de semana e feriados)
                </label>
                <label className="flex items-center text-sm text-gray-700 dark:text-gray-300">
                    <input type="checkbox" className="mr-2" checked={settings.monthEndEnabled}
                           onChange={e => alterar({monthEndEnabled: e.target.checked})}/>
                    Lembrete de fim de mês
                </label>
                {settings.monthEndEnabled && (
                    <div>
                        <label htmlFor="lembrete-fim-mes" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Avisar nos últimos dias úteis do mês
                        </label>
                        <select id="lembrete-fim-mes" value={settings.monthEndDays}
                                onChange={e => alterar({monthEndDays: Number(e.target.value)})} className={inputClass}>
                            {[1, 2, 3, 4, 5].map(n => (
                                <option key={n} value={n}>{n === 1 ? 'Último dia útil' : `${n} últimos dias úteis`}</option>
                            ))}
                        </select>
                    </div>
                )}
            </fieldset>

            <div className="flex gap-3 mt-4">
                <button type="button" onClick={() => void salvar()} disabled={salvando || !settings.dailyTime}
                        className="flex-1 btn-primary flex items-center justify-center disabled:opacity-50">
                    <FiSave className="w-4 h-4 mr-2" aria-hidden="true"/>
                    {salvando ? 'Salvando...' : 'Salvar lembretes'}
                </button>
                <button type="button" onClick={() => void testar()} disabled={testando}
                        className="btn-secondary flex items-center justify-center disabled:opacity-50">
                    <FiSend className="w-4 h-4 mr-2" aria-hidden="true"/>
                    Testar lembrete
                </button>
            </div>
        </div>
    );
};

export default RemindersSection;
