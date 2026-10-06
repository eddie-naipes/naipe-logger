import {describe, expect, it} from 'vitest';
import {describeNonWorkingDay, isVacation, nonWorkingDayLabel} from './nonWorkingDays';

describe('describeNonWorkingDay', () => {
    it('mantém o texto antigo para feriado nacional e fim de semana', () => {
        expect(describeNonWorkingDay({type: 'holiday', name: 'Natal'})).toBe('Feriado: Natal');
        expect(describeNonWorkingDay({type: 'weekend', name: 'Saturday'})).toBe('Fim de semana');
    });

    it('rotula os tipos novos do calendário de trabalho', () => {
        expect(describeNonWorkingDay({type: 'state_holiday', name: 'Revolução Constitucionalista'}))
            .toBe('Feriado estadual: Revolução Constitucionalista');
        expect(describeNonWorkingDay({type: 'municipal', name: 'Aniversário da cidade'}))
            .toBe('Feriado municipal: Aniversário da cidade');
        expect(describeNonWorkingDay({type: 'bridge', name: 'Emenda'})).toBe('Ponte: Emenda');
        expect(describeNonWorkingDay({type: 'vacation', name: 'Férias/ausência'})).toBe('Férias/ausência');
        expect(describeNonWorkingDay({type: 'vacation', name: 'Férias de julho'})).toBe('Férias/ausência: Férias de julho');
    });

    it('tolera tipo desconhecido vindo de um backend mais novo', () => {
        expect(nonWorkingDayLabel('outro_tipo')).toBe('Dia não útil');
    });

    it('identifica férias', () => {
        expect(isVacation({type: 'vacation'})).toBe(true);
        expect(isVacation({type: 'holiday'})).toBe(false);
        expect(isVacation(undefined)).toBe(false);
    });
});
