import {lazy, Suspense, useCallback, useEffect, useMemo, useState} from 'react';
import {Route, Routes, useLocation, useNavigate} from 'react-router';
import {toast, ToastContainer} from 'react-toastify';
// O react-toastify 11 injeta o próprio CSS; não há mais import de ReactToastify.css.

import {GetAppSettings, IsConfigured, SaveAppSettings} from '@wailsjs/go/backend/App';

import Sidebar from './components/Sidebar';
import Header from './components/Header';
import ErrorBoundary from './components/ErrorBoundary';
import UpdateBanner from './components/UpdateBanner';
import StartupNotices from './components/StartupNotices';
import {TimeEntriesContext, type TimeEntriesSignal} from './contexts/TimeEntriesContext';

import Dashboard from './pages/Dashboard';

// O Dashboard é a tela inicial e vem no pacote principal; as demais páginas
// são carregadas na primeira visita, para o app abrir sem baixar relatórios,
// grade semanal etc. (o pacote único passava de 500 kB).
const Config = lazy(() => import('./pages/Config'));
const Tasks = lazy(() => import('./pages/Task'));
const TimeLog = lazy(() => import('./pages/TimeLog'));
const Templates = lazy(() => import('./pages/Templates'));
const CompletarPeriodo = lazy(() => import('./pages/CompletarPeriodo'));
const Semana = lazy(() => import('./pages/Semana'));
const Reports = lazy(() => import('./pages/Reports'));
const Fechamento = lazy(() => import('./pages/Fechamento'));
const NotFound = lazy(() => import('./pages/NotFound'));

const CarregandoPagina = () => (
    <div className="flex items-center justify-center py-24" role="status" aria-live="polite">
        <div className="animate-spin-slow w-10 h-10 border-4 border-primary-600 border-t-transparent rounded-full"/>
        <span className="sr-only">Carregando página...</span>
    </div>
);

import {ThemeContext} from './contexts/ThemeContext';
import {UpdateContext} from './contexts/UpdateContext';
import useUpdate from './hooks/useUpdate';
import useReminderNavigation from './hooks/useReminderNavigation';
import {errMsg} from './utils/errors';

function App() {
    const navigate = useNavigate();
    const location = useLocation();
    const [darkMode, setDarkMode] = useState(false);
    const [sidebarOpen, setSidebarOpen] = useState(true);
    const [loading, setLoading] = useState(true);
    const [isConfigured, setIsConfigured] = useState(false);
    const [autoCheckUpdates, setAutoCheckUpdates] = useState(false);
    // Lê a preferência só na inicialização: mudar o toggle na Config vale para
    // a próxima abertura do app.
    const updater = useUpdate({autoCheck: autoCheckUpdates});
    useReminderNavigation();

    const [timeEntriesVersion, setTimeEntriesVersion] = useState(0);
    const notifyTimeEntriesChanged = useCallback(() => setTimeEntriesVersion(v => v + 1), []);
    const timeEntriesSignal = useMemo<TimeEntriesSignal>(
        () => ({version: timeEntriesVersion, notifyChanged: notifyTimeEntriesChanged}),
        [timeEntriesVersion, notifyTimeEntriesChanged]
    );

    // O frontend só recebe um booleano; o token vive no cofre do sistema.
    const checkIfConfigured = useCallback(async (): Promise<void> => {
        try {
            setIsConfigured(await IsConfigured());
        } catch (error) {
            console.error('Erro ao verificar configuração:', error);
            setIsConfigured(false);
        }
    }, []);

    useEffect(() => {
        const loadSettings = async () => {
            try {
                const settings = await GetAppSettings();
                setDarkMode(settings.darkMode);
                setAutoCheckUpdates(settings.checkUpdatesOnStartup);

                await checkIfConfigured();
            } catch (error) {
                console.error('Erro ao carregar configurações:', error);
            } finally {
                setLoading(false);
            }
        };

        void loadSettings();
    }, [checkIfConfigured]);

    // Sem configuração, só a tela de Config faz sentido: as demais chamariam a
    // API sem token.
    useEffect(() => {
        if (!loading && !isConfigured && location.pathname !== '/config') {
            void navigate('/config');
        }
    }, [loading, isConfigured, location.pathname, navigate]);

    // useCallback + useMemo: sem isso o value do ThemeContext muda a cada render
    // do App e força todos os consumidores a renderizar de novo. A classe 'dark'
    // do <html> é aplicada pelo efeito abaixo.
    const toggleDarkMode = useCallback(async (): Promise<void> => {
        const newMode = !darkMode;
        setDarkMode(newMode);
        try {
            const settings = await GetAppSettings();
            settings.darkMode = newMode;
            await SaveAppSettings(settings);
        } catch (error) {
            console.error('Erro ao alternar tema:', error);
            toast.error('Erro ao salvar preferência de tema: ' + errMsg(error));
        }
    }, [darkMode]);

    // O contexto expõe toggleDarkMode como `() => void`: o erro já é tratado aqui.
    const themeValue = useMemo(() => ({
        darkMode,
        toggleDarkMode: () => void toggleDarkMode()
    }), [darkMode, toggleDarkMode]);

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
        <ThemeContext.Provider value={themeValue}>
            <UpdateContext.Provider value={updater}>
                <TimeEntriesContext.Provider value={timeEntriesSignal}>
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

                        <UpdateBanner/>

                        <main className="flex-1 overflow-y-auto p-4">
                            <ErrorBoundary resetKey={location.pathname}>
                                <Suspense fallback={<CarregandoPagina/>}>
                                <Routes>
                                    <Route path="/" element={<Dashboard/>}/>
                                    <Route path="/config" element={<Config onConfigSaved={() => void checkIfConfigured()}/>}/>
                                    <Route path="/tasks" element={<Tasks/>}/>
                                    <Route path="/timelog" element={<TimeLog/>}/>
                                    <Route path="/templates" element={<Templates/>}/>
                                    <Route path="/completar" element={<CompletarPeriodo/>}/>
                                    <Route path="/semana" element={<Semana/>}/>
                                    <Route path="/relatorios" element={<Reports/>}/>
                                    <Route path="/fechamento" element={<Fechamento/>}/>
                                    <Route path="*" element={<NotFound/>}/>
                                </Routes>
                                </Suspense>
                            </ErrorBoundary>
                        </main>
                    </div>
                </div>

                <StartupNotices/>

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
                </TimeEntriesContext.Provider>
            </UpdateContext.Provider>
        </ThemeContext.Provider>
    );
}

export default App;