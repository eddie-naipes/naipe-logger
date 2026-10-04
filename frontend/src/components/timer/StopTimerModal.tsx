import {useEffect, useState} from 'react';
import {toast} from 'react-toastify';
import {FiSquare} from 'react-icons/fi';
import {PreviewTimerStop, StopTimer} from '@wailsjs/go/backend/App';
import Modal from '../Modal';
import {errMsg} from '../../utils/errors';
import {formatDateBR} from '../../utils/dates';
import {paraBinding} from '../../types/backend';
import {useTimeEntriesSignal} from '../../contexts/TimeEntriesContext';
import type {TimerEntry, TimerState} from './timerTypes';
import type {timer} from '@wailsjs/go/models';

interface StopTimerModalProps {
    isOpen: boolean;
    state: TimerState;
    onClose: () => void;
    onStopped: (state: TimerState) => void;
}

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white';

const dataBR = (ymd: string) => formatDateBR(ymd, 'dd/MM/yyyy', ymd);

// Revisão antes de lançar: um lançamento por dia (o cronômetro pode ter
// atravessado a meia-noite), com minutos editáveis e a descrição. Também
// permite descartar o tempo sem lançar.
const StopTimerModal = ({isOpen, state, onClose, onStopped}: StopTimerModalProps) => {
    const {notifyChanged} = useTimeEntriesSignal();
    const [entries, setEntries] = useState<TimerEntry[] | null>(null);
    const [descricao, setDescricao] = useState(state.description);
    const [enviando, setEnviando] = useState(false);
    const [confirmarDescarte, setConfirmarDescarte] = useState(false);

    useEffect(() => {
        if (!isOpen) return;
        setEntries(null);
        setConfirmarDescarte(false);
        setDescricao(state.description);
        PreviewTimerStop()
            .then(prev => setEntries(prev ?? []))
            .catch((error: unknown) => {
                toast.error('Erro ao calcular o tempo: ' + errMsg(error));
                setEntries([]);
            });
        // Só ao abrir: a descrição digitada não deve ser sobrescrita.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [isOpen]);

    const setMinutos = (date: string, minutes: number) => {
        setEntries(prev => (prev ?? []).map(e => (e.date === date ? {...e, minutes} : e)));
    };

    const invalido = !entries || entries.length === 0 || entries.some(e => !Number.isInteger(e.minutes) || e.minutes < 1 || e.minutes > 1440);

    const lancar = async () => {
        if (!entries || invalido) return;
        setEnviando(true);
        try {
            const res = await StopTimer(true, descricao, paraBinding<timer.Entry[]>(entries));
            onStopped(res.state);
            const total = entries.reduce((s, e) => s + e.minutes, 0);
            toast.success(`Lançado: ${total} min em ${state.taskName}`);
            notifyChanged();
            onClose();
        } catch (error) {
            toast.error('Erro ao lançar o cronômetro: ' + errMsg(error));
            // Dias que já foram lançados antes da falha precisam aparecer.
            notifyChanged();
        } finally {
            setEnviando(false);
        }
    };

    const descartar = async () => {
        setEnviando(true);
        try {
            const res = await StopTimer(false, '', []);
            onStopped(res.state);
            toast.info('Cronômetro descartado.');
            onClose();
        } catch (error) {
            toast.error('Erro ao descartar o cronômetro: ' + errMsg(error));
        } finally {
            setEnviando(false);
        }
    };

    return (
        <Modal
            isOpen={isOpen}
            onClose={onClose}
            closeDisabled={enviando}
            title="Parar cronômetro"
            icon={<FiSquare className="w-5 h-5 mr-2" aria-hidden="true"/>}
            size="md"
            footer={(
                <>
                    {confirmarDescarte ? (
                        <button type="button" className="btn-danger mr-auto" onClick={() => void descartar()} disabled={enviando}>
                            Confirmar descarte
                        </button>
                    ) : (
                        <button type="button" className="btn-secondary mr-auto" onClick={() => setConfirmarDescarte(true)} disabled={enviando}>
                            Descartar
                        </button>
                    )}
                    <button type="button" className="btn-secondary" onClick={onClose} disabled={enviando}>
                        Continuar contando
                    </button>
                    <button
                        type="button"
                        className="btn-primary disabled:opacity-50"
                        onClick={() => void lancar()}
                        disabled={invalido || enviando}
                    >
                        {enviando ? 'Lançando...' : 'Lançar'}
                    </button>
                </>
            )}
        >
            <p className="text-sm text-gray-700 dark:text-gray-300 mb-3">
                <span className="font-medium">{state.taskName}</span>
                {state.projectName && <span className="text-gray-500 dark:text-gray-400"> · {state.projectName}</span>}
            </p>

            {entries === null ? (
                <p className="text-sm text-gray-500 dark:text-gray-400 mb-3">Calculando...</p>
            ) : entries.length === 0 ? (
                <p className="text-sm text-amber-700 dark:text-amber-300 mb-3">Não há tempo a lançar.</p>
            ) : (
                <table className="w-full text-sm mb-4">
                    <thead>
                    <tr className="text-left text-gray-500 dark:text-gray-400">
                        <th className="pb-1 font-medium">Dia</th>
                        <th className="pb-1 font-medium">Início</th>
                        <th className="pb-1 font-medium">Minutos</th>
                    </tr>
                    </thead>
                    <tbody>
                    {entries.map(e => (
                        <tr key={e.date} className="text-gray-900 dark:text-white">
                            <td className="py-1">{dataBR(e.date)}</td>
                            <td className="py-1">{e.time.substring(0, 5)}</td>
                            <td className="py-1">
                                <input
                                    type="number"
                                    min={1}
                                    max={1440}
                                    aria-label={`Minutos de ${dataBR(e.date)}`}
                                    value={Number.isNaN(e.minutes) ? '' : e.minutes}
                                    onChange={ev => setMinutos(e.date, ev.target.valueAsNumber)}
                                    className={`${inputClass} w-24 p-1.5`}
                                />
                            </td>
                        </tr>
                    ))}
                    </tbody>
                </table>
            )}

            <label htmlFor="timer-parar-descricao" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Descrição
            </label>
            <textarea
                id="timer-parar-descricao"
                rows={2}
                value={descricao}
                onChange={e => setDescricao(e.target.value)}
                className={inputClass}
            />
            {state.loggedDates.length > 0 && (
                <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">
                    Já lançados numa tentativa anterior: {state.loggedDates.map(dataBR).join(', ')}.
                </p>
            )}
        </Modal>
    );
};

export default StopTimerModal;
