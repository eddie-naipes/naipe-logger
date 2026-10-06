import { FiMenu, FiAlertCircle } from 'react-icons/fi';
import TimerWidget from './timer/TimerWidget';

interface HeaderProps {
    onMenuClick: () => void;
    isConfigured: boolean;
}

const Header = ({ onMenuClick, isConfigured }: HeaderProps) => {
    return (
        <header className="bg-white border-b border-gray-200 dark:bg-gray-800 dark:border-gray-700">
            <div className="flex items-center justify-between h-16 px-4">
                {/* Botão de toggle do sidebar */}
                <button
                    type="button"
                    onClick={onMenuClick}
                    aria-label="Abrir ou fechar o menu"
                    className="p-2 rounded-md lg:hidden focus:outline-none focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-600"
                >
                    <FiMenu className="w-6 h-6 text-gray-500 dark:text-gray-400" />
                </button>

                {/* Título para desktop */}
                <div className="hidden lg:block">
                    <h1 className="text-lg font-semibold text-gray-800 dark:text-white">
                        Teamwork Time Logger
                    </h1>
                </div>

                {/* Espaço flexível */}
                <div className="flex-grow"></div>

                {isConfigured && <TimerWidget />}

                {/* Alerta de configuração */}
                {!isConfigured && (
                    <div className="flex items-center text-yellow-600 dark:text-yellow-500 mr-4">
                        <FiAlertCircle className="w-5 h-5 mr-2" />
                        <span className="text-sm">Configuração necessária</span>
                    </div>
                )}
            </div>
        </header>
    );
};

export default Header;