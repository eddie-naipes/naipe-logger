// Tipos da integração com Git (backend/gitlog e config.GitIntegration).
import type {config, gitlog} from '@wailsjs/go/models';
import type {Dados} from '../../types/backend';

export type GitCommit = Dados<gitlog.Commit>;
export type GitSuggestion = Dados<gitlog.Result>;
export type GitIntegration = Dados<config.GitIntegration>;
