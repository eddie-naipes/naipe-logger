// errMsg extrai uma mensagem legível de qualquer erro. No Wails v2 os erros dos
// bindings Go chegam como string, não como Error — `error.message` vira
// undefined e o toast mostrava "Erro desconhecido" ou "undefined".
export const errMsg = (error, fallback = 'Erro desconhecido') => {
    if (error === null || error === undefined || error === '') return fallback;
    if (typeof error === 'string') return error;
    if (error instanceof Error) return error.message || fallback;
    if (typeof error === 'object') {
        if (typeof error.message === 'string' && error.message) return error.message;
        if (typeof error.error === 'string' && error.error) return error.error;
        try {
            const json = JSON.stringify(error);
            return json && json !== '{}' ? json : fallback;
        } catch {
            return fallback;
        }
    }
    return String(error);
};
