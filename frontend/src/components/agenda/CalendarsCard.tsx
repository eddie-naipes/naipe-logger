import {type FormEvent, useEffect, useState} from 'react';
import {toast} from 'react-toastify';
import {FiCalendar, FiFileText, FiLoader, FiPlus, FiTrash2, FiX} from 'react-icons/fi';
import {
    AddAgendaCalendar,
    ClearAgendaFile,
    GetAgendaFile,
    ImportAgendaFile,
    ListAgendaCalendars,
    RemoveAgendaCalendar
} from '@wailsjs/go/backend/App';
import type {backend, config} from '@wailsjs/go/models';
import type {Dados} from '../../types/backend';
import {errMsg} from '../../utils/errors';

type AgendaCalendar = Dados<config.AgendaCalendar>;
type AgendaFileInfo = Dados<backend.AgendaFileInfo>;

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white';
const labelClass = 'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';

interface CalendarsCardProps {
    // Avisado quando as fontes mudam (para a página buscar de novo).
    onSourcesChanged?: () => void;
}

// Agendas cadastradas (link iCal privado, guardado no cofre do sistema) e o
// arquivo .ics importado na sessão. O link completo nunca volta do backend:
// a lista mostra só host + "…" + final.
const CalendarsCard = ({onSourcesChanged}: CalendarsCardProps) => {
    const [calendars, setCalendars] = useState<AgendaCalendar[]>([]);
    const [file, setFile] = useState<AgendaFileInfo | null>(null);
    const [name, setName] = useState('');
    const [link, setLink] = useState('');
    const [adding, setAdding] = useState(false);
    const [importing, setImporting] = useState(false);

    useEffect(() => {
        let cancelled = false;
        Promise.all([ListAgendaCalendars(), GetAgendaFile()])
            .then(([lista, arquivo]) => {
                if (cancelled) return;
                setCalendars(lista ?? []);
                setFile(arquivo ?? null);
            })
            .catch((err: unknown) => toast.error('Erro ao carregar as agendas: ' + errMsg(err)));
        return () => {
            cancelled = true;
        };
    }, []);

    const add = async (e: FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        if (!link.trim()) {
            toast.warning('Cole o link iCal privado da agenda.');
            return;
        }
        setAdding(true);
        try {
            const cal = await AddAgendaCalendar(name, link);
            setCalendars(prev => [...prev, cal]);
            setName('');
            setLink('');
            toast.success(`Agenda "${cal.name}" adicionada.`);
            onSourcesChanged?.();
        } catch (err) {
            toast.error('Erro ao adicionar a agenda: ' + errMsg(err));
        } finally {
            setAdding(false);
        }
    };

    const remove = async (cal: AgendaCalendar) => {
        if (!window.confirm(`Remover a agenda "${cal.name}"? O link será apagado do cofre do sistema.`)) return;
        try {
            await RemoveAgendaCalendar(cal.id);
            setCalendars(prev => prev.filter(c => c.id !== cal.id));
            toast.success('Agenda removida.');
            onSourcesChanged?.();
        } catch (err) {
            toast.error('Erro ao remover a agenda: ' + errMsg(err));
        }
    };

    const importFile = async () => {
        setImporting(true);
        try {
            const info = await ImportAgendaFile();
            if (!info) return;
            setFile(info);
            toast.success(`Arquivo "${info.name}" carregado com ${info.events} evento(s).`);
            onSourcesChanged?.();
        } catch (err) {
            toast.error('Erro ao importar o arquivo: ' + errMsg(err));
        } finally {
            setImporting(false);
        }
    };

    const clearFile = async () => {
        try {
            await ClearAgendaFile();
            setFile(null);
            onSourcesChanged?.();
        } catch (err) {
            toast.error('Erro ao descartar o arquivo: ' + errMsg(err));
        }
    };

    return (
        <div className="card">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                <FiCalendar className="w-5 h-5 mr-2" aria-hidden="true"/>
                Agendas
            </h2>

            {calendars.length === 0 ? (
                <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">Nenhuma agenda cadastrada.</p>
            ) : (
                <ul className="mb-4 divide-y divide-gray-200 dark:divide-gray-700">
                    {calendars.map(cal => (
                        <li key={cal.id} className="flex items-center justify-between py-2">
                            <div className="min-w-0">
                                <p className="text-sm font-medium text-gray-900 dark:text-white truncate">{cal.name}</p>
                                <p className="text-xs text-gray-500 dark:text-gray-400 truncate">{cal.maskedUrl}</p>
                            </div>
                            <button
                                type="button"
                                onClick={() => void remove(cal)}
                                aria-label={`Remover a agenda ${cal.name}`}
                                className="p-2 text-red-600 hover:bg-red-50 rounded-lg dark:text-red-400 dark:hover:bg-red-900/20"
                            >
                                <FiTrash2 className="w-4 h-4" aria-hidden="true"/>
                            </button>
                        </li>
                    ))}
                </ul>
            )}

            <form onSubmit={e => void add(e)} className="space-y-3">
                <div>
                    <label htmlFor="agenda-nome" className={labelClass}>Nome (opcional)</label>
                    <input id="agenda-nome" value={name} onChange={e => setName(e.target.value)}
                           placeholder="Ex.: Trabalho" className={inputClass}/>
                </div>
                <div>
                    <label htmlFor="agenda-link" className={labelClass}>Link iCal privado</label>
                    <input id="agenda-link" type="password" autoComplete="off" value={link}
                           onChange={e => setLink(e.target.value)}
                           placeholder="https://… .ics" className={inputClass}/>
                    <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                        Google Agenda: Configurações da agenda → “Endereço secreto no formato iCal”. Outlook:
                        Configurações → Calendário → Calendários compartilhados → Publicar (link .ics). O link dá
                        acesso de leitura à agenda: ele fica só no cofre de credenciais do sistema.
                    </p>
                </div>
                <button type="submit" disabled={adding}
                        className="btn-primary w-full flex items-center justify-center disabled:opacity-50">
                    {adding
                        ? <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/>
                        : <FiPlus className="w-4 h-4 mr-2" aria-hidden="true"/>}
                    {adding ? 'Validando...' : 'Adicionar agenda'}
                </button>
            </form>

            <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
                {file ? (
                    <div className="flex items-center justify-between text-sm text-gray-700 dark:text-gray-300">
                        <span className="flex items-center min-w-0">
                            <FiFileText className="w-4 h-4 mr-2 shrink-0" aria-hidden="true"/>
                            <span className="truncate">{file.name} ({file.events} eventos)</span>
                        </span>
                        <button type="button" onClick={() => void clearFile()} aria-label="Descartar o arquivo importado"
                                className="p-2 hover:bg-gray-100 rounded-lg dark:hover:bg-gray-700">
                            <FiX className="w-4 h-4" aria-hidden="true"/>
                        </button>
                    </div>
                ) : (
                    <button type="button" onClick={() => void importFile()} disabled={importing}
                            className="btn-secondary w-full flex items-center justify-center disabled:opacity-50">
                        <FiFileText className="w-4 h-4 mr-2" aria-hidden="true"/>
                        {importing ? 'Abrindo...' : 'Importar arquivo .ics'}
                    </button>
                )}
                <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    O arquivo vale só enquanto o app estiver aberto.
                </p>
            </div>
        </div>
    );
};

export default CalendarsCard;
