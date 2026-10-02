import {type MouseEvent, type ReactNode, type RefObject, useEffect, useId, useRef} from 'react';
import {FiX} from 'react-icons/fi';

const FOCAVEIS = [
    'a[href]',
    'button:not([disabled])',
    'input:not([disabled]):not([type="hidden"])',
    'select:not([disabled])',
    'textarea:not([disabled])',
    '[tabindex]:not([tabindex="-1"])'
].join(',');

export type ModalSize = 'sm' | 'md' | 'lg' | 'xl' | '2xl';

const LARGURAS: Record<ModalSize, string> = {
    sm: 'max-w-md',
    md: 'max-w-lg',
    lg: 'max-w-3xl',
    xl: 'max-w-6xl',
    '2xl': 'max-w-7xl'
};

// Modal acessível único do app: role="dialog" + aria-modal, título ligado por
// aria-labelledby, Esc fecha, foco vai para dentro ao abrir e volta para quem
// abriu ao fechar, e Tab/Shift+Tab ficam presos no diálogo.
//
// `closeDisabled` impede fechar (Esc, X e clique no fundo) durante operações em
// andamento, como uma gravação.
export interface ModalProps {
    isOpen: boolean;
    onClose?: () => void;
    title: ReactNode;
    icon?: ReactNode;
    children?: ReactNode;
    footer?: ReactNode;
    size?: ModalSize;
    closeDisabled?: boolean;
    closeOnBackdrop?: boolean;
    zIndex?: string;
    initialFocusRef?: RefObject<HTMLElement | null>;
}

const Modal = ({
                   isOpen,
                   onClose,
                   title,
                   icon,
                   children,
                   footer,
                   size = 'md',
                   closeDisabled = false,
                   closeOnBackdrop = true,
                   zIndex = 'z-50',
                   initialFocusRef
               }: ModalProps) => {
    const titleId = useId();
    const dialogRef = useRef<HTMLDivElement>(null);
    const contentRef = useRef<HTMLDivElement>(null);
    const onCloseRef = useRef(onClose);
    const closeDisabledRef = useRef(closeDisabled);

    // Refs com o valor mais recente: o efeito abaixo só roda ao abrir/fechar e
    // não pode usar uma versão velha de onClose/closeDisabled.
    useEffect(() => {
        onCloseRef.current = onClose;
        closeDisabledRef.current = closeDisabled;
    });

    useEffect(() => {
        if (!isOpen) return undefined;

        const anterior = document.activeElement instanceof HTMLElement ? document.activeElement : null;
        const dialog = dialogRef.current;

        const alvo = initialFocusRef?.current
            ?? dialog?.querySelector<HTMLElement>('[data-autofocus]')
            ?? contentRef.current?.querySelector<HTMLElement>(FOCAVEIS)
            ?? dialog?.querySelector<HTMLElement>(FOCAVEIS)
            ?? dialog;
        alvo?.focus();

        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                // Só o diálogo mais ao topo reage (modais aninhados).
                const abertos = document.querySelectorAll('[data-modal-dialog]');
                if (abertos[abertos.length - 1] !== dialog) return;
                e.stopPropagation();
                if (!closeDisabledRef.current) onCloseRef.current?.();
                return;
            }

            if (e.key !== 'Tab' || !dialog) return;
            const abertos = document.querySelectorAll('[data-modal-dialog]');
            if (abertos[abertos.length - 1] !== dialog) return;

            const focaveis = Array.from(dialog.querySelectorAll<HTMLElement>(FOCAVEIS))
                .filter(el => el.offsetParent !== null || el === document.activeElement);
            if (focaveis.length === 0) {
                e.preventDefault();
                dialog.focus();
                return;
            }
            const primeiro = focaveis[0];
            const ultimo = focaveis[focaveis.length - 1];
            if (!primeiro || !ultimo) return;

            if (e.shiftKey && (document.activeElement === primeiro || !dialog.contains(document.activeElement))) {
                e.preventDefault();
                ultimo.focus();
            } else if (!e.shiftKey && (document.activeElement === ultimo || !dialog.contains(document.activeElement))) {
                e.preventDefault();
                primeiro.focus();
            }
        };

        document.addEventListener('keydown', handleKeyDown);
        return () => {
            document.removeEventListener('keydown', handleKeyDown);
            if (anterior && document.contains(anterior)) {
                anterior.focus();
            }
        };
        // initialFocusRef é um objeto ref (estável): na prática o efeito só
        // roda ao abrir/fechar.
    }, [isOpen, initialFocusRef]);

    if (!isOpen) return null;

    const handleBackdrop = (e: MouseEvent<HTMLDivElement>) => {
        if (e.target === e.currentTarget && closeOnBackdrop && !closeDisabled) {
            onClose?.();
        }
    };

    return (
        <div
            className={`fixed inset-0 ${zIndex} flex items-center justify-center bg-black bg-opacity-50 p-4`}
            onMouseDown={handleBackdrop}
        >
            <div
                ref={dialogRef}
                role="dialog"
                aria-modal="true"
                aria-labelledby={titleId}
                tabIndex={-1}
                data-modal-dialog=""
                className={`bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full ${LARGURAS[size]} max-h-[90vh] flex flex-col focus:outline-none`}
            >
                <div className="flex justify-between items-center px-6 py-4 border-b border-gray-200 dark:border-gray-700">
                    <h2 id={titleId} className="text-lg font-semibold text-gray-900 dark:text-white flex items-center">
                        {icon}
                        {title}
                    </h2>
                    <button
                        type="button"
                        onClick={onClose}
                        disabled={closeDisabled}
                        aria-label="Fechar"
                        className="p-1 rounded-full text-gray-400 hover:text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700 dark:hover:text-gray-300 disabled:opacity-50"
                    >
                        <FiX className="w-5 h-5" aria-hidden="true"/>
                    </button>
                </div>

                <div ref={contentRef} className="p-6 overflow-y-auto flex-1">
                    {children}
                </div>

                {footer && (
                    <div className="px-6 py-3 border-t border-gray-200 dark:border-gray-700 flex justify-end items-center gap-3">
                        {footer}
                    </div>
                )}
            </div>
        </div>
    );
};

export default Modal;
