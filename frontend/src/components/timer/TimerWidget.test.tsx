import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';
import {act, render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {toast} from 'react-toastify';
import {
    GetSavedTasks,
    GetTimerState,
    PauseTimer,
    PreviewTimerStop,
    ResumeTimer,
    StartTimer,
    StopTimer
} from '@wailsjs/go/backend/App';
import {EventsOn} from '@wailsjs/runtime/runtime';
import TimerWidget from './TimerWidget';
import {TimeEntriesContext} from '../../contexts/TimeEntriesContext';
import {EVENT_TIMER_CHANGED, EVENT_TIMER_LOGGED, elapsedSeconds, formatElapsed, type TimerState} from './timerTypes';

vi.mock('@wailsjs/go/backend/App', () => ({
    GetTimerState: vi.fn(),
    PauseTimer: vi.fn(),
    ResumeTimer: vi.fn(),
    DiscardTimer: vi.fn(),
    PreviewTimerStop: vi.fn(),
    StopTimer: vi.fn(),
    StartTimer: vi.fn(),
    GetSavedTasks: vi.fn(),
    GetTasks: vi.fn(),
}));

vi.mock('@wailsjs/runtime/runtime', () => ({
    EventsOn: vi.fn(),
}));

vi.mock('react-toastify', () => ({
    toast: {success: vi.fn(), warning: vi.fn(), error: vi.fn(), info: vi.fn()},
}));

const vazio: TimerState = {
    running: false, paused: false, taskId: 0, taskName: '', projectName: '', description: '', billable: false,
    startedAt: '', firstStartedAt: '', accumulatedSeconds: 0, elapsedSeconds: 0, segments: [], date: '', loggedDates: [],
};

const rodando = (extra: Partial<TimerState> = {}): TimerState => ({
    ...vazio,
    running: true,
    taskId: 77,
    taskName: 'Revisar PR',
    projectName: 'Logger',
    description: 'codando',
    billable: true,
    startedAt: new Date(Date.now() - 65_000).toISOString(),
    firstStartedAt: new Date(Date.now() - 65_000).toISOString(),
    date: '2026-09-09',
    ...extra,
});

let ouvintes: Map<string, (dados: unknown) => void>;

const renderWidget = (notifyChanged = vi.fn()) => {
    render(
        <TimeEntriesContext.Provider value={{version: 0, notifyChanged}}>
            <TimerWidget/>
        </TimeEntriesContext.Provider>
    );
    return notifyChanged;
};

describe('TimerWidget', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        ouvintes = new Map();
        vi.mocked(EventsOn).mockImplementation((evento, callback) => {
            ouvintes.set(evento, callback as (dados: unknown) => void);
            return () => ouvintes.delete(evento);
        });
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    it('sem cronômetro mostra o botão para iniciar e inicia por uma tarefa salva', async () => {
        vi.mocked(GetTimerState).mockResolvedValue(vazio as never);
        vi.mocked(GetSavedTasks).mockResolvedValue([
            {taskId: 77, taskName: 'Revisar PR', projectId: 1, projectName: 'Logger', entries: []},
        ] as never);
        vi.mocked(StartTimer).mockResolvedValue(rodando() as never);
        renderWidget();

        await userEvent.click(await screen.findByRole('button', {name: /cronômetro/i}));
        await userEvent.click(await screen.findByRole('button', {name: /revisar pr/i}));
        await userEvent.type(screen.getByLabelText('Descrição'), 'codando');
        await userEvent.click(screen.getByRole('button', {name: 'Iniciar'}));

        await waitFor(() => expect(StartTimer).toHaveBeenCalledWith(
            {taskId: 77, taskName: 'Revisar PR', projectName: 'Logger'}, 'codando', true));
        expect(await screen.findByTestId('timer-elapsed')).toBeInTheDocument();
    });

    it('exibe a tarefa e o tempo correndo a partir do startedAt', async () => {
        vi.mocked(GetTimerState).mockResolvedValue(rodando({accumulatedSeconds: 600}) as never);
        renderWidget();

        expect(await screen.findByText('Revisar PR')).toBeInTheDocument();
        expect(screen.getByText('Logger')).toBeInTheDocument();
        // 600s acumulados + ~65s do trecho atual.
        expect(screen.getByTestId('timer-elapsed').textContent).toMatch(/^11:0\d$/);
    });

    it('pausa, retoma e reage ao evento timer:changed', async () => {
        vi.mocked(GetTimerState).mockResolvedValue(rodando() as never);
        vi.mocked(PauseTimer).mockResolvedValue(rodando({running: false, paused: true, startedAt: '', accumulatedSeconds: 65}) as never);
        vi.mocked(ResumeTimer).mockResolvedValue(rodando() as never);
        renderWidget();

        await userEvent.click(await screen.findByRole('button', {name: 'Pausar cronômetro'}));
        expect(PauseTimer).toHaveBeenCalled();
        expect(await screen.findByText('pausado')).toBeInTheDocument();
        expect(screen.getByTestId('timer-elapsed').textContent).toBe('01:05');

        await userEvent.click(screen.getByRole('button', {name: 'Retomar cronômetro'}));
        expect(ResumeTimer).toHaveBeenCalled();
        await screen.findByRole('button', {name: 'Pausar cronômetro'});

        // Parado por uma notificação: o widget volta ao botão de iniciar.
        act(() => ouvintes.get(EVENT_TIMER_CHANGED)?.(vazio));
        expect(await screen.findByRole('button', {name: /cronômetro/i})).toBeInTheDocument();
    });

    it('ao parar, revisa minutos e descrição, lança e chama notifyChanged', async () => {
        vi.mocked(GetTimerState).mockResolvedValue(rodando() as never);
        vi.mocked(PreviewTimerStop).mockResolvedValue([
            {date: '2026-09-09', time: '23:30:00', minutes: 30, seconds: 1800},
            {date: '2026-09-10', time: '00:00:00', minutes: 20, seconds: 1200},
        ] as never);
        vi.mocked(StopTimer).mockResolvedValue({logged: true, results: [], state: vazio} as never);
        const notifyChanged = renderWidget();

        await userEvent.click(await screen.findByRole('button', {name: 'Parar cronômetro'}));
        const minutosDia10 = await screen.findByLabelText('Minutos de 10/09/2026');
        expect(screen.getByLabelText('Minutos de 09/09/2026')).toHaveValue(30);

        await userEvent.clear(minutosDia10);
        await userEvent.type(minutosDia10, '25');
        const descricao = screen.getByLabelText('Descrição');
        await userEvent.clear(descricao);
        await userEvent.type(descricao, 'revisado');
        await userEvent.click(screen.getByRole('button', {name: 'Lançar'}));

        await waitFor(() => expect(StopTimer).toHaveBeenCalledWith(true, 'revisado', [
            {date: '2026-09-09', time: '23:30:00', minutes: 30, seconds: 1800},
            {date: '2026-09-10', time: '00:00:00', minutes: 25, seconds: 1200},
        ]));
        expect(notifyChanged).toHaveBeenCalledTimes(1);
        expect(toast.success).toHaveBeenCalled();
        expect(await screen.findByRole('button', {name: /cronômetro/i})).toBeInTheDocument();
    });

    it('descarta sem lançar após confirmação e sem notifyChanged', async () => {
        vi.mocked(GetTimerState).mockResolvedValue(rodando() as never);
        vi.mocked(PreviewTimerStop).mockResolvedValue([{date: '2026-09-09', time: '10:00:00', minutes: 5, seconds: 300}] as never);
        vi.mocked(StopTimer).mockResolvedValue({logged: false, results: [], state: vazio} as never);
        const notifyChanged = renderWidget();

        await userEvent.click(await screen.findByRole('button', {name: 'Parar cronômetro'}));
        await userEvent.click(await screen.findByRole('button', {name: 'Descartar'}));
        expect(StopTimer).not.toHaveBeenCalled();
        await userEvent.click(screen.getByRole('button', {name: 'Confirmar descarte'}));

        await waitFor(() => expect(StopTimer).toHaveBeenCalledWith(false, '', []));
        expect(notifyChanged).not.toHaveBeenCalled();
    });

    it('erro ao lançar mantém o modal aberto e mostra o erro', async () => {
        vi.mocked(GetTimerState).mockResolvedValue(rodando() as never);
        vi.mocked(PreviewTimerStop).mockResolvedValue([{date: '2026-09-09', time: '10:00:00', minutes: 5, seconds: 300}] as never);
        vi.mocked(StopTimer).mockRejectedValue('API não configurada');
        renderWidget();

        await userEvent.click(await screen.findByRole('button', {name: 'Parar cronômetro'}));
        await userEvent.click(await screen.findByRole('button', {name: 'Lançar'}));

        await waitFor(() => expect(toast.error).toHaveBeenCalledWith(expect.stringContaining('API não configurada')));
        expect(screen.getByRole('dialog')).toBeInTheDocument();
    });

    it('lançamento pela notificação (timer:logged) avisa e chama notifyChanged', async () => {
        vi.mocked(GetTimerState).mockResolvedValue(rodando() as never);
        const notifyChanged = renderWidget();
        await screen.findByText('Revisar PR');

        act(() => ouvintes.get(EVENT_TIMER_LOGGED)?.({logged: true, result: {logged: true, results: [], state: vazio}}));
        expect(notifyChanged).toHaveBeenCalledTimes(1);
        expect(toast.success).toHaveBeenCalled();
    });
});

describe('cálculo do tempo', () => {
    it('formata e soma o trecho em andamento', () => {
        expect(formatElapsed(65)).toBe('01:05');
        expect(formatElapsed(3723)).toBe('1:02:03');
        const inicio = Date.parse('2026-09-09T10:00:00-03:00');
        const st = {...vazio, running: true, startedAt: '2026-09-09T10:00:00-03:00', accumulatedSeconds: 100};
        expect(elapsedSeconds(st, inicio + 30_000)).toBe(130);
        expect(elapsedSeconds({...st, running: false, paused: true}, inicio + 30_000)).toBe(100);
    });
});
