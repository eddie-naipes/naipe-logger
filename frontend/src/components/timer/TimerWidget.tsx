import {useEffect, useState} from 'react';
import {FiPause, FiPlay, FiSquare, FiWatch} from 'react-icons/fi';
import useTimer from '../../hooks/useTimer';
import StartTimerModal from './StartTimerModal';
import StopTimerModal from './StopTimerModal';
import {elapsedSeconds, formatElapsed, isTimerActive} from './timerTypes';

const iconButton = 'p-1.5 rounded-md text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-primary-500 disabled:opacity-50';

// Cronômetro no cabeçalho: tarefa e tempo correndo, com pausar/retomar/parar.
// O tempo é calculado no cliente a partir de startedAt e do acumulado, e só
// re-renderiza a cada segundo enquanto estiver rodando.
const TimerWidget = () => {
    const {state, setState, busy, pause, resume} = useTimer();
    const [agora, setAgora] = useState(() => Date.now());
    const [startOpen, setStartOpen] = useState(false);
    const [stopOpen, setStopOpen] = useState(false);

    const rodando = state?.running ?? false;
    useEffect(() => {
        if (!rodando) return undefined;
        setAgora(Date.now());
        const id = setInterval(() => setAgora(Date.now()), 1000);
        return () => clearInterval(id);
    }, [rodando]);

    if (!isTimerActive(state)) {
        return (
            <>
                <button
                    type="button"
                    onClick={() => setStartOpen(true)}
                    className="flex items-center text-sm px-3 py-1.5 mr-3 rounded-lg border border-gray-300 text-gray-700 hover:bg-gray-100 dark:border-gray-600 dark:text-gray-200 dark:hover:bg-gray-700"
                >
                    <FiWatch className="w-4 h-4 mr-1.5" aria-hidden="true"/>
                    Cronômetro
                </button>
                <StartTimerModal isOpen={startOpen} onClose={() => setStartOpen(false)} onStarted={setState}/>
            </>
        );
    }

    const segundos = elapsedSeconds(state, agora);

    return (
        <div
            className="flex items-center gap-2 mr-3 px-3 py-1 rounded-lg border border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-900"
            role="group"
            aria-label="Cronômetro"
        >
            <span
                className={`w-2 h-2 rounded-full ${state.running ? 'bg-green-500 animate-pulse' : 'bg-amber-500'}`}
                aria-hidden="true"
            />
            <div className="flex flex-col leading-tight max-w-[14rem]">
                <span className="text-sm font-medium text-gray-900 dark:text-white truncate" title={state.taskName}>
                    {state.taskName}
                </span>
                {state.projectName && (
                    <span className="text-xs text-gray-500 dark:text-gray-400 truncate">{state.projectName}</span>
                )}
            </div>
            <span className="font-mono text-sm tabular-nums text-gray-900 dark:text-white" aria-live="off" data-testid="timer-elapsed">
                {formatElapsed(segundos)}
            </span>
            {state.paused && <span className="text-xs text-amber-600 dark:text-amber-400">pausado</span>}
            {state.running ? (
                <button type="button" className={iconButton} onClick={() => void pause()} disabled={busy} aria-label="Pausar cronômetro" title="Pausar">
                    <FiPause className="w-4 h-4" aria-hidden="true"/>
                </button>
            ) : (
                <button type="button" className={iconButton} onClick={() => void resume()} disabled={busy} aria-label="Retomar cronômetro" title="Retomar">
                    <FiPlay className="w-4 h-4" aria-hidden="true"/>
                </button>
            )}
            <button type="button" className={iconButton} onClick={() => setStopOpen(true)} disabled={busy} aria-label="Parar cronômetro" title="Parar">
                <FiSquare className="w-4 h-4" aria-hidden="true"/>
            </button>
            <StopTimerModal isOpen={stopOpen} state={state} onClose={() => setStopOpen(false)} onStopped={setState}/>
        </div>
    );
};

export default TimerWidget;
