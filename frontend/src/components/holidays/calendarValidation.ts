import type {Absence, CustomHoliday} from '../../types/backend';
import {parseLocalDate} from '../../utils/dates';

// Validações do formulário do calendário de trabalho. O backend
// (holidays.Validate) confere tudo de novo ao salvar; aqui é só para o
// usuário ver o erro antes.

export const CUSTOM_HOLIDAY_TYPES = [
    {value: 'municipal', label: 'Feriado municipal'},
    {value: 'ponte', label: 'Ponte'},
    {value: 'outro', label: 'Outra folga'}
] as const;

export const customHolidayTypeLabel = (type: string): string =>
    CUSTOM_HOLIDAY_TYPES.find(t => t.value === type)?.label ?? type;

const MAX_NOME = 120;
// Mesmo limite do backend: ausências de até dois anos.
const MAX_DIAS_AUSENCIA = 731;

const isYMD = (value: string): boolean => /^\d{4}-\d{2}-\d{2}$/.test(value) && parseLocalDate(value) !== null;

// validateCustomHoliday devolve a mensagem de erro ou null.
export const validateCustomHoliday = (holiday: CustomHoliday, existentes: readonly CustomHoliday[]): string | null => {
    if (!isYMD(holiday.date)) return 'Informe uma data válida.';
    const nome = holiday.name.trim();
    if (!nome) return 'Informe o nome do feriado.';
    if (nome.length > MAX_NOME) return `O nome deve ter no máximo ${MAX_NOME} caracteres.`;
    if (!CUSTOM_HOLIDAY_TYPES.some(t => t.value === holiday.type)) return 'Escolha o tipo do feriado.';

    const mesDia = holiday.date.slice(5);
    const repetido = existentes.some(h => holiday.recurring || h.recurring
        ? h.date.slice(5) === mesDia
        : h.date === holiday.date);
    if (repetido) return 'Já existe um feriado cadastrado nessa data.';
    return null;
};

// validateAbsence devolve a mensagem de erro ou null.
export const validateAbsence = (absence: Absence): string | null => {
    if (!isYMD(absence.start)) return 'Informe a data inicial.';
    if (!isYMD(absence.end)) return 'Informe a data final.';
    if (absence.end < absence.start) return 'A data final não pode ser anterior à inicial.';
    const inicio = parseLocalDate(absence.start);
    const fim = parseLocalDate(absence.end);
    if (inicio && fim && (fim.getTime() - inicio.getTime()) / 86_400_000 > MAX_DIAS_AUSENCIA) {
        return 'O período deve ter no máximo 2 anos.';
    }
    if (absence.description.trim().length > MAX_NOME) return `A descrição deve ter no máximo ${MAX_NOME} caracteres.`;
    return null;
};
