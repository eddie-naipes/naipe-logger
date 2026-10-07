import {useEffect, useState} from 'react';
import {toast} from 'react-toastify';
import {FiPower} from 'react-icons/fi';
import {GetAutostart, SetAutostart} from '@wailsjs/go/backend/App';
import type {autostart} from '@wailsjs/go/models';
import type {Dados} from '../../types/backend';
import {errMsg} from '../../utils/errors';

type AutostartStatus = Dados<autostart.Status>;

// Seção "Iniciar com o sistema": o estado vem do SO (registro do Windows,
// LaunchAgent do macOS, .desktop do Linux), não da configuração do app.
const AutostartSection = () => {
    const [status, setStatus] = useState<AutostartStatus | null>(null);
    const [salvando, setSalvando] = useState(false);

    useEffect(() => {
        GetAutostart()
            .then(setStatus)
            .catch((error: unknown) => toast.error('Erro ao consultar o início com o sistema: ' + errMsg(error)));
    }, []);

    if (!status) return null;

    const alternar = async (enabled: boolean) => {
        setSalvando(true);
        try {
            const novo = await SetAutostart(enabled);
            setStatus(novo);
            toast.success(novo.enabled
                ? 'O app vai abrir minimizado quando você entrar no sistema.'
                : 'O app não vai mais abrir com o sistema.');
        } catch (error) {
            toast.error('Erro ao alterar o início com o sistema: ' + errMsg(error));
        } finally {
            setSalvando(false);
        }
    };

    return (
        <div className="card max-w-md mx-auto mt-6">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                <FiPower className="w-5 h-5 mr-2" aria-hidden="true"/>
                Inicialização
            </h2>
            <label className="flex items-start gap-2 text-sm text-gray-700 dark:text-gray-300">
                <input
                    type="checkbox"
                    className="mt-1"
                    checked={status.enabled}
                    disabled={!status.supported || salvando}
                    onChange={e => void alternar(e.target.checked)}
                />
                <span>Iniciar com o sistema (minimizado)</span>
            </label>
            {status.reason && (
                <p className="mt-2 text-xs text-amber-700 dark:text-amber-300">{status.reason}</p>
            )}
            <p className="mt-3 text-xs text-gray-500 dark:text-gray-400">
                Os lembretes e o cronômetro só funcionam com o app aberto. Com esta opção ele abre sozinho,
                minimizado na barra de tarefas, quando você entra no computador.
            </p>
        </div>
    );
};

export default AutostartSection;
