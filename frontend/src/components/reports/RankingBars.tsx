import {formatMinutes, percent, SERIE_1} from './format';

export interface RankingItem {
    key: string;
    label: string;
    sublabel?: string;
    minutes: number;
}

interface RankingBarsProps {
    title: string;
    items: readonly RankingItem[];
    totalMinutes: number;
    // limit: o que passar disso é somado em "Outros" (nunca vira cor nova).
    limit?: number;
}

// Ranking em barras horizontais de uma só série (azul): rótulo à esquerda,
// barra fina com ponta arredondada e valor na ponta, em tinta de texto.
const RankingBars = ({title, items, totalMinutes, limit = 8}: RankingBarsProps) => {
    const visiveis = items.slice(0, limit);
    const resto = items.slice(limit);
    const linhas: RankingItem[] = resto.length > 0
        ? [...visiveis, {
            key: '__outros__',
            label: `Outros (${resto.length})`,
            minutes: resto.reduce((s, i) => s + i.minutes, 0)
        }]
        : visiveis;
    const max = Math.max(1, ...linhas.map(l => l.minutes));

    return (
        <section className="card !p-4" aria-label={title}>
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-3">{title}</h3>
            {linhas.length === 0 ? (
                <p className="text-sm text-gray-500 dark:text-gray-400">Nenhum lançamento no período.</p>
            ) : (
                <ul className="space-y-2">
                    {linhas.map(item => (
                        <li key={item.key} title={`${item.label}: ${formatMinutes(item.minutes)} (${percent(item.minutes, totalMinutes)}%)`}>
                            <div className="flex justify-between text-xs mb-0.5">
                                <span className="truncate text-gray-800 dark:text-gray-200 pr-2">
                                    {item.label}
                                    {item.sublabel && <span className="text-gray-500 dark:text-gray-400"> · {item.sublabel}</span>}
                                </span>
                                <span className="flex-shrink-0 tabular-nums text-gray-700 dark:text-gray-300">
                                    {formatMinutes(item.minutes)} <span className="text-gray-500 dark:text-gray-400">({percent(item.minutes, totalMinutes)}%)</span>
                                </span>
                            </div>
                            <div className="h-3 rounded-r-[4px] bg-gray-100 dark:bg-gray-700/50" aria-hidden="true">
                                <div className={`h-3 rounded-r-[4px] ${SERIE_1}`} style={{width: `${(item.minutes / max) * 100}%`}}/>
                            </div>
                        </li>
                    ))}
                </ul>
            )}
        </section>
    );
};

export default RankingBars;
