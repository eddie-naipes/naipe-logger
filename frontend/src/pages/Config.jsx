import React, { useState, useEffect } from 'react';
import { toast } from 'react-toastify';
import { FiSave, FiLoader, FiEye, FiEyeOff, FiUser, FiLogOut, FiAlertTriangle, FiExternalLink, FiClock } from 'react-icons/fi';
import whaleTeamLogo from '../assets/whaleTeam.png';
import {
    ConnectWithToken,
    GetPublicConfig,
    LegacyCredentialPurged,
    Logout,
    SetMinutosPorDia
} from '../../wailsjs/go/backend/App';
import {errMsg} from '../utils/errors';
import TimeInputComponent from '../components/TimeInputComponent';
import {setMinutosPorDiaCache} from '../hooks/useMinutosPorDia';
import {
    formatHoursMinutes,
    hoursAndMinutesToMinutes,
    MINUTOS_POR_DIA_PADRAO,
    minutesToHoursAndMinutes
} from '../utils/time';

const DEFAULT_HOST = 'teamwork.onebrain.com.br';

const Config = ({ onConfigSaved }) => {
    const [isLoading, setIsLoading] = useState(true);
    const [isConfigured, setIsConfigured] = useState(false);
    const [isConnecting, setIsConnecting] = useState(false);
    const [isLoggingOut, setIsLoggingOut] = useState(false);
    const [legacyPurged, setLegacyPurged] = useState(false);
    const [form, setForm] = useState({ host: DEFAULT_HOST, token: '' });
    const [showToken, setShowToken] = useState(false);
    const [configuredHost, setConfiguredHost] = useState('');
    const [minutosPorDia, setMinutosPorDia] = useState(MINUTOS_POR_DIA_PADRAO);
    const [jornada, setJornada] = useState(minutesToHoursAndMinutes(MINUTOS_POR_DIA_PADRAO));
    const [isSavingJornada, setIsSavingJornada] = useState(false);

    useEffect(() => {
        const checkExistingConfig = async () => {
            try {
                // GetPublicConfig nunca devolve o token: o segredo permanece no
                // cofre de credenciais do sistema, fora do alcance do webview.
                const publicConfig = await GetPublicConfig();
                setIsConfigured(publicConfig.configured);
                setConfiguredHost(publicConfig.apiHost || '');
                if (publicConfig.apiHost) {
                    setForm(prev => ({ ...prev, host: publicConfig.apiHost }));
                }
                if (publicConfig.minutosPorDia > 0) {
                    setMinutosPorDia(publicConfig.minutosPorDia);
                    setJornada(minutesToHoursAndMinutes(publicConfig.minutosPorDia));
                }

                setLegacyPurged(await LegacyCredentialPurged());
            } catch (error) {
                console.error('Erro ao verificar configuração existente:', error);
            } finally {
                setIsLoading(false);
            }
        };

        checkExistingConfig();
    }, []);

    const handleChange = (e) => {
        const { name, value } = e.target;
        setForm(prev => ({ ...prev, [name]: value }));
    };

    const handleConnect = async (e) => {
        e.preventDefault();

        if (!form.host.trim() || !form.token.trim()) {
            toast.warning('Informe o domínio da empresa e o token de API.');
            return;
        }

        setIsConnecting(true);

        try {
            const result = await ConnectWithToken(
                form.token.trim(),
                form.host.trim()
            );

            if (result && result.success) {
                // Não guardamos o token no estado do React após o envio.
                setForm(prev => ({ ...prev, token: '' }));
                setIsConfigured(true);
                setConfiguredHost(result.instanceId || form.host.trim());
                setLegacyPurged(false);
                toast.success('Token validado. Configuração concluída.');

                if (onConfigSaved) {
                    onConfigSaved();
                }
            } else {
                toast.error(result?.message || 'Falha ao validar o token de API.');
            }
        } catch (error) {
            console.error('Erro ao conectar:', error);
            toast.error('Erro ao conectar: ' + errMsg(error));
        } finally {
            setIsConnecting(false);
        }
    };

    const jornadaMinutos = hoursAndMinutesToMinutes(jornada.hours, jornada.minutes);

    const handleSaveJornada = async (e) => {
        e.preventDefault();

        if (jornadaMinutos <= 0) {
            toast.warning('Informe uma jornada diária maior que zero.');
            return;
        }

        setIsSavingJornada(true);
        try {
            await SetMinutosPorDia(jornadaMinutos);
            setMinutosPorDia(jornadaMinutos);
            setMinutosPorDiaCache(jornadaMinutos);
            toast.success(`Jornada diária definida em ${formatHoursMinutes(jornadaMinutos)}.`);
        } catch (error) {
            console.error('Erro ao salvar jornada diária:', error);
            toast.error('Erro ao salvar jornada diária: ' + errMsg(error));
        } finally {
            setIsSavingJornada(false);
        }
    };

    const handleLogout = async () => {
        if (!window.confirm('Tem certeza que deseja sair? O token será removido do cofre de credenciais do sistema.')) {
            return;
        }

        setIsLoggingOut(true);
        try {
            await Logout();

            setIsConfigured(false);
            setConfiguredHost('');
            setForm({ host: DEFAULT_HOST, token: '' });

            toast.success('Logout realizado. Token removido do cofre.');

            if (onConfigSaved) {
                onConfigSaved();
            }
        } catch (error) {
            console.error('Erro ao fazer logout:', error);
            toast.error('Erro ao remover configuração: ' + errMsg(error));
        } finally {
            setIsLoggingOut(false);
        }
    };

    if (isLoading) {
        return (
            <div className="flex items-center justify-center h-full">
                <div className="animate-spin-slow w-12 h-12 border-4 border-primary-600 border-t-transparent rounded-full"></div>
            </div>
        );
    }

    return (
        <div>
            <div className="mb-6">
                <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">Configuração do Teamwork</h1>
                <p className="text-gray-600 dark:text-gray-400">
                    {isConfigured
                        ? 'Sua conta está configurada e pronta para uso.'
                        : 'Conecte-se com um token de API para começar a usar o aplicativo'}
                </p>
            </div>

            {legacyPurged && (
                <div className="card max-w-md mx-auto mb-4 border-l-4 border-amber-500">
                    <div className="flex items-start">
                        <FiAlertTriangle className="mt-0.5 w-5 h-5 text-amber-500 mr-3 flex-shrink-0" />
                        <div>
                            <h3 className="text-md font-medium text-amber-800 dark:text-amber-400">
                                Credencial antiga removida
                            </h3>
                            <p className="mt-2 text-sm text-amber-700 dark:text-amber-300">
                                Versões anteriores guardavam seu <strong>email e senha</strong> em disco com uma
                                proteção que podia ser revertida. Essa credencial foi apagada.
                            </p>
                            <p className="mt-2 text-sm text-amber-700 dark:text-amber-300">
                                Recomendamos <strong>trocar sua senha do Teamwork</strong> e usar um token de API
                                abaixo.
                            </p>
                        </div>
                    </div>
                </div>
            )}

            <div className="card max-w-md mx-auto">
                <div className="flex justify-center mb-6">
                    <img
                        src={whaleTeamLogo}
                        alt="Whale Team Logo"
                        className="h-32 object-contain"
                    />
                </div>

                <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                    {isConfigured ? 'Status da Conta' : 'Conectar ao Teamwork'}
                </h2>

                {isConfigured ? (
                    <div>
                        <div className="bg-green-50 dark:bg-green-900/20 p-5 rounded-lg border-l-4 border-green-500 dark:border-green-700 mb-6">
                            <div className="flex items-start">
                                <FiUser className="mt-0.5 w-5 h-5 text-green-500 dark:text-green-400 mr-3" />
                                <div>
                                    <h3 className="text-md font-medium text-green-800 dark:text-green-400">
                                        Configuração Ativa
                                    </h3>
                                    <p className="mt-2 text-sm text-green-700 dark:text-green-300">
                                        Conectado a <strong>{configuredHost}</strong>. O token está guardado no
                                        cofre de credenciais do sistema operacional.
                                    </p>
                                </div>
                            </div>
                        </div>

                        <div className="flex justify-center">
                            <button
                                type="button"
                                onClick={handleLogout}
                                disabled={isLoggingOut}
                                className="flex items-center justify-center px-5 py-2.5 font-medium text-sm disabled:opacity-50 bg-red-50 text-red-700 hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/40 border border-red-200 dark:border-red-800 rounded-lg"
                            >
                                {isLoggingOut ? (
                                    <>
                                        <FiLoader className="w-5 h-5 mr-2 animate-spin" />
                                        Saindo...
                                    </>
                                ) : (
                                    <>
                                        <FiLogOut className="w-5 h-5 mr-2" />
                                        Fazer Logout
                                    </>
                                )}
                            </button>
                        </div>
                    </div>
                ) : (
                    <form onSubmit={handleConnect} className="space-y-4">
                        <div>
                            <label htmlFor="host" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Domínio da empresa <span className="text-red-500">*</span>
                            </label>
                            <input
                                type="text"
                                id="host"
                                name="host"
                                value={form.host}
                                onChange={handleChange}
                                className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white"
                                placeholder="suaempresa.teamwork.com"
                                autoComplete="off"
                                required
                            />
                            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                                Somente https. O endereço é usado exatamente como informado.
                            </p>
                        </div>

                        <div>
                            <label htmlFor="token" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Token de API <span className="text-red-500">*</span>
                            </label>
                            <div className="relative">
                                <input
                                    type={showToken ? 'text' : 'password'}
                                    id="token"
                                    name="token"
                                    value={form.token}
                                    onChange={handleChange}
                                    className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 pr-10 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white"
                                    placeholder="Cole aqui seu token de API"
                                    autoComplete="off"
                                    spellCheck="false"
                                    required
                                />
                                <button
                                    type="button"
                                    onClick={() => setShowToken(!showToken)}
                                    className="absolute inset-y-0 right-0 px-3 flex items-center"
                                    aria-label={showToken ? 'Ocultar token' : 'Mostrar token'}
                                >
                                    {showToken ? (
                                        <FiEyeOff className="w-5 h-5 text-gray-500 dark:text-gray-400" />
                                    ) : (
                                        <FiEye className="w-5 h-5 text-gray-500 dark:text-gray-400" />
                                    )}
                                </button>
                            </div>
                        </div>

                        <div className="bg-blue-50 dark:bg-blue-900/20 p-3 rounded-lg border border-blue-200 dark:border-blue-800">
                            <p className="text-sm text-blue-700 dark:text-blue-300 flex items-start">
                                <FiExternalLink className="w-4 h-4 mr-2 mt-0.5 flex-shrink-0" />
                                <span>
                                    Gere um token no Teamwork em <strong>Perfil &rarr; Edit My Details &rarr; API &amp;
                                    Mobile</strong>. Sua senha não é usada nem armazenada por este aplicativo.
                                </span>
                            </p>
                        </div>

                        <button
                            type="submit"
                            disabled={isConnecting}
                            className="w-full btn-primary flex items-center justify-center"
                        >
                            {isConnecting ? (
                                <>
                                    <FiLoader className="w-5 h-5 mr-2 animate-spin" />
                                    Validando token...
                                </>
                            ) : (
                                <>
                                    <FiSave className="w-5 h-5 mr-2" />
                                    Conectar ao Teamwork
                                </>
                            )}
                        </button>
                    </form>
                )}
            </div>

            {isConfigured && (
                <div className="card max-w-md mx-auto mt-6">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-1 flex items-center">
                        <FiClock className="w-5 h-5 mr-2" aria-hidden="true" />
                        Jornada Diária
                    </h2>
                    <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                        Usada como meta do dia no calendário e como base da meta mensal do dashboard.
                        Atual: <strong>{formatHoursMinutes(minutosPorDia)}</strong>.
                    </p>
                    <form onSubmit={handleSaveJornada} className="space-y-4">
                        <TimeInputComponent
                            hours={jornada.hours}
                            minutes={jornada.minutes}
                            onTimeChange={(hours, minutes) => setJornada({ hours, minutes })}
                            label="Horas por dia (horas:minutos)"
                            disabled={isSavingJornada}
                            showTotalMinutes={true}
                        />
                        <button
                            type="submit"
                            disabled={isSavingJornada || jornadaMinutos <= 0 || jornadaMinutos === minutosPorDia}
                            className="w-full btn-primary flex items-center justify-center disabled:opacity-50"
                        >
                            {isSavingJornada ? (
                                <>
                                    <FiLoader className="w-5 h-5 mr-2 animate-spin" aria-hidden="true" />
                                    Salvando...
                                </>
                            ) : (
                                <>
                                    <FiSave className="w-5 h-5 mr-2" aria-hidden="true" />
                                    Salvar Jornada
                                </>
                            )}
                        </button>
                    </form>
                </div>
            )}
        </div>
    );
};

export default Config;
