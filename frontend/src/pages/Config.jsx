import React, { useState, useEffect } from 'react';
import { toast } from 'react-toastify';
import { FiSave, FiCheck, FiLoader, FiEye, FiEyeOff } from 'react-icons/fi';

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
        } else {
            setConfig({ ...config, [name]: value });
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
                // Cria uma configuração atualizada com o ID obtido
                const updatedConfig = {...config, userID: userId};

                // Atualiza o estado local
                setConfig(updatedConfig);

                // Salva a configuração no backend automaticamente
                await SaveConfig(updatedConfig);

                toast.success(`ID do usuário (${userId}) obtido e configuração salva com sucesso!`);

                // Informa ao componente pai que a configuração foi salva
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

            <div className="card max-w-3xl">
                <form onSubmit={handleSave}>
                    <div className="space-y-6">
                        {/* Token de Autenticação */}
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

                        {/* ID do Usuário */}
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

                        {/* Host da API */}
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

                        {/* Minutos por Dia */}
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

                        {/* Ações */}
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