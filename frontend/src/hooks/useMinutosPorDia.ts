import {useEffect, useState} from 'react';
import {GetPublicConfig} from '@wailsjs/go/backend/App';
import {MINUTOS_POR_DIA_PADRAO} from '../utils/time';

// Jornada diária (minutos) configurada pelo usuário. O valor fica num cache de
// módulo para que calendário e dashboard não refaçam a chamada a cada montagem;
// setMinutosPorDiaCache avisa os componentes montados quando a Config muda.
let cache: number | null = null;
let pendente: Promise<number> | null = null;
const ouvintes = new Set<(minutos: number) => void>();

const carregar = (): Promise<number> => {
    if (!pendente) {
        pendente = GetPublicConfig()
            .then(cfg => {
                const valor = Number(cfg?.minutosPorDia);
                cache = valor > 0 ? valor : MINUTOS_POR_DIA_PADRAO;
                return cache;
            })
            .catch((error: unknown) => {
                console.error('Erro ao carregar a jornada diária:', error);
                pendente = null;
                return MINUTOS_POR_DIA_PADRAO;
            });
    }
    return pendente;
};

export const setMinutosPorDiaCache = (minutos: number): void => {
    cache = minutos;
    pendente = Promise.resolve(minutos);
    ouvintes.forEach(fn => fn(minutos));
};

const useMinutosPorDia = (): number => {
    const [minutos, setMinutos] = useState(cache ?? MINUTOS_POR_DIA_PADRAO);

    useEffect(() => {
        let cancelled = false;
        ouvintes.add(setMinutos);
        if (cache === null) {
            void carregar().then(valor => {
                if (!cancelled) setMinutos(valor);
            });
        }
        return () => {
            cancelled = true;
            ouvintes.delete(setMinutos);
        };
    }, []);

    return minutos;
};

export default useMinutosPorDia;
