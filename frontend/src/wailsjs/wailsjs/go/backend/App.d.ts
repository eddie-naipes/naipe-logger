// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT
import {api} from '../models';
import {config} from '../models';

export function ApplyTemplate(arg1:string):Promise<void>;

export function CalculateTotalMinutes(arg1:Array<api.Task>):Promise<number>;

export function ClearSavedTasks():Promise<void>;

export function CreateDistributionPlan(arg1:Array<string>,arg2:Array<api.Task>):Promise<Array<api.WorkDay>>;

export function CreateDistributionPlanFromLoggedTime(arg1:number,arg2:number,arg3:Array<api.Task>):Promise<Array<api.WorkDay>>;

export function DeleteMultipleTimeEntries(arg1:Array<number>):Promise<Array<api.DeleteTimeEntryResult>>;

export function DeleteTemplate(arg1:string):Promise<void>;

export function DeleteTimeEntry(arg1:number):Promise<void>;

export function DeleteTimeEntryV2(arg1:number):Promise<void>;

export function DownloadCurrentMonthReport():Promise<string>;

export function DownloadTimeReport(arg1:string,arg2:string):Promise<string>;

export function GetAllNonWorkingDays(arg1:number,arg2:number):Promise<Array<Record<string, any>>>;

export function GetAllTimeEntriesForDay(arg1:string):Promise<Array<api.TimeEntryReport>>;

export function GetAppSettings():Promise<config.AppSettings>;

export function GetBrazilianHolidays(arg1:number):Promise<Record<string, api.Holiday>>;

export function GetConfig():Promise<api.Config>;

export function GetCurrentUserId():Promise<number>;

export function GetCurrentUserIdWithConfig(arg1:api.Config):Promise<number>;

export function GetDashboardStats():Promise<Record<string, any>>;

export function GetDeletedTimeEntries(arg1:string,arg2:string):Promise<Array<api.TimeEntryReport>>;

export function GetEntriesFromLoggedTime(arg1:number,arg2:number):Promise<Array<Record<string, any>>>;

export function GetHolidaysForMonth(arg1:number,arg2:number):Promise<Array<api.Holiday>>;

export function GetLoggedTimeFromCalendarAPI(arg1:number,arg2:number):Promise<api.LoggedTimeResponse>;

export function GetProjects():Promise<Array<api.Project>>;

export function GetRecentActivities():Promise<Array<Record<string, any>>>;

export function GetSavedTasks():Promise<Array<api.Task>>;

export function GetTaskDetails(arg1:number):Promise<api.TeamworkTask>;

export function GetTasks():Promise<Array<api.TeamworkTask>>;

export function GetTasksByProject(arg1:number):Promise<Array<api.TeamworkTask>>;

export function GetTasksWithUpcomingDeadlines():Promise<Array<Record<string, any>>>;

export function GetTemplate(arg1:string):Promise<api.Template|boolean>;

export function GetTemplates():Promise<Record<string, api.Template>>;

export function GetTimeEntriesForPeriod(arg1:string,arg2:string):Promise<Array<api.TimeEntryReport>>;

export function GetTimeEntriesForPeriodV2(arg1:string,arg2:string,arg3:boolean):Promise<Array<api.TimeEntryReport>>;

export function GetTimeEntriesWithDetails(arg1:string,arg2:string):Promise<Array<api.TimeEntryReport>>;

export function GetTimeTotalsForPeriod(arg1:string,arg2:string):Promise<api.TimeTotal>;

export function GetUserProfile():Promise<Record<string, any>>;

export function GetWorkingDays(arg1:string,arg2:string):Promise<Array<string>>;

export function IsWorkDay(arg1:string):Promise<boolean>;

export function LogMultipleTimes(arg1:Array<api.WorkDay>):Promise<Array<api.TimeLogResult>>;

export function LogTime(arg1:number,arg2:api.TimeEntry):Promise<api.TimeLogResult>;

export function LoginWithCredentials(arg1:string,arg2:string,arg3:string):Promise<api.LoginResponse>;

export function OpenDirectoryPath(arg1:string):Promise<void>;

export function RemoveTask(arg1:number):Promise<void>;

export function SaveAppSettings(arg1:config.AppSettings):Promise<void>;

export function SaveConfig(arg1:api.Config):Promise<void>;

export function SaveTask(arg1:api.Task):Promise<void>;

export function SaveTemplate(arg1:api.Template):Promise<void>;

export function TestConnection(arg1:api.Config):Promise<Array<any>>;

export function UpdateTimeEntry(arg1:number,arg2:api.TimeEntry):Promise<api.TimeLogResult>;
