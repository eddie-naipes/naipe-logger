import React, {useEffect, useState} from 'react';
import {Route, Routes, useLocation, useNavigate} from 'react-router-dom';
import {toast, ToastContainer} from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';

import {GetAppSettings, SaveAppSettings} from '../wailsjs/go/backend/App';

import Sidebar from './components/Sidebar';
import Header from './components/Header';

import Dashboard from './pages/Dashboard.jsx';
import Config from './pages/Config';
import Tasks from './pages/Task';
import TimeLog from './pages/TimeLog';
import Templates from './pages/Templates';
import NotFound from './pages/NotFound';

import {ThemeContext} from './contexts/ThemeContext';

function App() {
    const navigate = useNavigate();
    const location = useLocation();
    const [darkMode, setDarkMode] = useState(false);
    const [sidebarOpen, setSidebarOpen] = useState(true);
    const [loading, setLoading] = useState(true);
    const [isConfigured, setIsConfigured] = useState(false);

    useEffect(() => {
        const loadSettings = async () => {
            try {
                const settings = await GetAppSettings();
                setDarkMode(settings.darkMode);

                await checkIfConfigured();
            } catch (error) {
                console.error('Erro ao carregar configurações:', error);
            } finally {
                setLoading(false);
            }
        };

        loadSettings();
    }, []);

    const checkIfConfigured = async () => {
        try {
            const config = await window.go.backend.App.GetConfig();
            const configured = config.authToken && config.authToken.length > 0 &&
                config.apiHost && config.apiHost.length > 0;

            setIsConfigured(configured);

            if (!configured && location.pathname !== '/config') {
                navigate('/config');
            }
        } catch (error) {
            console.error('Erro ao verificar configuração:', error);
            setIsConfigured(false);
        }
    };

    const toggleDarkMode = async () => {
        try {
            const newMode = !darkMode;
            setDarkMode(newMode);

            const settings = await GetAppSettings();
            settings.darkMode = newMode;
            await SaveAppSettings(settings);

            if (newMode) {
                document.documentElement.classList.add('dark');
            } else {
                document.documentElement.classList.remove('dark');
            }
        } catch (error) {
            console.error('Erro ao alternar tema:', error);
            toast.error('Erro ao salvar preferência de tema');
        }
    };

    useEffect(() => {
        if (darkMode) {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark');
        }
    }, [darkMode]);

    if (loading) {
        return (
            <div className="flex items-center justify-center h-screen bg-gray-100 dark:bg-gray-900">
                <div className="text-center">
                    <div
                        className="animate-spin-slow w-16 h-16 border-4 border-primary-600 border-t-transparent rounded-full mx-auto"></div>
                    <p className="mt-4 text-gray-700 dark:text-gray-300">Carregando...</p>
                </div>
            </div>
        );
    }

    return (
        <ThemeContext.Provider value={{darkMode, toggleDarkMode}}>
            <div className="flex h-full bg-gray-50 dark:bg-gray-900">
                {/* Sidebar */}
                <Sidebar
                    isOpen={sidebarOpen}
                    onClose={() => setSidebarOpen(false)}
                    isConfigured={isConfigured}
                />

                {/* Conteúdo principal */}
                <div className="flex flex-col flex-1 overflow-hidden">
                    <Header
                        onMenuClick={() => setSidebarOpen(!sidebarOpen)}
                        isConfigured={isConfigured}
                    />

                    <main className="flex-1 overflow-y-auto p-4">
                        <Routes>
                            <Route path="/" element={<Dashboard/>}/>
                            <Route path="/config" element={<Config onConfigSaved={checkIfConfigured}/>}/>
                            <Route path="/tasks" element={<Tasks/>}/>
                            <Route path="/timelog" element={<TimeLog/>}/>
                            <Route path="/templates" element={<Templates/>}/>
                            <Route path="*" element={<NotFound/>}/>
                        </Routes>
                    </main>
                </div>
            </div>

            {/* Notificações */}
            <ToastContainer
                position="bottom-right"
                autoClose={5000}
                hideProgressBar={false}
                newestOnTop
                closeOnClick
                rtl={false}
                pauseOnFocusLoss
                draggable
                pauseOnHover
                theme={darkMode ? 'dark' : 'light'}
            />
        </ThemeContext.Provider>
    );
}

export default App;