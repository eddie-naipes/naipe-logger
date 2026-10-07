import {useCallback, useEffect, useState} from 'react';
import {RunMonthAudit} from '@wailsjs/go/backend/App';
import {useOnTimeEntriesChanged} from '../contexts/TimeEntriesContext';
import {yearMonth} from '../components/audit/auditLabels';
import type {AuditResult} from '../types/backend';
import {errMsg} from '../utils/errors';

export interface MonthAuditState {
    result: AuditResult | null;
    loading: boolean;
    error: string | null;
    reload: () => void;
}

// Audita o mês ('YYYY-MM'). Trocar o mês rápido não deixa uma resposta antiga
// sobrescrever a nova (flag cancelled), e lançamentos criados/editados/apagados
// em qualquer tela reexecutam a auditoria.
const useMonthAudit = (month: string): MonthAuditState => {
    const [result, setResult] = useState<AuditResult | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [reloadToken, setReloadToken] = useState(0);

    const reload = useCallback(() => setReloadToken(t => t + 1), []);
    useOnTimeEntriesChanged(reload);

    useEffect(() => {
        let cancelled = false;
        const load = async () => {
            const ym = yearMonth(month);
            if (!ym) {
                setError('Mês inválido.');
                return;
            }
            setLoading(true);
            setError(null);
            try {
                const r = await RunMonthAudit(ym.year, ym.month);
                if (!cancelled) setResult(r);
            } catch (err) {
                if (!cancelled) {
                    console.error('Erro ao auditar o mês:', err);
                    setError(errMsg(err));
                }
            } finally {
                if (!cancelled) setLoading(false);
            }
        };
        void load();
        return () => {
            cancelled = true;
        };
    }, [month, reloadToken]);

    return {result, loading, error, reload};
};

export default useMonthAudit;
