import type {timer} from '@wailsjs/go/models';
import type {Dados} from '../../types/backend';

// Tipos do cronômetro (backend/timer) e dos eventos que o backend emite.
export type TimerState = Dados<timer.State>;
export type TimerEntry = Dados<timer.Entry>;
export type TimerStopResult = Dados<timer.StopResult>;
export type TimerTaskRef = Dados<timer.TaskRef>;

export const EVENT_TIMER_CHANGED = 'timer:changed';
export const EVENT_TIMER_LOGGED = 'timer:logged';

// Payload de "timer:logged" (backend.TimerLogged): o cronômetro foi parado e
// lançado pela notificação "Parar e lançar".
export interface TimerLoggedEvent {
    logged: boolean;
    error?: string;
    result: TimerStopResult;
}

export const isTimerActive = (state: TimerState | null): state is TimerState =>
    state !== null && (state.running || state.paused);

// Segundos decorridos calculados no cliente: o acumulado dos trechos fechados
// mais o trecho em andamento desde startedAt.
export const elapsedSeconds = (state: TimerState, nowMs: number = Date.now()): number => {
    let total = state.accumulatedSeconds;
    if (state.running && state.startedAt) {
        const inicio = Date.parse(state.startedAt);
        if (!Number.isNaN(inicio) && nowMs > inicio) {
            total += Math.floor((nowMs - inicio) / 1000);
        }
    }
    return total;
};

// "1:02:03" (horas sem zero à esquerda) ou "02:03" abaixo de uma hora.
export const formatElapsed = (seconds: number): string => {
    const s = Math.max(0, Math.floor(seconds));
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    const sec = s % 60;
    const mm = String(m).padStart(2, '0');
    const ss = String(sec).padStart(2, '0');
    return h > 0 ? `${h}:${mm}:${ss}` : `${mm}:${ss}`;
};
