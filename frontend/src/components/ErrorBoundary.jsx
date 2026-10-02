import React from 'react';
import {FiAlertTriangle, FiRefreshCw} from 'react-icons/fi';

// ErrorBoundary evita que um erro de renderização em uma página deixe a janela
// inteira em branco: mostra uma tela amigável com a opção de recarregar.
class ErrorBoundary extends React.Component {
    constructor(props) {
        super(props);
        this.state = {error: null};
    }

    static getDerivedStateFromError(error) {
        return {error};
    }

    componentDidCatch(error, info) {
        console.error('Erro inesperado na interface:', error, info?.componentStack);
    }

    componentDidUpdate(prevProps) {
        // Ao navegar para outra rota, dá uma nova chance à árvore.
        if (this.state.error && prevProps.resetKey !== this.props.resetKey) {
            this.setState({error: null});
        }
    }

    handleReload = () => {
        window.location.reload();
    };

    render() {
        if (!this.state.error) {
            return this.props.children;
        }

        const detalhe = this.state.error?.message || String(this.state.error);

        return (
            <div role="alert" className="flex flex-col items-center justify-center h-full p-6">
                <div className="card max-w-lg w-full text-center">
                    <FiAlertTriangle className="w-12 h-12 mx-auto text-amber-500 mb-4" aria-hidden="true"/>
                    <h1 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
                        Algo deu errado
                    </h1>
                    <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                        Ocorreu um erro inesperado ao exibir esta tela. Seus lançamentos já enviados
                        não foram afetados. Recarregue o aplicativo para continuar.
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-500 mb-6 break-words">
                        {detalhe}
                    </p>
                    <button
                        type="button"
                        onClick={this.handleReload}
                        className="btn-primary inline-flex items-center"
                    >
                        <FiRefreshCw className="w-4 h-4 mr-2" aria-hidden="true"/>
                        Recarregar
                    </button>
                </div>
            </div>
        );
    }
}

export default ErrorBoundary;
