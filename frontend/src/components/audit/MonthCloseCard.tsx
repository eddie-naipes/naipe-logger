import {useState} from 'react';
import {Link} from 'react-router';
import {FiAlertCircle, FiCheckCircle, FiChevronRight, FiLoader} from 'react-icons/fi';
import useMonthAudit from '../../hooks/useMonthAudit';
import {currentMonth} from '../../utils/fillGaps';

// Card do Dashboard com o número de problemas do fechamento do mês atual e
// link para /fechamento. Falhas ficam discretas: o Dashboard já mostra os
// próprios erros de conexão.
const MonthCloseCard = () => {
    const [month] = useState(() => currentMonth());
    const {result, error} = useMonthAudit(month);
    const summary = result?.summary ?? null;
    const failed = error !== null;

    const pendentes = summary ? summary.errorCount + summary.warningCount : 0;
    const ok = summary !== null && pendentes === 0;

    let texto = 'Verificando o mês...';
    if (failed) texto = 'Não foi possível verificar o mês.';
    else if (ok) texto = 'Nenhum problema. Pronto para entregar!';
    else if (summary) texto = pendentes === 1 ? '1 problema para revisar' : `${pendentes} problemas para revisar`;

    return (
        <Link to="/fechamento"
              className="card !p-4 mb-6 flex items-center justify-between gap-3 hover:border-primary-400 dark:hover:border-primary-500 transition-colors"
              aria-label={`Fechamento do mês: ${texto}`}>
            <div className="flex items-center gap-3">
                {summary === null && !failed && <FiLoader className="w-6 h-6 animate-spin text-primary-600" aria-hidden="true"/>}
                {ok && <FiCheckCircle className="w-6 h-6 text-green-500" aria-hidden="true"/>}
                {(failed || (summary && !ok)) && (
                    <FiAlertCircle className={`w-6 h-6 ${summary && summary.errorCount > 0 ? 'text-red-500' : 'text-amber-500'}`} aria-hidden="true"/>
                )}
                <div>
                    <p className="text-sm font-semibold text-gray-900 dark:text-white">Fechamento do mês</p>
                    <p className="text-sm text-gray-600 dark:text-gray-400" data-testid="fechamento-resumo">{texto}</p>
                </div>
            </div>
            <FiChevronRight className="w-5 h-5 text-gray-400" aria-hidden="true"/>
        </Link>
    );
};

export default MonthCloseCard;
