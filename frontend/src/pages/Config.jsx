import React, { useState, useEffect } from 'react';
import { toast } from 'react-toastify';
import { FiSave, FiCheck, FiLoader, FiEye, FiEyeOff, FiUser, FiInfo } from 'react-icons/fi';

// Remova LoginWithCredentials da importação direta
import { GetConfig, SaveConfig, TestConnection } from '../../wailsjs/go/backend/App';

const Config = ({ onConfigSaved }) => {
    const [config, setConfig] = useState({
        authToken: '',
        userID: 0,
        apiHost: '',
        minutosPorDia: 480,
    });

    const [isLoading, setIsLoading] = useState(true);
    const [isTesting, setIsTesting] = useState(false);
    const [isSaving, setIsSaving] = useState(false);
    const [showToken, setShowToken] = useState(false);
    const [isLoggingIn, setIsLoggingIn] = useState(false);
    const [showLoginForm, setShowLoginForm] = useState(false);
    const [credentials, setCredentials] = useState({
        email: '',
        password: '',
        host: ''
    });
    const [showPassword, setShowPassword] = useState(false);
    const [loginResponse, setLoginResponse] = useState(null);
    const [debugInfo, setDebugInfo] = useState(null); // Para debugging

    // Carregar configurações ao inicializar
    useEffect(() => {
        const loadConfig = async () => {
            try {
                const savedConfig = await GetConfig();
                if (savedConfig) {
                    setConfig({
                        authToken: savedConfig.authToken || '',
                        userID: savedConfig.userID || 0,
                        apiHost: savedConfig.apiHost || '',
                        minutosPorDia: savedConfig.minutosPorDia || 480,
                    });

                    if (savedConfig.apiHost) {
                        setCredentials(prev => ({
                            ...prev,
                            host: savedConfig.apiHost
                        }));
                    }
                }
            } catch (error) {
                console.error('Erro ao carregar configurações:', error);
                toast.error('Erro ao carregar configurações.');
            } finally {
                setIsLoading(false);
            }
        };

        loadConfig();
    }, []);

    // Monitorar mudanças no objeto config para debug
    useEffect(() => {
        console.log("Estado config atualizado:", config);
    }, [config]);

    // Monitorar mudanças no objeto loginResponse para debug
    useEffect(() => {
        if (loginResponse) {
            console.log("Login Response atualizado:", loginResponse);
            setDebugInfo(JSON.stringify(loginResponse, null, 2));
        }
    }, [loginResponse]);

    const handleChange = (e) => {
        const { name, value } = e.target;

        if (name === 'userID' || name === 'minutosPorDia') {
            setConfig({ ...config, [name]: parseInt(value) || 0 });
        } else if (name === 'apiHost') {
            let cleanedValue = value;
            if (value.startsWith('http://')) {
                cleanedValue = value.substring(7);
            } else if (value.startsWith('https://')) {
                cleanedValue = value.substring(8);
            }
            setConfig({ ...config, [name]: cleanedValue });

            setCredentials(prev => ({ ...prev, host: cleanedValue }));
        } else {
            setConfig({ ...config, [name]: value });
        }
    };

    const handleCredentialsChange = (e) => {
        const { name, value } = e.target;

        if (name === 'host') {
            let cleanedValue = value;
            if (value.startsWith('http://')) {
                cleanedValue = value.substring(7);
            } else if (value.startsWith('https://')) {
                cleanedValue = value.substring(8);
            }
            setCredentials({ ...credentials, [name]: cleanedValue });

            setConfig(prev => ({ ...prev, apiHost: cleanedValue }));
        } else {
            setCredentials({ ...credentials, [name]: value });
        }
    };

    const handleSave = async (e) => {
        e.preventDefault();

        if (!config.authToken || !config.apiHost || config.userID <= 0) {
            toast.warning('Preencha todos os campos obrigatórios.');
            return;
        }

        setIsSaving(true);

        try {
            await SaveConfig(config);
            toast.success('Configurações salvas com sucesso!');

            if (onConfigSaved) {
                onConfigSaved();
            }
        } catch (error) {
            console.error('Erro ao salvar configurações:', error);
            toast.error('Erro ao salvar configurações.');
        } finally {
            setIsSaving(false);
        }
    };

    const testConnection = async () => {
        if (!config.authToken || !config.apiHost) {
            toast.warning('Preencha o Token de Autenticação e o Host da API.');
            return;
        }

        setIsTesting(true);

        try {
            const result = await TestConnection(config);

            if (Array.isArray(result) && result.length >= 2) {
                const [success, message] = result;

                if (success) {
                    toast.success(message || "Conexão estabelecida com sucesso!");
                } else {
                    toast.error(message || "Falha ao conectar com o Teamwork. Verifique suas credenciais.");
                }
            } else {
                if (result === true) {
                    toast.success("Conexão estabelecida com sucesso!");
                } else {
                    toast.error("Falha ao conectar com o Teamwork. Verifique suas credenciais.");
                }
            }
        } catch (error) {
            console.error('Erro ao testar conexão:', error);
            toast.error('Erro ao testar conexão com o Teamwork.');
        } finally {
            setIsTesting(false);
        }
    };

    const fetchUserId = async () => {
        if (!config.authToken || !config.apiHost) {
            toast.warning("Preencha o Token de Autenticação e o Host da API primeiro.");
            return;
        }

        setIsTesting(true);

        try {
            const tempConfig = {
                authToken: config.authToken,
                apiHost: config.apiHost,
                userID: 0,
                minutosPorDia: config.minutosPorDia || 480
            };

            const userId = await window.go.backend.App.GetCurrentUserIdWithConfig(tempConfig);

            if (userId) {
                const updatedConfig = {...config, userID: userId};
                setConfig(updatedConfig);
                await SaveConfig(updatedConfig);
                toast.success(`ID do usuário (${userId}) obtido e configuração salva com sucesso!`);

                if (onConfigSaved) {
                    onConfigSaved();
                }
            }
        } catch (error) {
            console.error("Erro ao obter ID do usuário:", error);
            toast.error("Erro ao obter ID do usuário. Verifique suas credenciais.");
        } finally {
            setIsTesting(false);
        }
    };

    const handleLogin = async (e) => {
        e.preventDefault();

        if (!credentials.email || !credentials.password || !credentials.host) {
            toast.warning("Preencha todos os campos de login.");
            return;
        }

        setIsLoggingIn(true);
        setLoginResponse(null);
        setDebugInfo(null);

        try {
            console.log("Enviando requisição de login para:", credentials.host);

            // Usar diretamente o método LoginWithCredentials
            const result = await window.go.backend.App.LoginWithCredentials(
                credentials.email,
                credentials.password,
                credentials.host
            );

            // Armazenar resultado para debugging
            setLoginResponse(result);
            console.log("Resultado do login:", result);

            if (result && result.success) {
                // Atualizar configuração com os dados recebidos
                const newConfig = {
                    authToken: result.token || "",
                    userID: result.userId || 0,
                    apiHost: credentials.host,
                    minutosPorDia: config.minutosPorDia || 480
                };

                console.log("Nova configuração após login:", newConfig);

                // Garantir que o token tenha sido recebido
                if (!result.token) {
                    console.error("Token não recebido na resposta de login");
                    toast.warning("Login bem-sucedido, mas o token não foi recebido.");
                    return;
                }

                // Garantir que o ID do usuário tenha sido recebido
                if (!result.userId) {
                    console.warn("ID do usuário não recebido na resposta de login");
                }

                // Atualizar o estado local
                setConfig(newConfig);

                // Atualizar as configurações e notificar o componente pai
                await SaveConfig(newConfig);
                toast.success("Login bem-sucedido! Token obtido e configurações atualizadas.");

                if (onConfigSaved) {
                    onConfigSaved();
                }

                // Limpar senha por segurança
                setCredentials(prev => ({ ...prev, password: '' }));

                // Fechar o formulário de login
                setShowLoginForm(false);
            } else {
                const errorMessage = result && result.message
                    ? result.message
                    : "Falha na autenticação. Verifique suas credenciais.";

                toast.error(errorMessage);
            }
        } catch (error) {
            console.error("Erro ao fazer login:", error);
            toast.error("Erro ao tentar login: " + (error.message || error));
            setDebugInfo(JSON.stringify(error, null, 2));
        } finally {
            setIsLoggingIn(false);
        }
    };

    const toggleDebugInfo = () => {
        setDebugInfo(prev => prev ? null : "Clique em Login para ver informações de debug");
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
                <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">Configurações</h1>
                <p className="text-gray-600 dark:text-gray-400">Configure sua conexão com o Teamwork</p>
            </div>

            <div className="card max-w-3xl mb-6">
                <div className="flex justify-between items-center mb-4">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Login com Email/Senha</h2>
                    <div className="flex">
                        <button
                            type="button"
                            onClick={toggleDebugInfo}
                            className="mr-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
                        >
                            <FiInfo className="w-5 h-5" />
                        </button>
                        <button
                            type="button"
                            onClick={() => setShowLoginForm(!showLoginForm)}
                            className="text-primary-600 hover:text-primary-700 dark:text-primary-500 dark:hover:text-primary-400"
                        >
                            {showLoginForm ? 'Ocultar' : 'Mostrar'}
                        </button>
                    </div>
                </div>

                {debugInfo && (
                    <div className="bg-gray-100 dark:bg-gray-800 p-3 rounded mb-4 overflow-auto max-h-60 text-xs font-mono">
                        <pre>{debugInfo}</pre>
                    </div>
                )}

                {showLoginForm && (
                    <form onSubmit={handleLogin} className="space-y-4">
                        <div>
                            <label htmlFor="loginHost" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Host do Teamwork *
                            </label>
                            <input
                                type="text"
                                id="loginHost"
                                name="host"
                                value={credentials.host}
                                onChange={handleCredentialsChange}
                                className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white"
                                placeholder="Ex: teamwork.empresa.com.br"
                                required
                            />
                            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                                Host da sua instância do Teamwork, sem http/https.
                            </p>
                        </div>

                        <div>
                            <label htmlFor="email" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Email *
                            </label>
                            <input
                                type="email"
                                id="email"
                                name="email"
                                value={credentials.email}
                                onChange={handleCredentialsChange}
                                className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white"
                                placeholder="seu.email@empresa.com"
                                required
                            />
                        </div>

                        <div>
                            <label htmlFor="password" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Senha *
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

                        <button
                            type="submit"
                            disabled={isLoggingIn}
                            className="btn-primary flex items-center justify-center"
                        >
                            {isLoggingIn ? (
                                <>
                                    <FiLoader className="w-4 h-4 mr-2 animate-spin" />
                                    Autenticando...
                                </>
                            ) : (
                                <>
                                    <FiUser className="w-4 h-4 mr-2" />
                                    Fazer Login
                                </>
                            )}
                        </button>
                    </form>
                )}
            </div>

            <div className="card max-w-3xl">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Configuração Manual</h2>
                <form onSubmit={handleSave}>
                    <div className="space-y-6">
                        <div>
                            <label htmlFor="authToken" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Token de Autenticação *
                            </label>
                            <div className="relative">
                                <input
                                    type={showToken ? "text" : "password"}
                                    id="authToken"
                                    name="authToken"
                                    value={config.authToken}
                                    onChange={handleChange}
                                    className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 pr-10 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white"
                                    placeholder="Seu token de API do Teamwork"
                                    required
                                />
                                <button
                                    type="button"
                                    onClick={() => setShowToken(!showToken)}
                                    className="absolute inset-y-0 right-0 px-3 flex items-center"
                                >
                                    {showToken ? (
                                        <FiEyeOff className="w-5 h-5 text-gray-500 dark:text-gray-400" />
                                    ) : (
                                        <FiEye className="w-5 h-5 text-gray-500 dark:text-gray-400" />
                                    )}
                                </button>
                            </div>
                            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                                Encontre seu token em Configurações → API do Teamwork.
                            </p>
                        </div>

                        <div>
                            <label htmlFor="userID" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                ID do Usuário *
                            </label>
                            <input
                                type="number"
                                id="userID"
                                name="userID"
                                value={config.userID}
                                onChange={handleChange}
                                className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white"
                                placeholder="ID do seu usuário no Teamwork"
                                required
                            />
                            <div className="mt-2">
                                <button
                                    type="button"
                                    onClick={fetchUserId}
                                    className="text-sm text-primary-600 hover:text-primary-800 dark:text-primary-400 dark:hover:text-primary-300"
                                >
                                    Obter ID Automaticamente
                                </button>
                            </div>
                        </div>

                        <div>
                            <label htmlFor="apiHost" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Host da API *
                            </label>
                            <input
                                type="text"
                                id="apiHost"
                                name="apiHost"
                                value={config.apiHost}
                                onChange={handleChange}
                                className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white"
                                placeholder="Ex: teamwork.empresa.com.br"
                                required
                            />
                            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                                Host da sua instância do Teamwork, sem http/https.
                            </p>
                        </div>

                        <div>
                            <label htmlFor="minutosPorDia" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Minutos por Dia
                            </label>
                            <input
                                type="number"
                                id="minutosPorDia"
                                name="minutosPorDia"
                                value={config.minutosPorDia}
                                onChange={handleChange}
                                className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white"
                                placeholder="480"
                            />
                            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                                Quantidade padrão de minutos por dia (480 = 8 horas).
                            </p>
                        </div>

                        <div className="flex flex-col sm:flex-row space-y-3 sm:space-y-0 sm:space-x-3">
                            <button
                                type="button"
                                onClick={testConnection}
                                disabled={isTesting || isSaving}
                                className="btn-secondary flex items-center justify-center"
                            >
                                {isTesting ? (
                                    <>
                                        <FiLoader className="w-4 h-4 mr-2 animate-spin" />
                                        Testando...
                                    </>
                                ) : (
                                    <>
                                        <FiCheck className="w-4 h-4 mr-2" />
                                        Testar Conexão
                                    </>
                                )}
                            </button>

                            <button
                                type="submit"
                                disabled={isSaving || isTesting}
                                className="btn-primary flex items-center justify-center"
                            >
                                {isSaving ? (
                                    <>
                                        <FiLoader className="w-4 h-4 mr-2 animate-spin" />
                                        Salvando...
                                    </>
                                ) : (
                                    <>
                                        <FiSave className="w-4 h-4 mr-2" />
                                        Salvar Configurações
                                    </>
                                )}
                            </button>
                        </div>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default Config;