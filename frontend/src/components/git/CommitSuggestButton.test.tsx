import {beforeEach, describe, expect, it, vi} from 'vitest';
import {render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {BuildGitSuggestion, GetGitIntegration, GetGitSuggestion} from '@wailsjs/go/backend/App';
import type {config, gitlog} from '@wailsjs/go/models';
import CommitSuggestButton from './CommitSuggestButton';

vi.mock('@wailsjs/go/backend/App', () => ({
    BuildGitSuggestion: vi.fn(),
    GetGitIntegration: vi.fn(),
    GetGitSuggestion: vi.fn(),
}));

const integracao = (repos: string[], enabled = true) =>
    ({enabled, repositories: repos, authorEmail: ''}) as config.GitIntegration;

const sugestao = {
    suggestion: 'corrige login; ajusta relatorio',
    commits: [
        {repo: 'app', hash: 'abc1234', time: '09:00', subject: 'feat: corrige login'},
        {repo: 'app', hash: 'def5678', time: '15:30', subject: 'fix: ajusta relatorio'},
    ],
    warnings: ['C:\\nao-repo: não é um repositório git'],
} as gitlog.Result;

describe('CommitSuggestButton', () => {
    beforeEach(() => {
        vi.mocked(GetGitIntegration).mockResolvedValue(integracao(['C:\\repos\\app']));
        vi.mocked(GetGitSuggestion).mockResolvedValue(sugestao);
        vi.mocked(BuildGitSuggestion).mockResolvedValue('ajusta relatorio');
    });

    it('fica oculto sem repositórios configurados', async () => {
        vi.mocked(GetGitIntegration).mockResolvedValue(integracao([]));
        render(<CommitSuggestButton date="2026-09-15" onSuggest={vi.fn()}/>);
        await waitFor(() => expect(GetGitIntegration).toHaveBeenCalled());
        expect(screen.queryByRole('button', {name: /commits/i})).not.toBeInTheDocument();
    });

    it('fica oculto com a integração desativada', async () => {
        vi.mocked(GetGitIntegration).mockResolvedValue(integracao(['C:\\r'], false));
        render(<CommitSuggestButton date="2026-09-15" onSuggest={vi.fn()}/>);
        await waitFor(() => expect(GetGitIntegration).toHaveBeenCalled());
        expect(screen.queryByRole('button', {name: /commits/i})).not.toBeInTheDocument();
    });

    it('busca os commits do dia e devolve a sugestão completa', async () => {
        const onSuggest = vi.fn();
        render(<CommitSuggestButton date="2026-09-15" onSuggest={onSuggest}/>);

        await userEvent.click(await screen.findByRole('button', {name: /Sugerir pelos commits/}));
        expect(GetGitSuggestion).toHaveBeenCalledWith('2026-09-15');

        const dialogo = await screen.findByRole('dialog', {name: 'Descrição pelos commits'});
        expect(await screen.findByText('feat: corrige login')).toBeInTheDocument();
        expect(dialogo).toHaveTextContent('não é um repositório git');
        expect(dialogo).toHaveTextContent('corrige login; ajusta relatorio');

        await userEvent.click(screen.getByRole('button', {name: 'Usar descrição'}));
        expect(onSuggest).toHaveBeenCalledWith('corrige login; ajusta relatorio');
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });

    it('monta a descrição só com os commits escolhidos', async () => {
        const onSuggest = vi.fn();
        render(<CommitSuggestButton date="2026-09-15" onSuggest={onSuggest}/>);
        await userEvent.click(await screen.findByRole('button', {name: /Sugerir pelos commits/}));

        await userEvent.click(await screen.findByRole('checkbox', {name: /corrige login/}));
        expect(BuildGitSuggestion).toHaveBeenCalledWith([sugestao.commits[1]]);
        await screen.findByText('ajusta relatorio');

        await userEvent.click(screen.getByRole('button', {name: 'Usar descrição'}));
        expect(onSuggest).toHaveBeenCalledWith('ajusta relatorio');
    });

    it('sem nenhum commit escolhido não deixa usar', async () => {
        render(<CommitSuggestButton date="2026-09-15" onSuggest={vi.fn()}/>);
        await userEvent.click(await screen.findByRole('button', {name: /Sugerir pelos commits/}));
        await userEvent.click(await screen.findByRole('checkbox', {name: /corrige login/}));
        await userEvent.click(screen.getByRole('checkbox', {name: /ajusta relatorio/}));
        expect(screen.getByRole('button', {name: 'Usar descrição'})).toBeDisabled();
    });

    it('mostra o erro do backend', async () => {
        vi.mocked(GetGitSuggestion).mockRejectedValue('git não encontrado no PATH');
        render(<CommitSuggestButton date="2026-09-15" onSuggest={vi.fn()}/>);
        await userEvent.click(await screen.findByRole('button', {name: /Sugerir pelos commits/}));
        expect(await screen.findByRole('alert')).toHaveTextContent('git não encontrado no PATH');
        expect(screen.getByRole('button', {name: 'Usar descrição'})).toBeDisabled();
    });

    it('troca o dia e busca de novo', async () => {
        render(<CommitSuggestButton date="2026-09-15" onSuggest={vi.fn()}/>);
        await userEvent.click(await screen.findByRole('button', {name: /Sugerir pelos commits/}));
        const campo = await screen.findByLabelText('Dia dos commits');
        await userEvent.clear(campo);
        await userEvent.type(campo, '2026-09-16');
        await waitFor(() => expect(GetGitSuggestion).toHaveBeenLastCalledWith('2026-09-16'));
    });
});
