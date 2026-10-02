import {format, isValid, parseISO} from 'date-fns';

// Datas trafegam entre front e backend como 'YYYY-MM-DD' sem fuso. `new Date('2024-05-01')`
// interpreta a string como meia-noite UTC e, no Brasil (UTC-3), vira 30/04 às 21h;
// `toISOString()` faz o caminho inverso e pode adiantar o dia à noite. Todo parse e
// formatação de data do frontend deve passar por aqui.

// parseLocalDate converte 'YYYY-MM-DD' (ou ISO com horário) em Date local.
// Devolve null para valores vazios ou inválidos.
export const parseLocalDate = (value) => {
    if (!value) return null;
    if (value instanceof Date) return isValid(value) ? value : null;
    const parsed = parseISO(String(value));
    return isValid(parsed) ? parsed : null;
};

// toYMD formata uma Date local como 'YYYY-MM-DD'.
export const toYMD = (date) => format(date, 'yyyy-MM-dd');

// todayYMD é a data de hoje no fuso local.
export const todayYMD = () => toYMD(new Date());

// utcTimestampToYMD converte um timestamp em ms que representa um dia em UTC
// (formato da API de calendário do Teamwork, meia-noite UTC) em 'YYYY-MM-DD'.
// Aqui a leitura em UTC é a correta: a data local poderia cair no dia anterior.
export const utcTimestampToYMD = (timestamp) => {
    const date = new Date(timestamp);
    if (Number.isNaN(date.getTime())) return null;
    const y = date.getUTCFullYear();
    const m = String(date.getUTCMonth() + 1).padStart(2, '0');
    const d = String(date.getUTCDate()).padStart(2, '0');
    return `${y}-${m}-${d}`;
};

// formatDateBR formata uma data ('YYYY-MM-DD' ou Date) com o padrão do date-fns
// informado; devolve o fallback quando a data é inválida.
export const formatDateBR = (value, pattern = 'dd/MM/yyyy', fallback = '—', options) => {
    const date = parseLocalDate(value);
    if (!date) return fallback;
    return format(date, pattern, options);
};
