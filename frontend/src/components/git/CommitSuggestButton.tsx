import {useEffect, useRef, useState} from 'react';
import {FiAlertTriangle, FiGitCommit, FiLoader} from 'react-icons/fi';
import {BuildGitSuggestion, GetGitIntegration, GetGitSuggestion} from '@wailsjs/go/backend/App';
import Modal from '../Modal';
import {errMsg} from '../../utils/errors';
import {todayYMD} from '../../utils/dates';
import {paraBinding} from '../../types/backend';
import type {GitCommit, GitSuggestion} from './gitTypes';

// Botão que sugere a descrição de um lançamento a partir dos commits do dia
// nos repositórios configurados em Configurações > Integração com Git. Abre um
// diálogo com os commits para o usuário escolher quais entram e devolve o
// texto por onSuggest. Fica oculto enquanto a integração estiver desativada ou
// sem repositórios.
interface CommitSuggestButtonProps {
    // Dia dos commits (AAAA-MM-DD); vazio usa hoje. Pode ser trocado no diálogo.
    date?: string;
    onSuggest: (text: string) => void;
    disabled?: boolean;
}

const chave = (c: GitCommit) => `${c.repo}:${c.hash}`;

const CommitSuggestButton = ({date, onSuggest, disabled = false}: CommitSuggestButtonProps) => {
    const [visivel, setVisivel] = useState(false);
    const [aberto, setAberto] = useState(false);
    const [dia, setDia] = useState('');
    const [carregando, setCarregando] = useState(false);
    const [erro, setErro] = useState('');
    const [resultado, setResultado] = useState<GitSuggestion | null>(null);
    const [selecionados, setSelecionados] = useState<Set<string>>(new Set());
    const [previa, setPrevia] = useState('');
    // Descarta respostas de buscas/prévias antigas (o usuário trocou o dia ou
    // a seleção antes da anterior voltar).
    const pedidoRef = useRef(0);

    useEffect(() => {
        let ativo = true;
        GetGitIntegration()
            .then(cfg => {
                if (ativo) setVisivel(cfg.enabled && (cfg.repositories ?? []).length > 0);
            })
            .catch(() => {
                if (ativo) setVisivel(false);
            });
        return () => {
            ativo = false;
        };
    }, []);

    const buscar = async (novoDia: string) => {
        const pedido = ++pedidoRef.current;
        setDia(novoDia);
        setCarregando(true);
        setErro('');
        setResultado(null);
        try {
            const res = await GetGitSuggestion(novoDia);
            if (pedido !== pedidoRef.current) return;
            const commits = res.commits ?? [];
            setResultado({...res, commits, warnings: res.warnings ?? []});
            setSelecionados(new Set(commits.map(chave)));
            setPrevia(res.suggestion);
        } catch (e) {
            if (pedido === pedidoRef.current) setErro(errMsg(e));
        } finally {
            if (pedido === pedidoRef.current) setCarregando(false);
        }
    };

    const abrir = () => {
        setAberto(true);
        void buscar(date || todayYMD());
    };

    const alternar = async (commit: GitCommit) => {
        if (!resultado) return;
        const proximo = new Set(selecionados);
        if (proximo.has(chave(commit))) {
            proximo.delete(chave(commit));
        } else {
            proximo.add(chave(commit));
        }
        setSelecionados(proximo);

        const escolhidos = resultado.commits.filter(c => proximo.has(chave(c)));
        const pedido = ++pedidoRef.current;
        if (escolhidos.length === resultado.commits.length) {
            setPrevia(resultado.suggestion);
            return;
        }
        if (escolhidos.length === 0) {
            setPrevia('');
            return;
        }
        try {
            const texto = await BuildGitSuggestion(paraBinding(escolhidos));
            if (pedido === pedidoRef.current) setPrevia(texto);
        } catch (e) {
            if (pedido === pedidoRef.current) setErro(errMsg(e));
        }
    };

    const usar = () => {
        onSuggest(previa);
        setAberto(false);
    };

    if (!visivel) return null;

    const variosRepos = new Set(resultado?.commits.map(c => c.repo)).size > 1;

    return (
        <>
            <button
                type="button"
                onClick={abrir}
                disabled={disabled}
                title="Sugerir descrição a partir dos commits do dia"
                className="inline-flex items-center text-xs text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 disabled:opacity-50 disabled:cursor-not-allowed"
            >
                <FiGitCommit className="w-4 h-4 mr-1" aria-hidden="true"/>
                Sugerir pelos commits
            </button>

            <Modal
                isOpen={aberto}
                onClose={() => setAberto(false)}
                size="lg"
                zIndex="z-70"
                title="Descrição pelos commits"
                icon={<FiGitCommit className="w-5 h-5 mr-2" aria-hidden="true"/>}
                footer={
                    <>
                        <button
                            type="button"
                            onClick={() => setAberto(false)}
                            className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 dark:bg-gray-700 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-600"
                        >
                            Cancelar
                        </button>
                        <button
                            type="button"
                            onClick={usar}
                            disabled={carregando || previa === ''}
                            className="px-4 py-2 rounded-md text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 dark:bg-primary-700 dark:hover:bg-primary-800 disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            Usar descrição
                        </button>
                    </>
                }
            >
                <div className="space-y-4 text-sm">
                    <div>
                        <label htmlFor="git-suggest-day" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Dia dos commits
                        </label>
                        <input
                            id="git-suggest-day"
                            type="date"
                            value={dia}
                            onChange={(e) => {
                                if (e.target.value) void buscar(e.target.value);
                            }}
                            className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg p-2 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                        />
                    </div>

                    {carregando && (
                        <p className="flex items-center text-gray-500 dark:text-gray-400">
                            <FiLoader className="w-4 h-4 mr-2 animate-spin" aria-hidden="true"/>
                            Lendo os commits...
                        </p>
                    )}

                    {erro && (
                        <p role="alert" className="text-red-600 dark:text-red-400">{erro}</p>
                    )}

                    {resultado && resultado.warnings.length > 0 && (
                        <ul className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-amber-800 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-300 space-y-1">
                            {resultado.warnings.map(w => (
                                <li key={w} className="flex items-start">
                                    <FiAlertTriangle className="w-4 h-4 mr-2 mt-0.5 shrink-0" aria-hidden="true"/>
                                    <span className="break-all">{w}</span>
                                </li>
                            ))}
                        </ul>
                    )}

                    {resultado && !carregando && resultado.commits.length === 0 && (
                        <p className="text-gray-500 dark:text-gray-400">Nenhum commit seu neste dia.</p>
                    )}

                    {resultado && resultado.commits.length > 0 && (
                        <fieldset>
                            <legend className="font-medium text-gray-700 dark:text-gray-300 mb-2">
                                Commits a incluir
                            </legend>
                            <ul className="divide-y divide-gray-200 dark:divide-gray-700 rounded-lg border border-gray-200 dark:border-gray-700">
                                {resultado.commits.map(c => (
                                    <li key={chave(c)}>
                                        <label className="flex items-start gap-3 p-2 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50">
                                            <input
                                                type="checkbox"
                                                checked={selecionados.has(chave(c))}
                                                onChange={() => void alternar(c)}
                                                className="mt-1"
                                            />
                                            <span className="min-w-0 text-gray-900 dark:text-white">
                                                {c.subject}
                                                <span className="block text-xs text-gray-500 dark:text-gray-400">
                                                    {c.time} · <code>{c.hash}</code>{variosRepos && ` · ${c.repo}`}
                                                </span>
                                            </span>
                                        </label>
                                    </li>
                                ))}
                            </ul>
                        </fieldset>
                    )}

                    {previa && (
                        <div className="rounded-lg bg-blue-50 dark:bg-blue-900/20 p-3">
                            <p className="text-xs text-blue-600 dark:text-blue-400 mb-1">Prévia ({previa.length} caracteres)</p>
                            <p className="text-gray-900 dark:text-white wrap-break-word">{previa}</p>
                        </div>
                    )}
                </div>
            </Modal>
        </>
    );
};

export default CommitSuggestButton;
