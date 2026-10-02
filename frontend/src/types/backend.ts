// Tipos dos dados que chegam do backend Go.
//
// Sempre que o binding tem tipo gerado pelo Wails (namespaces api/backend/config
// de wailsjs/go/models.ts), usamos o gerado via Dados<>. Os tipos escritos à mão
// aqui cobrem só o que o gerador não descreve: bindings que devolvem
// map[string]interface{} e campos que são structs anônimas no Go (o Wails os
// tipa como `any`). Os nomes de campo seguem as tags `json` dos structs em
// backend/api/*.go e backend/*.go.
import type {api, config, legacy, update} from '@wailsjs/go/models';

// As classes geradas pelo Wails têm, além dos campos, o método convertValues.
// Dados<T> fica só com os campos — é o que de fato trafega em JSON — para que o
// estado do React possa ser montado com objetos literais e spreads.
export type Dados<T> = T extends readonly (infer U)[]
    ? Dados<U>[]
    : T extends object
        ? {[K in keyof T as T[K] extends (...args: never[]) => unknown ? never : K]: Dados<T[K]>}
        : T;

// paraBinding converte dados puros no tipo de classe que a assinatura gerada do
// binding pede. É seguro: o Wails serializa o argumento com JSON.stringify, que
// ignora métodos, então o backend recebe exatamente os mesmos campos. A
// conversão fica concentrada aqui para não espalhar `as` pelo código. NoInfer
// faz T vir do parâmetro do binding, e não do argumento.
export const paraBinding = <T>(dados: Dados<NoInfer<T>>): T => dados as T;

export type TimeEntry = Dados<api.TimeEntry>;
export type Task = Dados<api.Task>;
export type EntryTask = Dados<api.EntryTask>;
export type WorkDay = Dados<api.WorkDay>;
export type TimeLogResult = Dados<api.TimeLogResult>;
export type DeleteTimeEntryResult = Dados<api.DeleteTimeEntryResult>;
export type DayConflict = Dados<api.DayConflict>;
export type TimeEntryReport = Dados<api.TimeEntryReport>;
export type TeamworkTask = Dados<api.TeamworkTask>;
export type Project = Dados<api.Project>;
export type Template = Dados<api.Template>;
export type Holiday = Dados<api.Holiday>;
export type AppSettings = Dados<config.AppSettings>;
export type UpdateInfo = Dados<update.Info>;
export type LegacyInstall = Dados<legacy.Install>;

// Payload do evento "update:progress" (update.Progress no Go). total pode ser
// 0 quando o servidor não informa o tamanho do download.
export interface UpdateProgress {
    received: number;
    total: number;
}

// GetHolidayCacheStats. O Wails tipa os campos time.Time como `any`; no JSON
// eles chegam como texto RFC 3339. Slices e mapas nil do Go viram null.
export type HolidayCacheDetail = Omit<Dados<api.HolidayCacheDetail>, 'cached_at' | 'expires_at' | 'sources'> & {
    cached_at: string;
    expires_at: string;
    sources: string[] | null;
};

export type HolidayCacheStats = Omit<Dados<api.HolidayCacheStats>, 'years' | 'cache_details'> & {
    years: number[] | null;
    cache_details: Record<number, HolidayCacheDetail> | null;
};

// --- Bindings que devolvem map[string]interface{} -------------------------

// GetDashboardStats (api.DashboardStats em backend/api/dashboard_types.go).
export interface DashboardStats {
    tarefasPendentes: number;
    projetos: number;
    horasLogadas: number;
    horasLogadasChange: number;
    diasUteisMes: number;
    diasUteisRestantes: number;
    diasUteisPassados: number;
}

// GetRecentActivities (api.RecentActivity).
export interface RecentActivity {
    id: number;
    type: string;
    description: string;
    minutes: number;
    date: string;
    projectId: number;
    projectName: string;
    taskId: number;
    taskName: string;
}

// GetTasksWithUpcomingDeadlines (api.UpcomingDeadline).
export interface UpcomingDeadline {
    id: number;
    name: string;
    dueDate: string;
    priority: string;
    projectId: number;
    projectName: string;
}

// GetAllNonWorkingDays (api.NonWorkingDay). description e isOptional só vêm
// nos feriados.
export interface NonWorkingDay {
    date: string;
    type: 'weekend' | 'holiday';
    name: string;
    description?: string;
    isOptional?: boolean;
}


// --- Campos que são structs anônimas no Go (tipados como any pelo Wails) ---

// api.LoggedTimeResponse.user: cada item de billable/nonbillable é
// [timestampUTCms, horas, minutos], tudo como texto.
export type CalendarDayTuple = [string, string, string];

export interface LoggedTimeUser {
    billable: CalendarDayTuple[] | null;
    nonbillable: CalendarDayTuple[] | null;
    firstname: string;
    lastname: string;
    id: string;
    endepoch: string;
    startepoch: string;
}

export interface LoggedTimeResponse {
    STATUS: string;
    user: LoggedTimeUser | null;
}

// api.TimeTotal["time-totals"].
export interface TimeTotalsMinutes {
    estimatedMinutes: number;
    estimatedMinutesActive: number;
    estimatedMinutesCompleted: number;
    estimatedMinutesFiltered: number;
    estimatedMinutesWithLoggedTime: number;
    minutes: number;
    minutesBillable: number;
    minutesBilled: number;
    minutesNonBillable: number;
    minutesNonBilled: number;
}

export interface TimeTotal {
    'time-totals': TimeTotalsMinutes | null;
}
