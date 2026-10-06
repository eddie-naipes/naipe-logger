import {useCallback, useEffect, useState} from 'react';
import {toast} from 'react-toastify';
import {DiscardTimer, GetTimerState, PauseTimer, ResumeTimer} from '@wailsjs/go/backend/App';
import {EventsOn} from '@wailsjs/runtime/runtime';
import {
    EVENT_TIMER_CHANGED,
    EVENT_TIMER_LOGGED,
    type TimerLoggedEvent,
    type TimerState
} from '../components/timer/timerTypes';
import {useTimeEntriesSignal} from '../contexts/TimeEntriesContext';
import {errMsg} from '../utils/errors';

// Estado do cronômetro sincronizado com o backend: carga inicial, evento
// "timer:changed" (inclusive quando a mudança vem de uma notificação) e
// "timer:logged" (parado e lançado pela notificação).
const useTimer = () => {
    const [state, setState] = useState<TimerState | null>(null);
    const [busy, setBusy] = useState(false);
    const {notifyChanged} = useTimeEntriesSignal();

    useEffect(() => {
        let cancelado = false;
        GetTimerState()
            .then(st => {
                if (!cancelado) setState(st);
            })
            .catch((error: unknown) => console.error('Erro ao ler o cronômetro:', error));

        const offChanged = EventsOn(EVENT_TIMER_CHANGED, (st: TimerState) => setState(st));
        const offLogged = EventsOn(EVENT_TIMER_LOGGED, (ev: TimerLoggedEvent) => {
            setState(ev.result.state);
            if (ev.logged) {
                toast.success('Cronômetro parado e lançado no Teamwork.');
                notifyChanged();
            } else {
                toast.error('Não foi possível lançar o cronômetro: ' + (ev.error ?? 'erro desconhecido'));
                if (ev.result.results.some(r => r.success)) notifyChanged();
            }
        });

        return () => {
            cancelado = true;
            offChanged();
            offLogged();
        };
    }, [notifyChanged]);

    const run = useCallback(async (acao: () => Promise<TimerState>, falha: string) => {
        setBusy(true);
        try {
            setState(await acao());
        } catch (error) {
            toast.error(`${falha}: ${errMsg(error)}`);
        } finally {
            setBusy(false);
        }
    }, []);

    const pause = useCallback(() => run(PauseTimer, 'Erro ao pausar o cronômetro'), [run]);
    const resume = useCallback(() => run(ResumeTimer, 'Erro ao retomar o cronômetro'), [run]);
    const discard = useCallback(() => run(DiscardTimer, 'Erro ao descartar o cronômetro'), [run]);

    return {state, setState, busy, pause, resume, discard};
};

export default useTimer;
