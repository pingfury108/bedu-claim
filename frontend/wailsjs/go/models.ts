export namespace main {
	
	export class AutoClaimConfig {
	    ServerBaseURL: string;
	    Cookie: string;
	    TaskType: string;
	    ClaimLimit: number;
	    Interval: number;
	    MaxPages: number;
	    ConcurrentClaims: number;
	    StepID: number;
	    SubjectID: number;
	    ClueTypeID: number;
	    IncludeKeywords: string[];
	    ExcludeKeywords: string[];
	    StartTime: string;
	    EndTime: string;
	    authType: string;
	    authUsername: string;
	
	    static createFrom(source: any = {}) {
	        return new AutoClaimConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ServerBaseURL = source["ServerBaseURL"];
	        this.Cookie = source["Cookie"];
	        this.TaskType = source["TaskType"];
	        this.ClaimLimit = source["ClaimLimit"];
	        this.Interval = source["Interval"];
	        this.MaxPages = source["MaxPages"];
	        this.ConcurrentClaims = source["ConcurrentClaims"];
	        this.StepID = source["StepID"];
	        this.SubjectID = source["SubjectID"];
	        this.ClueTypeID = source["ClueTypeID"];
	        this.IncludeKeywords = source["IncludeKeywords"];
	        this.ExcludeKeywords = source["ExcludeKeywords"];
	        this.StartTime = source["StartTime"];
	        this.EndTime = source["EndTime"];
	        this.authType = source["authType"];
	        this.authUsername = source["authUsername"];
	    }
	}
	export class AutoClaimResponse {
	    success: boolean;
	    message: string;
	    taskId?: string;
	
	    static createFrom(source: any = {}) {
	        return new AutoClaimResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.taskId = source["taskId"];
	    }
	}
	export class AutoClaimStatusResponse {
	    success: boolean;
	    message: string;
	    isActive: boolean;
	    successfulClaims: number;
	    lastError: string;
	
	    static createFrom(source: any = {}) {
	        return new AutoClaimStatusResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.isActive = source["isActive"];
	        this.successfulClaims = source["successfulClaims"];
	        this.lastError = source["lastError"];
	    }
	}

}

