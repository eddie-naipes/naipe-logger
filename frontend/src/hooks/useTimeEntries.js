import {useCallback, useEffect, useRef, useState} from 'react';
import {toast} from 'react-toastify';
import {GetTimeEntriesForPeriodV2} from '../../wailsjs/go/backend/App';
import {todayYMD} from '../utils/dates';
import {errMsg} from '../utils/errors';

// Carrega as entradas de tempo do período para o Gerenciador de Apontamentos.
// Cada carga recebe um id: ao mudar datas/"incluir deletados" rapidamente, uma
// resposta antiga que chegue por último é descartada.
const useTimeEntries = (isOpen) => {
    const [entries, setEntries] = useState([]);
    const [loading, setLoading] = useState(false);
    const [dateRange, setDateRange] = useState(() => ({
        startDate: todayYMD(),
        endDate: todayYMD()
    }));
    const [showDeleted, setShowDeleted] = useState(false);

    const requestIdRef = useRef(0);

    useEffect(() => () => {
        // Invalida respostas pendentes ao desmontar.
        requestIdRef.current++;
    }, []);

    // reload devolve true quando a lista foi recarregada.
    const reload = useCallback(async () => {
        const requestId = ++requestIdRef.current;
        setLoading(true);
        try {
            const timeEntries = await GetTimeEntriesForPeriodV2(
                dateRange.startDate,
                dateRange.endDate,
                showDeleted
            );
            if (requestId !== requestIdRef.current) return false;
            setEntries(timeEntries || []);
            return true;
        } catch (error) {
            if (requestId !== requestIdRef.current) return false;
            console.error('Erro ao carregar entradas de tempo:', error);
            toast.error('Erro ao carregar entradas de tempo: ' + errMsg(error));
            return false;
        } finally {
            if (requestId === requestIdRef.current) setLoading(false);
        }
    }, [dateRange.startDate, dateRange.endDate, showDeleted]);

    useEffect(() => {
        if (isOpen) {
            reload();
        }
    }, [isOpen, reload]);

    return {entries, loading, dateRange, setDateRange, showDeleted, setShowDeleted, reload};
};

export default useTimeEntries;
