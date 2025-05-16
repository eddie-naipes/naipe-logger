import React, { useState, useEffect } from 'react';
import { toast } from 'react-toastify';
import { FiSave, FiLoader, FiEye, FiEyeOff, FiUser, FiInfo, FiLogOut, FiBriefcase } from 'react-icons/fi';
import whaleTeamLogo from '../assets/whaleTeam.png';
import { GetConfig } from '../../wailsjs/go/backend/App';

const Config = ({ onConfigSaved }) => {
    const [isLoading, setIsLoading] = useState(true);
    const [isConfigured, setIsConfigured] = useState(false);
    const [isLoggingIn, setIsLoggingIn] = useState(false);
    const [isLoggingOut, setIsLoggingOut] = useState(false);
    const [credentials, setCredentials] = useState({
        email: '',
        password: '',
        companyName: 'onebrain'
    });
    const [showPassword, setShowPassword] = useState(false);
    const [debugMode, setDebugMode] = useState(false);
    const [debugInfo, setDebugInfo] = useState(null);
    const [configuredHost, setConfiguredHost] = useState("");

    useEffect(() => {
        const checkExistingConfig = async () => {
            try {
                const savedConfig = await GetConfig();
                if (savedConfig && savedConfig.authToken && savedConfig.apiHost) {
                    setIsConfigured(true);
                    setConfiguredHost(savedConfig.apiHost);
                }
            } catch (error) {
                console.error('Erro ao verificar configuração existente:', error);
            } finally {
                setIsLoading(false);
            }
        };

        checkExistingConfig();
    }, []);

    const extractCompanyName = (host) => {
        let cleanHost = host;
        if (cleanHost.startsWith('http://')) {
            cleanHost = cleanHost.substring(7);
        } else if (cleanHost.startsWith('https://')) {
            cleanHost = cleanHost.substring(8);
        }

        if (cleanHost.startsWith('teamwork.')) {
            const parts = cleanHost.split('.');
            if (parts.length >= 3) {
                return parts[1];
            }
        }

        return cleanHost;
    };

    const formatCompanyNameToHost = (companyName) => {
        if (!companyName) return '';
        const formattedName = companyName.trim().toLowerCase();
        return `teamwork.${formattedName}.com.br`;
    };

    const handleCredentialsChange = (e) => {
        const { name, value } = e.target;
        setCredentials({ ...credentials, [name]: value });
    };

    const handleLogin = async (e) => {
        e.preventDefault();

        if (!credentials.email || !credentials.password) {
            toast.warning("Preencha todos os campos de login.");
            return;
        }

        setIsLoggingIn(true);
        setDebugInfo(null);

        const formattedHost = formatCompanyNameToHost(credentials.companyName);

        try {
            console.log("Enviando requisição de login para:", formattedHost);

            const result = await window.go.backend.App.LoginWithCredentials(
                credentials.email,
                credentials.password,
                formattedHost
            );

            if (debugMode) {
                setDebugInfo(JSON.stringify(result, null, 2));
            }

            if (result && result.success) {
                toast.success("Login realizado com sucesso! Configuração concluída.");

                setCredentials(prev => ({ ...prev, password: '' }));

                setIsConfigured(true);
                setConfiguredHost(formattedHost);

                if (onConfigSaved) {
                    onConfigSaved();
                }
            } else {
                const errorMessage = result && result.message
                    ? result.message
                    : "Falha na autenticação. Verifique suas credenciais.";

                toast.error(errorMessage);
            }
        } catch (error) {
            console.error("Erro ao fazer login:", error);
            toast.error("Erro ao tentar login: " + (error.message || error));
            if (debugMode) {
                setDebugInfo(JSON.stringify(error, null, 2));
            }
        } finally {
            setIsLoggingIn(false);
        }
    };

    const handleLogout = async () => {
        if (!window.confirm("Tem certeza que deseja sair? Esta ação removerá sua configuração atual.")) {
            return;
        }

        setIsLoggingOut(true);
        try {
            const emptyConfig = {
                authToken: "",
                userID: 0,
                apiHost: "",
                minutosPorDia: 480,
            };

            await window.go.backend.App.SaveConfig(emptyConfig);

            setIsConfigured(false);
            setConfiguredHost("");
            setCredentials({
                email: '',
                password: '',
                companyName: 'onebrain'
            });

            toast.success("Logout realizado com sucesso. Configuração removida.");

            if (onConfigSaved) {
                onConfigSaved();
            }
        } catch (error) {
            console.error("Erro ao fazer logout:", error);
            toast.error("Erro ao remover configuração: " + (error.message || error));
        } finally {
            setIsLoggingOut(false);
        }
    };

    const toggleDebugMode = () => {
        setDebugMode(!debugMode);
        if (!debugMode) {
            setDebugInfo("Modo de depuração ativado. Os detalhes da resposta de login serão exibidos aqui.");
        } else {
            setDebugInfo(null);
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
                        ? "Sua conta está configurada e pronta para uso."
                        : "Configure sua conexão para começar a usar o aplicativo"}
                </p>
            </div>

            <div className="card max-w-md mx-auto">
                <div className="flex justify-center mb-6">
                    <img
                        src={whaleTeamLogo}
                        alt="Whale Team Logo"
                        className="h-32 object-contain"
                    />
                </div>

                <div className="flex justify-between items-center mb-4">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                        {isConfigured ? "Status da Conta" : "Login no Teamwork"}
                    </h2>
                    <div className="flex items-center">
                        <button
                            type="button"
                            onClick={toggleDebugMode}
                            className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
                            title="Informações de Depuração"
                        >
                            <FiInfo className="w-5 h-5" />
                        </button>
                    </div>
                </div>

                {debugInfo && (
                    <div className="bg-gray-100 dark:bg-gray-800 p-3 rounded mb-4 overflow-auto max-h-60 text-xs font-mono">
                        <pre>{debugInfo}</pre>
                    </div>
                )}

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
                                        Você está conectado ao servidor <strong>{configuredHost}</strong> e pode usar todos os recursos do aplicativo.
                                    </p>
                                    <p className="mt-2 text-sm text-green-700 dark:text-green-300">
                                        Para alterar sua configuração, faça logout e reconfigure sua conta.
                                    </p>
                                </div>
                            </div>
                        </div>

                        <div className="flex justify-center">
                            <button
                                type="button"
                                onClick={handleLogout}
                                disabled={isLoggingOut}
                                className="btn flex items-center justify-center px-5 py-2.5 bg-red-50 text-red-700 hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/40 border border-red-200 dark:border-red-800 rounded-lg"
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
                    <form onSubmit={handleLogin} className="space-y-4">
                        <div>
                            <label htmlFor="email" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Email <span className="text-red-500">*</span>
                            </label>
                            <div className="relative">
                                <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                                    <FiUser className="h-5 w-5 text-gray-400" />
                                </div>
                                <input
                                    type="email"
                                    id="email"
                                    name="email"
                                    value={credentials.email}
                                    onChange={handleCredentialsChange}
                                    className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full pl-10 p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white"
                                    placeholder="seu.email@empresa.com"
                                    required
                                />
                            </div>
                        </div>

                        <div>
                            <label htmlFor="password" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Senha <span className="text-red-500">*</span>
                            </label>
                            <div className="relative">
                                <input
                                    type={showPassword ? "text" : "password"}
                                    id="password"
                                    name="password"
                                    value={credentials.password}
                                    onChange={handleCredentialsChange}
                                    className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 pr-10 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white"
                                    placeholder="Sua senha do Teamwork"
                                    required
                                />
                                <button
                                    type="button"
                                    onClick={() => setShowPassword(!showPassword)}
                                    className="absolute inset-y-0 right-0 px-3 flex items-center"
                                >
                                    {showPassword ? (
                                        <FiEyeOff className="w-5 h-5 text-gray-500 dark:text-gray-400" />
                                    ) : (
                                        <FiEye className="w-5 h-5 text-gray-500 dark:text-gray-400" />
                                    )}
                                </button>
                            </div>
                        </div>

                        <div className="bg-blue-50 dark:bg-blue-900/20 p-3 rounded-lg border border-blue-200 dark:border-blue-800">
                            <p className="text-sm text-blue-700 dark:text-blue-300">
                                Conectando ao servidor: <strong>teamwork.onebrain.com.br</strong>
                            </p>
                        </div>

                        <button
                            type="submit"
                            disabled={isLoggingIn}
                            className="w-full btn-primary flex items-center justify-center"
                        >
                            {isLoggingIn ? (
                                <>
                                    <FiLoader className="w-5 h-5 mr-2 animate-spin" />
                                    Conectando...
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
        </div>
    );
};

export default Config;