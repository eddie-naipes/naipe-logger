import {createContext, useContext, useEffect, useRef} from 'react';

// Sinal global de "os lançamentos mudaram no Teamwork". Quem cria, edita ou
// apaga lançamentos chama notifyChanged(); quem exibe dados derivados deles
// (dashboard, calendário) reage à mudança de version. Sem isso, apagar pelo
// gerenciador aberto na Sidebar não atualizava o dashboard até trocar de aba.
export interface TimeEntriesSignal {
    version: number;
    notifyChanged: () => void;
}

// Valor padrão inerte: componentes fora do provider (testes, telas isoladas)
// continuam funcionando, só sem a propagação.
export const TimeEntriesContext = createContext<TimeEntriesSignal>({
    version: 0,
    notifyChanged: () => {}
});

export const useTimeEntriesSignal = (): TimeEntriesSignal => useContext(TimeEntriesContext);

// O Teamwork leva um instante para refletir lançamentos recém-criados ou
// apagados nos totais e no calendário; recarregar na hora trazia o dado velho.
export const TIME_ENTRIES_REFRESH_DELAY_MS = 1000;

// Chama onChange a cada notificação, mas não na montagem: a carga inicial é
// responsabilidade do próprio componente. Notificações seguidas dentro do
// atraso viram uma única recarga.
export const useOnTimeEntriesChanged = (onChange: () => void, delayMs = TIME_ENTRIES_REFRESH_DELAY_MS): void => {
    const {version} = useTimeEntriesSignal();
    const inicial = useRef(version);
    const callback = useRef(onChange);

    useEffect(() => {
        callback.current = onChange;
    }, [onChange]);

    useEffect(() => {
        if (version === inicial.current) return;
        const timer = setTimeout(() => callback.current(), delayMs);
        return () => clearTimeout(timer);
    }, [version, delayMs]);
};
