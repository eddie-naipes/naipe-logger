// errMsg extrai uma mensagem legível de qualquer erro. No Wails v2 os erros dos
// bindings Go chegam como string, não como Error — `error.message` vira
// undefined e o toast mostrava "Erro desconhecido" ou "undefined".
export const errMsg = (error: unknown, fallback = 'Erro desconhecido'): string => {
    if (error === null || error === undefined || error === '') return fallback;
    if (typeof error === 'string') return error;
    if (error instanceof Error) return error.message || fallback;
    if (typeof error === 'object') {
        const {message, error: inner} = error as {message?: unknown; error?: unknown};
        if (typeof message === 'string' && message) return message;
        if (typeof inner === 'string' && inner) return inner;
        try {
            const json = JSON.stringify(error);
            return json && json !== '{}' ? json : fallback;
        } catch {
            return fallback;
        }
    }
    return String(error as string | number | boolean | bigint | symbol);
};
