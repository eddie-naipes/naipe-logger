import {formatHoursMinutes} from '../../utils/time';

// Cores dos gráficos de Relatórios, validadas com o validador da skill de
// dataviz (contraste >= 3:1 e separação para daltonismo) sobre as superfícies
// do app: branco no tema claro e gray-800 (#1f2937) no escuro.
// Série 1 (horas / cobrável) = azul; série 2 (não cobrável) = laranja.
export const SERIE_1 = 'bg-[#2a78d6] dark:bg-[#3987e5]';
export const SERIE_2 = 'bg-[#eb6834] dark:bg-[#d95926]';
// Linha de referência (jornada) usa a tinta secundária, não uma cor de série.
export const REFERENCIA = 'bg-gray-700 dark:bg-gray-300';
export const GRADE = 'bg-gray-200 dark:bg-gray-700';

// formatMinutes: 450 -> "7h30".
export const formatMinutes = (minutes: number): string => formatHoursMinutes(minutes);

// formatSigned: saldo com sinal explícito ("+2h", "−7h30").
export const formatSigned = (minutes: number): string => {
    if (minutes === 0) return '0h';
    return `${minutes > 0 ? '+' : '−'}${formatHoursMinutes(Math.abs(minutes))}`;
};

// percent arredondado; 0 quando o total é 0.
export const percent = (part: number, total: number): number =>
    total > 0 ? Math.round((part / total) * 100) : 0;

// niceMax arredonda o topo do eixo para horas cheias "redondas" (em minutos).
export const niceMax = (maxMinutes: number): number => {
    const horas = Math.max(1, Math.ceil(maxMinutes / 60));
    const passo = horas <= 4 ? 1 : horas <= 10 ? 2 : horas <= 20 ? 4 : Math.ceil(horas / 5);
    return Math.ceil(horas / passo) * passo * 60;
};
