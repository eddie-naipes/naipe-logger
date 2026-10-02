import {beforeEach, describe, expect, it, vi} from 'vitest';
import {render, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ErrorBoundary from './ErrorBoundary';

const Explode = ({erro}: {erro: unknown}): never => {
    throw erro;
};

describe('ErrorBoundary', () => {
    beforeEach(() => {
        // O React e o próprio boundary registram o erro no console; nos testes
        // isso só polui a saída.
        vi.spyOn(console, 'error').mockImplementation(() => {});
    });

    it('renderiza os filhos quando não há erro', () => {
        render(
            <ErrorBoundary>
                <p>conteúdo</p>
            </ErrorBoundary>
        );
        expect(screen.getByText('conteúdo')).toBeInTheDocument();
        expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    });

    it('mostra a tela de erro com a mensagem no lugar da árvore quebrada', () => {
        render(
            <ErrorBoundary>
                <Explode erro={new Error('quebrou ao renderizar')}/>
            </ErrorBoundary>
        );
        const alerta = screen.getByRole('alert');
        expect(alerta).toHaveTextContent('Algo deu errado');
        expect(alerta).toHaveTextContent('quebrou ao renderizar');
        expect(screen.getByRole('button', {name: 'Recarregar'})).toBeInTheDocument();
    });

    it('registra o erro no console', () => {
        render(
            <ErrorBoundary>
                <Explode erro={new Error('x')}/>
            </ErrorBoundary>
        );
        expect(console.error).toHaveBeenCalledWith(
            'Erro inesperado na interface:',
            expect.any(Error),
            expect.anything()
        );
    });

    it('aceita erros que não são Error', () => {
        render(
            <ErrorBoundary>
                <Explode erro="falha em texto"/>
            </ErrorBoundary>
        );
        expect(screen.getByRole('alert')).toHaveTextContent('falha em texto');
    });

    it('volta a renderizar os filhos quando resetKey muda (navegação)', () => {
        const {rerender} = render(
            <ErrorBoundary resetKey="/tasks">
                <Explode erro={new Error('x')}/>
            </ErrorBoundary>
        );
        expect(screen.getByRole('alert')).toBeInTheDocument();

        rerender(
            <ErrorBoundary resetKey="/config">
                <p>outra página</p>
            </ErrorBoundary>
        );
        expect(screen.queryByRole('alert')).not.toBeInTheDocument();
        expect(screen.getByText('outra página')).toBeInTheDocument();
    });

    it('Recarregar recarrega a janela', async () => {
        const reload = vi.fn();
        const locationOriginal = window.location;
        Object.defineProperty(window, 'location', {configurable: true, value: {...locationOriginal, reload}});
        try {
            render(
                <ErrorBoundary>
                    <Explode erro={new Error('x')}/>
                </ErrorBoundary>
            );
            await userEvent.click(screen.getByRole('button', {name: 'Recarregar'}));
            expect(reload).toHaveBeenCalledTimes(1);
        } finally {
            Object.defineProperty(window, 'location', {configurable: true, value: locationOriginal});
        }
    });
});
