import {useMemo, useState} from 'react';
import {FiChevronDown, FiChevronUp} from 'react-icons/fi';
import type {TaskTotal} from '../../types/backend';
import {formatMinutes, percent} from './format';

type Coluna = 'taskName' | 'projectName' | 'entryCount' | 'minutes' | 'billableMinutes';

const COLUNAS: readonly {key: Coluna; label: string; numeric: boolean}[] = [
    {key: 'taskName', label: 'Tarefa', numeric: false},
    {key: 'projectName', label: 'Projeto', numeric: false},
    {key: 'entryCount', label: 'Lançamentos', numeric: true},
    {key: 'minutes', label: 'Horas', numeric: true},
    {key: 'billableMinutes', label: 'Cobrável', numeric: true}
];

interface TaskTableProps {
    tasks: readonly TaskTotal[];
    totalMinutes: number;
}

// Tabela por tarefa ordenável pelo cabeçalho (clique alterna crescente e
// decrescente). Números começam em ordem decrescente; textos, crescente.
const TaskTable = ({tasks, totalMinutes}: TaskTableProps) => {
    const [coluna, setColuna] = useState<Coluna>('minutes');
    const [desc, setDesc] = useState(true);

    const ordenadas = useMemo(() => {
        const lista = [...tasks];
        lista.sort((a, b) => {
            const va = a[coluna];
            const vb = b[coluna];
            const cmp = typeof va === 'number' && typeof vb === 'number'
                ? va - vb
                : String(va).localeCompare(String(vb), 'pt-BR');
            return desc ? -cmp : cmp;
        });
        return lista;
    }, [tasks, coluna, desc]);

    const ordenarPor = (key: Coluna, numeric: boolean) => {
        if (key === coluna) {
            setDesc(d => !d);
        } else {
            setColuna(key);
            setDesc(numeric);
        }
    };

    if (tasks.length === 0) {
        return <p className="text-sm text-gray-500 dark:text-gray-400">Nenhum lançamento no período.</p>;
    }

    return (
        <div className="overflow-x-auto">
            <table className="w-full text-sm text-left text-gray-700 dark:text-gray-300">
                <thead className="text-xs uppercase text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-700">
                <tr>
                    {COLUNAS.map(c => {
                        const ativa = c.key === coluna;
                        return (
                            <th key={c.key} scope="col"
                                aria-sort={ativa ? (desc ? 'descending' : 'ascending') : 'none'}
                                className={`py-2 px-2 ${c.numeric ? 'text-right' : ''}`}>
                                <button type="button" onClick={() => ordenarPor(c.key, c.numeric)}
                                        className="inline-flex items-center uppercase hover:text-gray-900 dark:hover:text-white">
                                    {c.label}
                                    {ativa && (desc
                                        ? <FiChevronDown className="w-3 h-3 ml-0.5" aria-hidden="true"/>
                                        : <FiChevronUp className="w-3 h-3 ml-0.5" aria-hidden="true"/>)}
                                </button>
                            </th>
                        );
                    })}
                    <th scope="col" className="py-2 px-2 text-right">% do total</th>
                </tr>
                </thead>
                <tbody>
                {ordenadas.map(t => (
                    <tr key={`${t.projectId}-${t.taskId}-${t.taskName}`} className="border-b border-gray-100 dark:border-gray-700">
                        <td className="py-1.5 px-2 text-gray-900 dark:text-white">{t.taskName}</td>
                        <td className="py-1.5 px-2">{t.projectName}</td>
                        <td className="py-1.5 px-2 text-right tabular-nums">{t.entryCount}</td>
                        <td className="py-1.5 px-2 text-right tabular-nums">{formatMinutes(t.minutes)}</td>
                        <td className="py-1.5 px-2 text-right tabular-nums">{formatMinutes(t.billableMinutes)}</td>
                        <td className="py-1.5 px-2 text-right tabular-nums">{percent(t.minutes, totalMinutes)}%</td>
                    </tr>
                ))}
                </tbody>
            </table>
        </div>
    );
};

export default TaskTable;
