import {afterAll, beforeAll, describe, expect, it, vi} from 'vitest';
import {useRef, useState} from 'react';
import {render, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Modal, {type ModalProps} from './Modal';

// O jsdom não faz layout: offsetParent é sempre null e o Modal trataria todos
// os elementos como invisíveis. Aqui todo elemento ligado ao documento conta
// como visível.
let offsetParentOriginal: PropertyDescriptor | undefined;
beforeAll(() => {
    offsetParentOriginal = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'offsetParent');
    Object.defineProperty(HTMLElement.prototype, 'offsetParent', {
        configurable: true,
        get(this: HTMLElement) {
            return this.parentElement;
        },
    });
});
afterAll(() => {
    if (offsetParentOriginal) {
        Object.defineProperty(HTMLElement.prototype, 'offsetParent', offsetParentOriginal);
    }
});

const renderModal = (props: Partial<ModalProps> = {}) => {
    const onClose = vi.fn();
    const utils = render(
        <Modal isOpen onClose={onClose} title="Título do modal" footer={<button type="button">Rodapé</button>} {...props}>
            <input aria-label="Primeiro campo"/>
            <button type="button">Ação</button>
        </Modal>
    );
    return {...utils, onClose};
};

describe('Modal', () => {
    it('não renderiza nada fechado', () => {
        renderModal({isOpen: false});
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });

    it('é um diálogo modal rotulado pelo título', () => {
        renderModal();
        const dialog = screen.getByRole('dialog', {name: 'Título do modal'});
        expect(dialog).toHaveAttribute('aria-modal', 'true');
    });

    it('Esc fecha', async () => {
        const {onClose} = renderModal();
        await userEvent.keyboard('{Escape}');
        expect(onClose).toHaveBeenCalledTimes(1);
    });

    it('Esc não fecha com closeDisabled (operação em andamento)', async () => {
        const {onClose} = renderModal({closeDisabled: true});
        await userEvent.keyboard('{Escape}');
        expect(onClose).not.toHaveBeenCalled();
        expect(screen.getByRole('button', {name: 'Fechar'})).toBeDisabled();
    });

    it('o botão X fecha', async () => {
        const {onClose} = renderModal();
        await userEvent.click(screen.getByRole('button', {name: 'Fechar'}));
        expect(onClose).toHaveBeenCalledTimes(1);
    });

    it('clique no fundo fecha, clique dentro não', async () => {
        const {onClose} = renderModal();
        await userEvent.click(screen.getByRole('button', {name: 'Ação'}));
        expect(onClose).not.toHaveBeenCalled();

        const fundo = screen.getByRole('dialog').parentElement;
        expect(fundo).not.toBeNull();
        await userEvent.click(fundo!);
        expect(onClose).toHaveBeenCalledTimes(1);
    });

    it('clique no fundo não fecha com closeOnBackdrop=false', async () => {
        const {onClose} = renderModal({closeOnBackdrop: false});
        await userEvent.click(screen.getByRole('dialog').parentElement!);
        expect(onClose).not.toHaveBeenCalled();
    });

    it('ao abrir, o foco vai para o primeiro elemento focável do conteúdo', () => {
        renderModal();
        expect(screen.getByRole('textbox', {name: 'Primeiro campo'})).toHaveFocus();
    });

    it('data-autofocus tem prioridade', () => {
        render(
            <Modal isOpen onClose={vi.fn()} title="T">
                <input aria-label="A"/>
                <input aria-label="B" data-autofocus/>
            </Modal>
        );
        expect(screen.getByRole('textbox', {name: 'B'})).toHaveFocus();
    });

    it('initialFocusRef tem prioridade sobre todo o resto', () => {
        const ComRef = () => {
            const ref = useRef<HTMLButtonElement>(null);
            return (
                <Modal isOpen onClose={vi.fn()} title="T" initialFocusRef={ref}>
                    <input aria-label="A" data-autofocus/>
                    <button type="button" ref={ref}>Alvo</button>
                </Modal>
            );
        };
        render(<ComRef/>);
        expect(screen.getByRole('button', {name: 'Alvo'})).toHaveFocus();
    });

    it('ao fechar, devolve o foco a quem abriu', async () => {
        const Abridor = () => {
            const [aberto, setAberto] = useState(false);
            return (
                <>
                    <button type="button" onClick={() => setAberto(true)}>Abrir</button>
                    <Modal isOpen={aberto} onClose={() => setAberto(false)} title="T">
                        <input aria-label="Campo"/>
                    </Modal>
                </>
            );
        };
        render(<Abridor/>);
        const abrir = screen.getByRole('button', {name: 'Abrir'});

        await userEvent.click(abrir);
        expect(screen.getByRole('textbox', {name: 'Campo'})).toHaveFocus();

        await userEvent.keyboard('{Escape}');
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
        expect(abrir).toHaveFocus();
    });

    it('Tab no último elemento volta para o primeiro (foco preso)', async () => {
        renderModal();
        const fechar = screen.getByRole('button', {name: 'Fechar'});
        const rodape = screen.getByRole('button', {name: 'Rodapé'});

        rodape.focus();
        await userEvent.tab();
        expect(fechar).toHaveFocus();
    });

    it('Shift+Tab no primeiro elemento vai para o último', async () => {
        renderModal();
        const fechar = screen.getByRole('button', {name: 'Fechar'});
        const rodape = screen.getByRole('button', {name: 'Rodapé'});

        fechar.focus();
        await userEvent.tab({shift: true});
        expect(rodape).toHaveFocus();
    });

    it('Tab entre elementos do meio segue a ordem normal', async () => {
        renderModal();
        screen.getByRole('textbox', {name: 'Primeiro campo'}).focus();
        await userEvent.tab();
        expect(screen.getByRole('button', {name: 'Ação'})).toHaveFocus();
    });

    it('com modais aninhados, Esc fecha só o de cima', async () => {
        const fecharExterno = vi.fn();
        const fecharInterno = vi.fn();
        render(
            <>
                <Modal isOpen onClose={fecharExterno} title="Externo">
                    <button type="button">Externo</button>
                </Modal>
                <Modal isOpen onClose={fecharInterno} title="Interno">
                    <button type="button">Interno</button>
                </Modal>
            </>
        );

        await userEvent.keyboard('{Escape}');
        expect(fecharInterno).toHaveBeenCalledTimes(1);
        expect(fecharExterno).not.toHaveBeenCalled();
    });

    it('usa o onClose mais recente, sem closure velha', async () => {
        const primeiro = vi.fn();
        const segundo = vi.fn();
        const {rerender} = render(<Modal isOpen onClose={primeiro} title="T"><input aria-label="x"/></Modal>);
        rerender(<Modal isOpen onClose={segundo} title="T"><input aria-label="x"/></Modal>);

        await userEvent.keyboard('{Escape}');
        expect(segundo).toHaveBeenCalledTimes(1);
        expect(primeiro).not.toHaveBeenCalled();
    });
});
