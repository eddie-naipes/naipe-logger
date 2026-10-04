import {useEffect, useMemo, useState} from 'react';
import {toast} from 'react-toastify';
import {FiPlay, FiSearch} from 'react-icons/fi';
import {GetSavedTasks, GetTasks, StartTimer} from '@wailsjs/go/backend/App';
import Modal from '../Modal';
import {errMsg} from '../../utils/errors';
import type {TimerState, TimerTaskRef} from './timerTypes';

interface StartTimerModalProps {
    isOpen: boolean;
    onClose: () => void;
    onStarted: (state: TimerState) => void;
}

type Origem = 'salvas' | 'teamwork';

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white';

const normalizar = (texto: string) => texto.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase();

// Escolha da tarefa para iniciar o cronômetro: as tarefas salvas ou uma busca
// nas tarefas do Teamwork (carregadas só quando o usuário abre essa aba).
const StartTimerModal = ({isOpen, onClose, onStarted}: StartTimerModalProps) => {
    const [origem, setOrigem] = useState<Origem>('salvas');
    const [salvas, setSalvas] = useState<TimerTaskRef[]>([]);
    const [remotas, setRemotas] = useState<TimerTaskRef[] | null>(null);
    const [carregandoRemotas, setCarregandoRemotas] = useState(false);
    const [busca, setBusca] = useState('');
    const [selecionada, setSelecionada] = useState<TimerTaskRef | null>(null);
    const [descricao, setDescricao] = useState('');
    const [faturavel, setFaturavel] = useState(true);
    const [iniciando, setIniciando] = useState(false);

    useEffect(() => {
        if (!isOpen) return;
        setSelecionada(null);
        setBusca('');
        setDescricao('');
        GetSavedTasks()
            .then(tasks => setSalvas((tasks ?? []).map(t => ({
                taskId: t.taskId,
                taskName: t.taskName,
                projectName: t.projectName
            }))))
            .catch((error: unknown) => toast.error('Erro ao carregar tarefas salvas: ' + errMsg(error)));
    }, [isOpen]);

    useEffect(() => {
        if (!isOpen || origem !== 'teamwork' || remotas !== null) return;
        setCarregandoRemotas(true);
        GetTasks()
            .then(tasks => setRemotas((tasks ?? []).map(t => ({
                taskId: t.id,
                taskName: t.name || t.content,
                projectName: t.projectName
            }))))
            .catch((error: unknown) => {
                toast.error('Erro ao buscar tarefas do Teamwork: ' + errMsg(error));
                setRemotas([]);
            })
            .finally(() => setCarregandoRemotas(false));
    }, [isOpen, origem, remotas]);

    const lista = useMemo(() => {
        const base = origem === 'salvas' ? salvas : (remotas ?? []);
        const termo = normalizar(busca.trim());
        const filtradas = termo
            ? base.filter(t => normalizar(`${t.taskName} ${t.projectName} ${t.taskId}`).includes(termo))
            : base;
        return filtradas.slice(0, 100);
    }, [origem, salvas, remotas, busca]);

    const iniciar = async () => {
        if (!selecionada) return;
        setIniciando(true);
        try {
            const st = await StartTimer(selecionada, descricao, faturavel);
            onStarted(st);
            toast.success(`Cronômetro iniciado: ${selecionada.taskName}`);
            onClose();
        } catch (error) {
            toast.error('Erro ao iniciar o cronômetro: ' + errMsg(error));
        } finally {
            setIniciando(false);
        }
    };

    const abaClass = (ativa: boolean) => `px-3 py-1.5 text-sm rounded-md ${ativa
        ? 'bg-primary-600 text-white'
        : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700'}`;

    return (
        <Modal
            isOpen={isOpen}
            onClose={onClose}
            closeDisabled={iniciando}
            title="Iniciar cronômetro"
            icon={<FiPlay className="w-5 h-5 mr-2" aria-hidden="true"/>}
            size="md"
            footer={(
                <>
                    <button type="button" className="btn-secondary" onClick={onClose} disabled={iniciando}>
                        Cancelar
                    </button>
                    <button
                        type="button"
                        className="btn-primary disabled:opacity-50"
                        onClick={() => void iniciar()}
                        disabled={!selecionada || iniciando}
                    >
                        {iniciando ? 'Iniciando...' : 'Iniciar'}
                    </button>
                </>
            )}
        >
            <div className="flex gap-2 mb-3" role="tablist" aria-label="Origem da tarefa">
                <button type="button" role="tab" aria-selected={origem === 'salvas'}
                        className={abaClass(origem === 'salvas')} onClick={() => setOrigem('salvas')}>
                    Tarefas salvas
                </button>
                <button type="button" role="tab" aria-selected={origem === 'teamwork'}
                        className={abaClass(origem === 'teamwork')} onClick={() => setOrigem('teamwork')}>
                    Buscar no Teamwork
                </button>
            </div>

            <div className="relative mb-3">
                <FiSearch className="absolute left-3 top-3 w-4 h-4 text-gray-400" aria-hidden="true"/>
                <input
                    type="search"
                    aria-label="Filtrar tarefas"
                    placeholder="Filtrar por tarefa, projeto ou ID"
                    value={busca}
                    onChange={e => setBusca(e.target.value)}
                    className={`${inputClass} pl-9`}
                />
            </div>

            <ul className="max-h-60 overflow-y-auto border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700 mb-4">
                {carregandoRemotas && origem === 'teamwork' && (
                    <li className="p-3 text-sm text-gray-500 dark:text-gray-400">Carregando tarefas...</li>
                )}
                {!carregandoRemotas && lista.length === 0 && (
                    <li className="p-3 text-sm text-gray-500 dark:text-gray-400">
                        {origem === 'salvas' && salvas.length === 0
                            ? 'Nenhuma tarefa salva. Use a aba "Buscar no Teamwork".'
                            : 'Nenhuma tarefa encontrada.'}
                    </li>
                )}
                {lista.map(t => (
                    <li key={t.taskId}>
                        <button
                            type="button"
                            onClick={() => setSelecionada(t)}
                            aria-pressed={selecionada?.taskId === t.taskId}
                            className={`w-full text-left p-3 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 ${selecionada?.taskId === t.taskId ? 'bg-primary-50 dark:bg-primary-900/30' : ''}`}
                        >
                            <span className="block font-medium text-gray-900 dark:text-white">{t.taskName}</span>
                            <span className="block text-xs text-gray-500 dark:text-gray-400">{t.projectName} · #{t.taskId}</span>
                        </button>
                    </li>
                ))}
            </ul>

            <label htmlFor="timer-descricao" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Descrição
            </label>
            <input
                id="timer-descricao"
                type="text"
                value={descricao}
                onChange={e => setDescricao(e.target.value)}
                placeholder="O que você vai fazer?"
                className={`${inputClass} mb-3`}
            />
            <label className="flex items-center text-sm text-gray-700 dark:text-gray-300">
                <input
                    type="checkbox"
                    checked={faturavel}
                    onChange={e => setFaturavel(e.target.checked)}
                    className="mr-2"
                />
                Faturável
            </label>
        </Modal>
    );
};

export default StartTimerModal;
