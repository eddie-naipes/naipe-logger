import {useEffect, useState} from 'react';
import {toast} from 'react-toastify';
import {FiFolderPlus, FiGitBranch, FiLoader, FiSave, FiTrash2} from 'react-icons/fi';
import {GetGitIntegration, SaveGitIntegration, SelectGitRepository} from '@wailsjs/go/backend/App';
import {errMsg} from '../../utils/errors';
import {paraBinding} from '../../types/backend';
import type {GitIntegration} from '../git/gitTypes';

const inputClass = 'bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-primary-500 focus:border-primary-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:text-white';

// Seção "Integração com Git": repositórios cujos commits do dia viram
// sugestão de descrição nos lançamentos (botão "Sugerir pelos commits").
const GitIntegrationSection = () => {
    const [cfg, setCfg] = useState<GitIntegration | null>(null);
    const [salvando, setSalvando] = useState(false);
    const [escolhendo, setEscolhendo] = useState(false);

    useEffect(() => {
        GetGitIntegration()
            .then(c => setCfg({...c, repositories: c.repositories ?? []}))
            .catch((error: unknown) => {
                console.error('Erro ao carregar integração com Git:', error);
                setCfg({enabled: true, repositories: [], authorEmail: ''});
            });
    }, []);

    if (!cfg) return null;

    const salvar = async (novo: GitIntegration, aviso = 'Integração com Git salva.') => {
        setSalvando(true);
        try {
            await SaveGitIntegration(paraBinding(novo));
            setCfg(novo);
            toast.success(aviso);
        } catch (error) {
            toast.error('Não foi possível salvar: ' + errMsg(error));
        } finally {
            setSalvando(false);
        }
    };

    const adicionar = async () => {
        setEscolhendo(true);
        try {
            const pasta = await SelectGitRepository();
            if (!pasta) return;
            if (cfg.repositories.some(r => r.toLowerCase() === pasta.toLowerCase())) {
                toast.info('Esse repositório já está na lista.');
                return;
            }
            await salvar({...cfg, repositories: [...cfg.repositories, pasta]}, 'Repositório adicionado.');
        } catch (error) {
            toast.error('Pasta recusada: ' + errMsg(error));
        } finally {
            setEscolhendo(false);
        }
    };

    const remover = (repo: string) =>
        void salvar({...cfg, repositories: cfg.repositories.filter(r => r !== repo)}, 'Repositório removido.');

    return (
        <div className="card max-w-md mx-auto mt-6">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-1 flex items-center">
                <FiGitBranch className="w-5 h-5 mr-2" aria-hidden="true"/>
                Integração com Git
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                Sugere a descrição dos lançamentos a partir dos seus commits do dia nestes repositórios.
                Os prefixos como &quot;feat:&quot; e &quot;fix(api):&quot; são removidos.
            </p>

            <label className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300 mb-4">
                <input
                    type="checkbox"
                    checked={cfg.enabled}
                    disabled={salvando}
                    onChange={(e) => void salvar({...cfg, enabled: e.target.checked})}
                />
                Ativar sugestão pelos commits
            </label>

            <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Repositórios</h3>
            {cfg.repositories.length === 0 ? (
                <p className="text-sm text-gray-500 dark:text-gray-400 mb-3">Nenhum repositório adicionado.</p>
            ) : (
                <ul className="mb-3 divide-y divide-gray-200 dark:divide-gray-700 rounded-lg border border-gray-200 dark:border-gray-700">
                    {cfg.repositories.map(repo => (
                        <li key={repo} className="flex items-center justify-between gap-2 p-2">
                            <span className="text-xs font-mono break-all text-gray-700 dark:text-gray-200">{repo}</span>
                            <button
                                type="button"
                                onClick={() => remover(repo)}
                                disabled={salvando}
                                aria-label={`Remover ${repo}`}
                                title="Remover repositório"
                                className="text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 disabled:opacity-50"
                            >
                                <FiTrash2 className="w-4 h-4" aria-hidden="true"/>
                            </button>
                        </li>
                    ))}
                </ul>
            )}
            <button
                type="button"
                onClick={() => void adicionar()}
                disabled={salvando || escolhendo}
                className="w-full btn-secondary flex items-center justify-center disabled:opacity-50 mb-4"
            >
                {escolhendo
                    ? <FiLoader className="w-5 h-5 mr-2 animate-spin" aria-hidden="true"/>
                    : <FiFolderPlus className="w-5 h-5 mr-2" aria-hidden="true"/>}
                Adicionar repositório
            </button>

            <form
                onSubmit={(e) => {
                    e.preventDefault();
                    void salvar({...cfg, authorEmail: cfg.authorEmail.trim()});
                }}
            >
                <label htmlFor="git-author-email" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    E-mail do autor (opcional)
                </label>
                <div className="flex gap-2">
                    <input
                        id="git-author-email"
                        type="email"
                        value={cfg.authorEmail}
                        onChange={(e) => setCfg({...cfg, authorEmail: e.target.value})}
                        placeholder="Padrão: git config user.email de cada repositório"
                        className={inputClass}
                        disabled={salvando}
                    />
                    <button
                        type="submit"
                        disabled={salvando}
                        aria-label="Salvar e-mail do autor"
                        className="btn-primary flex items-center disabled:opacity-50"
                    >
                        <FiSave className="w-4 h-4" aria-hidden="true"/>
                    </button>
                </div>
            </form>
        </div>
    );
};

export default GitIntegrationSection;
