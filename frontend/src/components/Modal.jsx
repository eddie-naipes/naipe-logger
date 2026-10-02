import React, {useEffect, useId, useRef} from 'react';
import {FiX} from 'react-icons/fi';

const FOCAVEIS = [
    'a[href]',
    'button:not([disabled])',
    'input:not([disabled]):not([type="hidden"])',
    'select:not([disabled])',
    'textarea:not([disabled])',
    '[tabindex]:not([tabindex="-1"])'
].join(',');

const LARGURAS = {
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
               }) => {
    const titleId = useId();
    const dialogRef = useRef(null);
    const contentRef = useRef(null);
    const onCloseRef = useRef(onClose);
    const closeDisabledRef = useRef(closeDisabled);

    onCloseRef.current = onClose;
    closeDisabledRef.current = closeDisabled;

    useEffect(() => {
        if (!isOpen) return undefined;

        const anterior = document.activeElement;
        const dialog = dialogRef.current;

        const alvo = initialFocusRef?.current
            || dialog?.querySelector('[data-autofocus]')
            || contentRef.current?.querySelector(FOCAVEIS)
            || dialog?.querySelector(FOCAVEIS)
            || dialog;
        alvo?.focus();

        const handleKeyDown = (e) => {
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

            const focaveis = Array.from(dialog.querySelectorAll(FOCAVEIS))
                .filter(el => el.offsetParent !== null || el === document.activeElement);
            if (focaveis.length === 0) {
                e.preventDefault();
                dialog.focus();
                return;
            }
            const primeiro = focaveis[0];
            const ultimo = focaveis[focaveis.length - 1];

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
            if (anterior && typeof anterior.focus === 'function' && document.contains(anterior)) {
                anterior.focus();
            }
        };
        // initialFocusRef é estável (ref); reexecutar só ao abrir/fechar.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [isOpen]);

    if (!isOpen) return null;

    const handleBackdrop = (e) => {
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
                className={`bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full ${LARGURAS[size] || LARGURAS.md} max-h-[90vh] flex flex-col focus:outline-none`}
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
