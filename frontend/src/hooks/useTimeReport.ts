import {useCallback, useEffect, useState} from 'react';
import {GetTimeReportSummary} from '@wailsjs/go/backend/App';
import {useOnTimeEntriesChanged} from '../contexts/TimeEntriesContext';
import type {TimeReport} from '../types/backend';
import {errMsg} from '../utils/errors';
import {rangeError, type ReportRange} from '../utils/reportPeriods';

export interface TimeReportState {
    report: TimeReport | null;
    loading: boolean;
    error: string | null;
    reload: () => void;
}

// Carrega o resumo do período. Trocar o período rápido não deixa uma resposta
// antiga sobrescrever a nova (flag cancelled), e lançamentos criados/apagados
// em outras telas recarregam o relatório.
const useTimeReport = (range: ReportRange): TimeReportState => {
    const [report, setReport] = useState<TimeReport | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [reloadToken, setReloadToken] = useState(0);

    const reload = useCallback(() => setReloadToken(t => t + 1), []);
    useOnTimeEntriesChanged(reload);

    useEffect(() => {
        let cancelled = false;
        const invalido = rangeError(range);

        const load = async () => {
            if (invalido) {
                setReport(null);
                setError(invalido);
                return;
            }
            setLoading(true);
            setError(null);
            try {
                const result = await GetTimeReportSummary(range.startDate, range.endDate);
                if (!cancelled) setReport(result);
            } catch (err) {
                if (!cancelled) {
                    setReport(null);
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
    }, [range, reloadToken]);

    return {report, loading, error, reload};
};

export default useTimeReport;
