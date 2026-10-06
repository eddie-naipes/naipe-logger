export namespace agenda {
	
	export class ImportedRecord {
	    key: string;
	    date: string;
	    taskId: number;
	    entryId: number;
	    importedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportedRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.date = source["date"];
	        this.taskId = source["taskId"];
	        this.entryId = source["entryId"];
	        this.importedAt = source["importedAt"];
	    }
	}
	export class TaskRef {
	    taskId: number;
	    taskName: string;
	    projectId: number;
	    projectName: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.taskName = source["taskName"];
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	    }
	}
	export class PlanItem {
	    key: string;
	    source: string;
	    title: string;
	    date: string;
	    startTime: string;
	    endTime: string;
	    minutes: number;
	    status: string;
	    reason: string;
	    ruleIndex: number;
	    task: TaskRef;
	    description: string;
	    billable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PlanItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.source = source["source"];
	        this.title = source["title"];
	        this.date = source["date"];
	        this.startTime = source["startTime"];
	        this.endTime = source["endTime"];
	        this.minutes = source["minutes"];
	        this.status = source["status"];
	        this.reason = source["reason"];
	        this.ruleIndex = source["ruleIndex"];
	        this.task = this.convertValues(source["task"], TaskRef);
	        this.description = source["description"];
	        this.billable = source["billable"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace api {
	
	export class TaskConflict {
	    taskId: number;
	    taskName: string;
	    existingMinutes: number;
	    plannedMinutes: number;
	
	    static createFrom(source: any = {}) {
	        return new TaskConflict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.taskName = source["taskName"];
	        this.existingMinutes = source["existingMinutes"];
	        this.plannedMinutes = source["plannedMinutes"];
	    }
	}
	export class DayConflict {
	    date: string;
	    existingMinutes: number;
	    existingEntries: number;
	    plannedMinutes: number;
	    sameTask: TaskConflict[];
	
	    static createFrom(source: any = {}) {
	        return new DayConflict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.existingMinutes = source["existingMinutes"];
	        this.existingEntries = source["existingEntries"];
	        this.plannedMinutes = source["plannedMinutes"];
	        this.sameTask = this.convertValues(source["sameTask"], TaskConflict);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DeleteTimeEntryResult {
	    entryId: number;
	    success: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteTimeEntryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entryId = source["entryId"];
	        this.success = source["success"];
	        this.message = source["message"];
	    }
	}
	export class TimeEntry {
	    minutes: number;
	    userId: number;
	    time: string;
	    description: string;
	    isBillable: boolean;
	    date?: string;
	
	    static createFrom(source: any = {}) {
	        return new TimeEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.minutes = source["minutes"];
	        this.userId = source["userId"];
	        this.time = source["time"];
	        this.description = source["description"];
	        this.isBillable = source["isBillable"];
	        this.date = source["date"];
	    }
	}
	export class EntryTask {
	    taskId: number;
	    entry: TimeEntry;
	
	    static createFrom(source: any = {}) {
	        return new EntryTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.entry = this.convertValues(source["entry"], TimeEntry);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Holiday {
	    date: string;
	    name: string;
	    description?: string;
	    type: string;
	    isOptional: boolean;
	    source?: string;
	
	    static createFrom(source: any = {}) {
	        return new Holiday(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.type = source["type"];
	        this.isOptional = source["isOptional"];
	        this.source = source["source"];
	    }
	}
	export class HolidayCacheDetail {
	    holidays_count: number;
	    // Go type: time
	    cached_at: any;
	    // Go type: time
	    expires_at: any;
	    sources: string[];
	    is_expired: boolean;
	
	    static createFrom(source: any = {}) {
	        return new HolidayCacheDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.holidays_count = source["holidays_count"];
	        this.cached_at = this.convertValues(source["cached_at"], null);
	        this.expires_at = this.convertValues(source["expires_at"], null);
	        this.sources = source["sources"];
	        this.is_expired = source["is_expired"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HolidayCacheStats {
	    cached_years: number;
	    years: number[];
	    cache_details: Record<number, HolidayCacheDetail>;
	
	    static createFrom(source: any = {}) {
	        return new HolidayCacheStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cached_years = source["cached_years"];
	        this.years = source["years"];
	        this.cache_details = this.convertValues(source["cache_details"], HolidayCacheDetail, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LoggedTimeResponse {
	    STATUS: string;
	    // Go type: struct { Billable [][3]api
	    user: any;
	
	    static createFrom(source: any = {}) {
	        return new LoggedTimeResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.STATUS = source["STATUS"];
	        this.user = this.convertValues(source["user"], Object);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LoginResponse {
	    success: boolean;
	    message: string;
	    userId: number;
	    instanceId: string;
	
	    static createFrom(source: any = {}) {
	        return new LoginResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.userId = source["userId"];
	        this.instanceId = source["instanceId"];
	    }
	}
	export class Project {
	    id: number;
	    name: string;
	    description?: string;
	    status?: string;
	    // Go type: struct { ID int "json:\"id\""; Name string "json:\"name\"" }
	    company?: any;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.status = source["status"];
	        this.company = this.convertValues(source["company"], Object);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PublicConfig {
	    configured: boolean;
	    apiHost: string;
	    userId: number;
	    minutosPorDia: number;
	
	    static createFrom(source: any = {}) {
	        return new PublicConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.configured = source["configured"];
	        this.apiHost = source["apiHost"];
	        this.userId = source["userId"];
	        this.minutosPorDia = source["minutosPorDia"];
	    }
	}
	export class Task {
	    taskId: number;
	    taskName: string;
	    projectId: number;
	    projectName: string;
	    entries: TimeEntry[];
	    workingDays?: number[];
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.taskName = source["taskName"];
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	        this.entries = this.convertValues(source["entries"], TimeEntry);
	        this.workingDays = source["workingDays"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TaskAssignee {
	    id: number;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskAssignee(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	    }
	}
	
	export class TaskTag {
	    id: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskTag(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class TeamworkTask {
	    id: number;
	    content: string;
	    name?: string;
	    description?: string;
	    projectId: number;
	    projectName: string;
	    status?: string;
	    priority?: string;
	    createdAt?: string;
	    startDate?: string;
	    dueDate?: string;
	    tasklistId?: number;
	    tasklistName?: string;
	    tags?: TaskTag[];
	    assignees?: TaskAssignee[];
	    loggedMinutes?: number;
	
	    static createFrom(source: any = {}) {
	        return new TeamworkTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.content = source["content"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	        this.status = source["status"];
	        this.priority = source["priority"];
	        this.createdAt = source["createdAt"];
	        this.startDate = source["startDate"];
	        this.dueDate = source["dueDate"];
	        this.tasklistId = source["tasklistId"];
	        this.tasklistName = source["tasklistName"];
	        this.tags = this.convertValues(source["tags"], TaskTag);
	        this.assignees = this.convertValues(source["assignees"], TaskAssignee);
	        this.loggedMinutes = source["loggedMinutes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Template {
	    name: string;
	    tasks: Task[];
	    totalMin: number;
	
	    static createFrom(source: any = {}) {
	        return new Template(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.tasks = this.convertValues(source["tasks"], Task);
	        this.totalMin = source["totalMin"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class TimeEntryReport {
	    id: number;
	    projectId: number;
	    projectName: string;
	    taskId: number;
	    taskName: string;
	    tasklistId: number;
	    tasklistName: string;
	    userId: number;
	    userFirstName: string;
	    userLastName: string;
	    date: string;
	    hours: number;
	    minutes: number;
	    description: string;
	    isBillable: boolean;
	    isBilled: boolean;
	    startTime: string;
	    endTime: string;
	    status?: string;
	    createdAt?: string;
	    updatedAt?: string;
	    deletedAt?: string;
	    deletedBy?: string;
	
	    static createFrom(source: any = {}) {
	        return new TimeEntryReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	        this.taskId = source["taskId"];
	        this.taskName = source["taskName"];
	        this.tasklistId = source["tasklistId"];
	        this.tasklistName = source["tasklistName"];
	        this.userId = source["userId"];
	        this.userFirstName = source["userFirstName"];
	        this.userLastName = source["userLastName"];
	        this.date = source["date"];
	        this.hours = source["hours"];
	        this.minutes = source["minutes"];
	        this.description = source["description"];
	        this.isBillable = source["isBillable"];
	        this.isBilled = source["isBilled"];
	        this.startTime = source["startTime"];
	        this.endTime = source["endTime"];
	        this.status = source["status"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.deletedAt = source["deletedAt"];
	        this.deletedBy = source["deletedBy"];
	    }
	}
	export class TimeLogResult {
	    success: boolean;
	    message: string;
	    date: string;
	    taskId: number;
	    entryId: number;
	
	    static createFrom(source: any = {}) {
	        return new TimeLogResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.date = source["date"];
	        this.taskId = source["taskId"];
	        this.entryId = source["entryId"];
	    }
	}
	export class TimeTotal {
	    // Go type: struct { TotalCost float64 "json:\"totalCost\""; TotalCostBillable float64 "json:\"totalCostBillable\""; TotalCostBilled float64 "json:\"totalCostBilled\"" }
	    financialTotals: any;
	    // Go type: struct { EstimatedMinutes int "json:\"estimatedMinutes\""; Minutes int "json:\"minutes\""; MinutesBillable int "json:\"minutesBillable\"" }
	    subTasks: any;
	    // Go type: struct { EstimatedMinutes int "json:\"estimatedMinutes\""; EstimatedMinutesActive int "json:\"estimatedMinutesActive\""; EstimatedMinutesCompleted int "json:\"estimatedMinutesCompleted\""; EstimatedMinutesFiltered int "json:\"estimatedMinutesFiltered\""; EstimatedMinutesWithLoggedTime int "json:\"estimatedMinutesWithLoggedTime\""; Minutes int "json:\"minutes\""; MinutesBillable int "json:\"minutesBillable\""; MinutesBilled int "json:\"minutesBilled\""; MinutesNonBillable int "json:\"minutesNonBillable\""; MinutesNonBilled int "json:\"minutesNonBilled\"" }
	    "time-totals": any;
	
	    static createFrom(source: any = {}) {
	        return new TimeTotal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.financialTotals = this.convertValues(source["financialTotals"], Object);
	        this.subTasks = this.convertValues(source["subTasks"], Object);
	        this["time-totals"] = this.convertValues(source["time-totals"], Object);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkDay {
	    date: string;
	    entries: EntryTask[];
	    totalMin: number;
	
	    static createFrom(source: any = {}) {
	        return new WorkDay(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.entries = this.convertValues(source["entries"], EntryTask);
	        this.totalMin = source["totalMin"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace audit {
	
	export class Issue {
	    key: string;
	    type: string;
	    severity: string;
	    date: string;
	    entryIds: number[];
	    entries: api.TimeEntryReport[];
	    message: string;
	    action: string;
	    deleteEntryIds?: number[];
	    keepEntryId?: number;
	    minutes?: number;
	    missingMinutes?: number;
	    reason?: string;
	    ignored: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Issue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.type = source["type"];
	        this.severity = source["severity"];
	        this.date = source["date"];
	        this.entryIds = source["entryIds"];
	        this.entries = this.convertValues(source["entries"], api.TimeEntryReport);
	        this.message = source["message"];
	        this.action = source["action"];
	        this.deleteEntryIds = source["deleteEntryIds"];
	        this.keepEntryId = source["keepEntryId"];
	        this.minutes = source["minutes"];
	        this.missingMinutes = source["missingMinutes"];
	        this.reason = source["reason"];
	        this.ignored = source["ignored"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Summary {
	    year: number;
	    month: number;
	    loggedMinutes: number;
	    expectedMinutes: number;
	    expectedToDateMinutes: number;
	    loggedToDateMinutes: number;
	    workingDays: number;
	    workingDaysToDate: number;
	    minutesPerDay: number;
	    dailyLimitMinutes: number;
	    entryCount: number;
	    errorCount: number;
	    warningCount: number;
	    ignoredCount: number;
	    ready: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = source["year"];
	        this.month = source["month"];
	        this.loggedMinutes = source["loggedMinutes"];
	        this.expectedMinutes = source["expectedMinutes"];
	        this.expectedToDateMinutes = source["expectedToDateMinutes"];
	        this.loggedToDateMinutes = source["loggedToDateMinutes"];
	        this.workingDays = source["workingDays"];
	        this.workingDaysToDate = source["workingDaysToDate"];
	        this.minutesPerDay = source["minutesPerDay"];
	        this.dailyLimitMinutes = source["dailyLimitMinutes"];
	        this.entryCount = source["entryCount"];
	        this.errorCount = source["errorCount"];
	        this.warningCount = source["warningCount"];
	        this.ignoredCount = source["ignoredCount"];
	        this.ready = source["ready"];
	    }
	}
	export class TypeCount {
	    type: string;
	    severity: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new TypeCount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.severity = source["severity"];
	        this.count = source["count"];
	    }
	}
	export class Result {
	    issues: Issue[];
	    counts: TypeCount[];
	    summary: Summary;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.issues = this.convertValues(source["issues"], Issue);
	        this.counts = this.convertValues(source["counts"], TypeCount);
	        this.summary = this.convertValues(source["summary"], Summary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace autostart {
	
	export class Status {
	    enabled: boolean;
	    supported: boolean;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.supported = source["supported"];
	        this.reason = source["reason"];
	    }
	}

}

export namespace backend {
	
	export class AgendaFileInfo {
	    name: string;
	    events: number;
	
	    static createFrom(source: any = {}) {
	        return new AgendaFileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.events = source["events"];
	    }
	}
	export class AgendaPlan {
	    items: agenda.PlanItem[];
	    warnings: string[];
	    sources: string[];
	
	    static createFrom(source: any = {}) {
	        return new AgendaPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], agenda.PlanItem);
	        this.warnings = source["warnings"];
	        this.sources = source["sources"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FillGapsRequest {
	    start: string;
	    end: string;
	    templateName: string;
	    taskIds: number[];
	    includeFuture: boolean;
	    granularity: number;
	
	    static createFrom(source: any = {}) {
	        return new FillGapsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = source["start"];
	        this.end = source["end"];
	        this.templateName = source["templateName"];
	        this.taskIds = source["taskIds"];
	        this.includeFuture = source["includeFuture"];
	        this.granularity = source["granularity"];
	    }
	}
	export class FillGapsResult {
	    plan: api.WorkDay[];
	    days: planning.DaySummary[];
	    minutesPerDay: number;
	
	    static createFrom(source: any = {}) {
	        return new FillGapsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plan = this.convertValues(source["plan"], api.WorkDay);
	        this.days = this.convertValues(source["days"], planning.DaySummary);
	        this.minutesPerDay = source["minutesPerDay"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NotificationStatus {
	    available: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new NotificationStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.error = source["error"];
	    }
	}
	export class UserProfile {
	    id: number;
	    firstName: string;
	    lastName: string;
	    email: string;
	    avatarURL: string;
	    fullName: string;
	
	    static createFrom(source: any = {}) {
	        return new UserProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.firstName = source["firstName"];
	        this.lastName = source["lastName"];
	        this.email = source["email"];
	        this.avatarURL = source["avatarURL"];
	        this.fullName = source["fullName"];
	    }
	}

}

export namespace config {
	
	export class Absence {
	    start: string;
	    end: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new Absence(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = source["start"];
	        this.end = source["end"];
	        this.description = source["description"];
	    }
	}
	export class AgendaCalendar {
	    id: string;
	    name: string;
	    maskedUrl: string;
	    addedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new AgendaCalendar(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.maskedUrl = source["maskedUrl"];
	        this.addedAt = source["addedAt"];
	    }
	}
	export class AgendaTask {
	    taskId: number;
	    taskName: string;
	    projectId: number;
	    projectName: string;
	
	    static createFrom(source: any = {}) {
	        return new AgendaTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.taskName = source["taskName"];
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	    }
	}
	export class AgendaRule {
	    match: string;
	    isRegex: boolean;
	    task: AgendaTask;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new AgendaRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.match = source["match"];
	        this.isRegex = source["isRegex"];
	        this.task = this.convertValues(source["task"], AgendaTask);
	        this.description = source["description"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AgendaSettings {
	    calendars: AgendaCalendar[];
	    rules: AgendaRule[];
	    defaultTask: AgendaTask;
	    ignoreWords: string[];
	    minMinutes: number;
	    rounding: string;
	    userEmail: string;
	    billable: boolean;
	    includeTransparent: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AgendaSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.calendars = this.convertValues(source["calendars"], AgendaCalendar);
	        this.rules = this.convertValues(source["rules"], AgendaRule);
	        this.defaultTask = this.convertValues(source["defaultTask"], AgendaTask);
	        this.ignoreWords = source["ignoreWords"];
	        this.minMinutes = source["minMinutes"];
	        this.rounding = source["rounding"];
	        this.userEmail = source["userEmail"];
	        this.billable = source["billable"];
	        this.includeTransparent = source["includeTransparent"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class AppSettings {
	    darkMode: boolean;
	    autoUpdate: boolean;
	    startMinimized: boolean;
	    language: string;
	    checkUpdatesOnStartup: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.darkMode = source["darkMode"];
	        this.autoUpdate = source["autoUpdate"];
	        this.startMinimized = source["startMinimized"];
	        this.language = source["language"];
	        this.checkUpdatesOnStartup = source["checkUpdatesOnStartup"];
	    }
	}
	export class AuditSettings {
	    dailyLimitMinutes: number;
	    genericDescriptions: string[];
	    ignoredIssues: string[];
	
	    static createFrom(source: any = {}) {
	        return new AuditSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dailyLimitMinutes = source["dailyLimitMinutes"];
	        this.genericDescriptions = source["genericDescriptions"];
	        this.ignoredIssues = source["ignoredIssues"];
	    }
	}
	export class CustomHoliday {
	    date: string;
	    recurring: boolean;
	    name: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomHoliday(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.recurring = source["recurring"];
	        this.name = source["name"];
	        this.type = source["type"];
	    }
	}
	export class CalendarSettings {
	    uf: string;
	    disabledStateHolidays: string[];
	    customHolidays: CustomHoliday[];
	    absences: Absence[];
	
	    static createFrom(source: any = {}) {
	        return new CalendarSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uf = source["uf"];
	        this.disabledStateHolidays = source["disabledStateHolidays"];
	        this.customHolidays = this.convertValues(source["customHolidays"], CustomHoliday);
	        this.absences = this.convertValues(source["absences"], Absence);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class GitIntegration {
	    enabled: boolean;
	    repositories: string[];
	    authorEmail: string;
	
	    static createFrom(source: any = {}) {
	        return new GitIntegration(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.repositories = source["repositories"];
	        this.authorEmail = source["authorEmail"];
	    }
	}
	export class ReminderSettings {
	    enabled: boolean;
	    dailyTime: string;
	    workDaysOnly: boolean;
	    monthEndEnabled: boolean;
	    monthEndDays: number;
	
	    static createFrom(source: any = {}) {
	        return new ReminderSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.dailyTime = source["dailyTime"];
	        this.workDaysOnly = source["workDaysOnly"];
	        this.monthEndEnabled = source["monthEndEnabled"];
	        this.monthEndDays = source["monthEndDays"];
	    }
	}
	export class TimerSettings {
	    rounding: string;
	    longRunningHours: number;
	
	    static createFrom(source: any = {}) {
	        return new TimerSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rounding = source["rounding"];
	        this.longRunningHours = source["longRunningHours"];
	    }
	}

}

export namespace gitlog {
	
	export class Commit {
	    repo: string;
	    hash: string;
	    time: string;
	    subject: string;
	
	    static createFrom(source: any = {}) {
	        return new Commit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repo = source["repo"];
	        this.hash = source["hash"];
	        this.time = source["time"];
	        this.subject = source["subject"];
	    }
	}
	export class Result {
	    suggestion: string;
	    commits: Commit[];
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.suggestion = source["suggestion"];
	        this.commits = this.convertValues(source["commits"], Commit);
	        this.warnings = source["warnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace holidays {
	
	export class DayInfo {
	    date: string;
	    type: string;
	    name: string;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new DayInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.description = source["description"];
	    }
	}
	export class StateHoliday {
	    monthDay: string;
	    name: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new StateHoliday(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.monthDay = source["monthDay"];
	        this.name = source["name"];
	        this.source = source["source"];
	    }
	}
	export class State {
	    uf: string;
	    name: string;
	    holidays: StateHoliday[];
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uf = source["uf"];
	        this.name = source["name"];
	        this.holidays = this.convertValues(source["holidays"], StateHoliday);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace legacy {
	
	export class Install {
	    found: boolean;
	    displayName: string;
	    installLocation: string;
	    uninstallString: string;
	
	    static createFrom(source: any = {}) {
	        return new Install(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.found = source["found"];
	        this.displayName = source["displayName"];
	        this.installLocation = source["installLocation"];
	        this.uninstallString = source["uninstallString"];
	    }
	}

}

export namespace planning {
	
	export class CopyPlan {
	    plan: api.WorkDay[];
	    skippedDays: string[];
	    skippedEntries: number;
	
	    static createFrom(source: any = {}) {
	        return new CopyPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plan = this.convertValues(source["plan"], api.WorkDay);
	        this.skippedDays = source["skippedDays"];
	        this.skippedEntries = source["skippedEntries"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DaySummary {
	    date: string;
	    logged: number;
	    missing: number;
	    toLog: number;
	    future: boolean;
	    skipped: boolean;
	    skipCause?: string;
	
	    static createFrom(source: any = {}) {
	        return new DaySummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.logged = source["logged"];
	        this.missing = source["missing"];
	        this.toLog = source["toLog"];
	        this.future = source["future"];
	        this.skipped = source["skipped"];
	        this.skipCause = source["skipCause"];
	    }
	}
	export class WeekCell {
	    date: string;
	    minutes: number;
	    entries: api.TimeEntryReport[];
	
	    static createFrom(source: any = {}) {
	        return new WeekCell(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.minutes = source["minutes"];
	        this.entries = this.convertValues(source["entries"], api.TimeEntryReport);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WeekRow {
	    taskId: number;
	    taskName: string;
	    projectName: string;
	    saved: boolean;
	    cells: WeekCell[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new WeekRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.taskName = source["taskName"];
	        this.projectName = source["projectName"];
	        this.saved = source["saved"];
	        this.cells = this.convertValues(source["cells"], WeekCell);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WeekGrid {
	    weekStart: string;
	    days: string[];
	    rows: WeekRow[];
	    dayTotals: number[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new WeekGrid(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.weekStart = source["weekStart"];
	        this.days = source["days"];
	        this.rows = this.convertValues(source["rows"], WeekRow);
	        this.dayTotals = source["dayTotals"];
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace reports {
	
	export class DayTotal {
	    date: string;
	    minutes: number;
	    billableMinutes: number;
	    isWorkingDay: boolean;
	    expectedMinutes: number;
	
	    static createFrom(source: any = {}) {
	        return new DayTotal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.minutes = source["minutes"];
	        this.billableMinutes = source["billableMinutes"];
	        this.isWorkingDay = source["isWorkingDay"];
	        this.expectedMinutes = source["expectedMinutes"];
	    }
	}
	export class ProjectTotal {
	    projectId: number;
	    projectName: string;
	    minutes: number;
	    billableMinutes: number;
	    entryCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ProjectTotal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	        this.minutes = source["minutes"];
	        this.billableMinutes = source["billableMinutes"];
	        this.entryCount = source["entryCount"];
	    }
	}
	export class WeekTotal {
	    weekStart: string;
	    weekEnd: string;
	    minutes: number;
	    billableMinutes: number;
	    workingDays: number;
	    expectedMinutes: number;
	
	    static createFrom(source: any = {}) {
	        return new WeekTotal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.weekStart = source["weekStart"];
	        this.weekEnd = source["weekEnd"];
	        this.minutes = source["minutes"];
	        this.billableMinutes = source["billableMinutes"];
	        this.workingDays = source["workingDays"];
	        this.expectedMinutes = source["expectedMinutes"];
	    }
	}
	export class TaskTotal {
	    taskId: number;
	    taskName: string;
	    projectId: number;
	    projectName: string;
	    minutes: number;
	    billableMinutes: number;
	    entryCount: number;
	
	    static createFrom(source: any = {}) {
	        return new TaskTotal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.taskName = source["taskName"];
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	        this.minutes = source["minutes"];
	        this.billableMinutes = source["billableMinutes"];
	        this.entryCount = source["entryCount"];
	    }
	}
	export class Report {
	    startDate: string;
	    endDate: string;
	    totalMinutes: number;
	    billableMinutes: number;
	    nonBillableMinutes: number;
	    entryCount: number;
	    workingDays: number;
	    minutesPerDay: number;
	    expectedMinutes: number;
	    balanceMinutes: number;
	    daysWithEntries: number;
	    workingDaysWithoutEntries: number;
	    byProject: ProjectTotal[];
	    byTask: TaskTotal[];
	    byDay: DayTotal[];
	    byWeek: WeekTotal[];
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.startDate = source["startDate"];
	        this.endDate = source["endDate"];
	        this.totalMinutes = source["totalMinutes"];
	        this.billableMinutes = source["billableMinutes"];
	        this.nonBillableMinutes = source["nonBillableMinutes"];
	        this.entryCount = source["entryCount"];
	        this.workingDays = source["workingDays"];
	        this.minutesPerDay = source["minutesPerDay"];
	        this.expectedMinutes = source["expectedMinutes"];
	        this.balanceMinutes = source["balanceMinutes"];
	        this.daysWithEntries = source["daysWithEntries"];
	        this.workingDaysWithoutEntries = source["workingDaysWithoutEntries"];
	        this.byProject = this.convertValues(source["byProject"], ProjectTotal);
	        this.byTask = this.convertValues(source["byTask"], TaskTotal);
	        this.byDay = this.convertValues(source["byDay"], DayTotal);
	        this.byWeek = this.convertValues(source["byWeek"], WeekTotal);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace timer {
	
	export class Entry {
	    date: string;
	    time: string;
	    minutes: number;
	    seconds: number;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.time = source["time"];
	        this.minutes = source["minutes"];
	        this.seconds = source["seconds"];
	    }
	}
	export class Segment {
	    start: string;
	    end: string;
	
	    static createFrom(source: any = {}) {
	        return new Segment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = source["start"];
	        this.end = source["end"];
	    }
	}
	export class State {
	    running: boolean;
	    paused: boolean;
	    taskId: number;
	    taskName: string;
	    projectName: string;
	    description: string;
	    billable: boolean;
	    startedAt: string;
	    firstStartedAt: string;
	    accumulatedSeconds: number;
	    elapsedSeconds: number;
	    segments: Segment[];
	    date: string;
	    loggedDates: string[];
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.paused = source["paused"];
	        this.taskId = source["taskId"];
	        this.taskName = source["taskName"];
	        this.projectName = source["projectName"];
	        this.description = source["description"];
	        this.billable = source["billable"];
	        this.startedAt = source["startedAt"];
	        this.firstStartedAt = source["firstStartedAt"];
	        this.accumulatedSeconds = source["accumulatedSeconds"];
	        this.elapsedSeconds = source["elapsedSeconds"];
	        this.segments = this.convertValues(source["segments"], Segment);
	        this.date = source["date"];
	        this.loggedDates = source["loggedDates"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StopResult {
	    logged: boolean;
	    results: api.TimeLogResult[];
	    state: State;
	
	    static createFrom(source: any = {}) {
	        return new StopResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.logged = source["logged"];
	        this.results = this.convertValues(source["results"], api.TimeLogResult);
	        this.state = this.convertValues(source["state"], State);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TaskRef {
	    taskId: number;
	    taskName: string;
	    projectName: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.taskName = source["taskName"];
	        this.projectName = source["projectName"];
	    }
	}

}

export namespace update {
	
	export class Info {
	    available: boolean;
	    currentVersion: string;
	    latestVersion: string;
	    releaseNotes: string;
	    releaseUrl: string;
	    publishedAt: string;
	    canInstall: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.releaseNotes = source["releaseNotes"];
	        this.releaseUrl = source["releaseUrl"];
	        this.publishedAt = source["publishedAt"];
	        this.canInstall = source["canInstall"];
	    }
	}

}

