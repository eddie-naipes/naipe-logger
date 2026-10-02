import {useEffect, useState} from 'react';
import {addMonths, isAfter, startOfMonth} from 'date-fns';
import {GetAllNonWorkingDays} from '@wailsjs/go/backend/App';
import type {NonWorkingDay} from '../types/backend';
import {parseLocalDate} from '../utils/dates';

export type NonWorkingDaysMap = Record<string, NonWorkingDay | undefined>;

// Limite de meses consultados de uma vez, para um intervalo digitado errado
// (ex.: ano 2042) não disparar centenas de chamadas.
const MAX_MESES = 24;

// Dias não úteis (fins de semana e feriados) dos meses cobertos pelo período,
// indexados por 'YYYY-MM-DD'. Com flag cancelled: trocar o período rápido não
// deixa uma resposta antiga sobrescrever a nova.
const useNonWorkingDays = (startDate: string, endDate: string): NonWorkingDaysMap => {
    const [nonWorkingDays, setNonWorkingDays] = useState<NonWorkingDaysMap>({});

    useEffect(() => {
        let cancelled = false;

        const load = async () => {
            const inicio = parseLocalDate(startDate);
            const fim = parseLocalDate(endDate);
            if (!inicio || !fim || isAfter(inicio, fim)) {
                setNonWorkingDays({});
                return;
            }

            const meses: [number, number][] = [];
            for (let mes = startOfMonth(inicio); !isAfter(mes, fim) && meses.length < MAX_MESES; mes = addMonths(mes, 1)) {
                meses.push([mes.getFullYear(), mes.getMonth() + 1]);
            }

            try {
                const respostas = await Promise.all(
                    meses.map(([ano, mes]) => GetAllNonWorkingDays(ano, mes))
                );
                if (cancelled) return;

                // O binding devolve map genérico; o formato é o de api.NonWorkingDay.
                const mapa: NonWorkingDaysMap = {};
                respostas.forEach(dias => ((dias ?? []) as NonWorkingDay[]).forEach(day => {
                    mapa[day.date] = day;
                }));
                setNonWorkingDays(mapa);
            } catch (error) {
                console.error('Erro ao carregar dias não úteis:', error);
            }
        };

        void load();
        return () => {
            cancelled = true;
        };
    }, [startDate, endDate]);

    return nonWorkingDays;
};

export default useNonWorkingDays;
