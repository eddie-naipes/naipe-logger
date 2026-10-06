import type {AuditIssue} from '../../types/backend';

// Valores de audit.IssueType / audit.Action / audit.Severity no Go
// (backend/audit/audit.go). O Wails os tipa como string.
export type AuditIssueType =
    | 'incomplete_day'
    | 'missing_description'
    | 'generic_description'
    | 'duplicate'
    | 'non_working_day'
    | 'over_daily_limit'
    | 'no_task';

export type AuditAction = 'complete_period' | 'edit' | 'delete_duplicates' | 'edit_or_delete' | 'view_day';

interface TypeInfo {
    title: string;
    hint: string;
}

const TIPOS: Record<AuditIssueType, TypeInfo> = {
    incomplete_day: {
        title: 'Dias incompletos',
        hint: 'Dias úteis até hoje com menos horas que a jornada.'
    },
    missing_description: {
        title: 'Lançamentos sem descrição',
        hint: 'Todo lançamento precisa dizer o que foi feito.'
    },
    duplicate: {
        title: 'Possíveis duplicatas',
        hint: 'Mesma tarefa, mesmo dia, mesmo tempo e mesma descrição.'
    },
    generic_description: {
        title: 'Descrições genéricas',
        hint: 'Descrição igual ao nome da tarefa ou na sua lista de descrições genéricas.'
    },
    non_working_day: {
        title: 'Lançamentos em dia não útil',
        hint: 'Fins de semana, feriados, pontes e férias.'
    },
    over_daily_limit: {
        title: 'Dias acima do limite',
        hint: 'Dias com mais horas que o limite diário configurado.'
    },
    no_task: {
        title: 'Lançamentos sem tarefa',
        hint: 'Lançados só no projeto, sem tarefa associada.'
    }
};

export const typeInfo = (type: string): TypeInfo =>
    TIPOS[type as AuditIssueType] ?? {title: type, hint: ''};

export interface IssueGroup {
    type: string;
    severity: string;
    issues: AuditIssue[];
}

// groupIssues agrupa por tipo mantendo a ordem do backend (que já ordena os
// tipos por importância e, dentro deles, por dia). Um grupo com algum erro é
// tratado como erro.
export const groupIssues = (issues: readonly AuditIssue[]): IssueGroup[] => {
    const grupos: IssueGroup[] = [];
    const porTipo = new Map<string, IssueGroup>();
    for (const issue of issues) {
        let grupo = porTipo.get(issue.type);
        if (!grupo) {
            grupo = {type: issue.type, severity: issue.severity, issues: []};
            porTipo.set(issue.type, grupo);
            grupos.push(grupo);
        }
        grupo.issues.push(issue);
        if (issue.severity === 'error') grupo.severity = 'error';
    }
    return grupos;
};

// yearMonth separa 'YYYY-MM' em números; null quando o valor é inválido.
export const yearMonth = (month: string): {year: number; month: number} | null => {
    const m = /^(\d{4})-(\d{2})$/.exec(month);
    if (!m) return null;
    const ano = Number(m[1]);
    const mes = Number(m[2]);
    if (mes < 1 || mes > 12) return null;
    return {year: ano, month: mes};
};

// shiftMonth soma n meses a 'YYYY-MM'.
export const shiftMonth = (month: string, n: number): string => {
    const ym = yearMonth(month);
    if (!ym) return month;
    const total = ym.year * 12 + (ym.month - 1) + n;
    const ano = Math.floor(total / 12);
    const mes = (total % 12) + 1;
    return `${ano}-${String(mes).padStart(2, '0')}`;
};
