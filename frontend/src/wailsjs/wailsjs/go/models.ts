export namespace api {
	
	export class Config {
	    authToken: string;
	    userId: number;
	    apiHost: string;
	    minutosPorDia: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.authToken = source["authToken"];
	        this.userId = source["userId"];
	        this.apiHost = source["apiHost"];
	        this.minutosPorDia = source["minutosPorDia"];
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
	    }
	}
	export class LoggedTimeResponse {
	    STATUS: string;
	    // Go type: struct { Billable [][3]string "json:\"billable\""; Firstname string "json:\"firstname\""; Lastname string "json:\"lastname\""; Nonbillable [][3]string "json:\"nonbillable\""; ID string "json:\"id\""; Endepoch string "json:\"endepoch\""; Startepoch string "json:\"startepoch\"" }
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
	    token: string;
	    userId: number;
	    instanceId: string;
	
	    static createFrom(source: any = {}) {
	        return new LoginResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.token = source["token"];
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
	export class Task {
	    taskId: number;
	    taskName: string;
	    projectId: number;
	    projectName: string;
	    entries: TimeEntry[];
	
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
	export class  {
	    id: number;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new (source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	    }
	}
	export class  {
	    id: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new (source);
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
	    tags?: [];
	    assignees?: [];
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
	        this.tags = this.convertValues(source["tags"], );
	        this.assignees = this.convertValues(source["assignees"], );
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
	    }
	}
	export class TimeLogResult {
	    success: boolean;
	    message: string;
	    date: string;
	    taskId: number;
	
	    static createFrom(source: any = {}) {
	        return new TimeLogResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.date = source["date"];
	        this.taskId = source["taskId"];
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

export namespace config {
	
	export class AppSettings {
	    darkMode: boolean;
	    autoUpdate: boolean;
	    startMinimized: boolean;
	    language: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.darkMode = source["darkMode"];
	        this.autoUpdate = source["autoUpdate"];
	        this.startMinimized = source["startMinimized"];
	        this.language = source["language"];
	    }
	}

}

