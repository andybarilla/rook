export namespace api {
	
	export class BuildStatus {
	    name: string;
	    hasBuild: boolean;
	    status: string;
	    reasons?: string[];
	
	    static createFrom(source: any = {}) {
	        return new BuildStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.hasBuild = source["hasBuild"];
	        this.status = source["status"];
	        this.reasons = source["reasons"];
	    }
	}
	export class BuildCheckResult {
	    services: BuildStatus[];
	    hasStale: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BuildCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.services = this.convertValues(source["services"], BuildStatus);
	        this.hasStale = source["hasStale"];
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
	
	export class ServiceDiff {
	    name: string;
	    image?: string;
	    build?: string;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.image = source["image"];
	        this.build = source["build"];
	        this.reason = source["reason"];
	    }
	}
	export class DiscoverDiff {
	    source: string;
	    newServices: ServiceDiff[];
	    removedServices: ServiceDiff[];
	    hasChanges: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DiscoverDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.newServices = this.convertValues(source["newServices"], ServiceDiff);
	        this.removedServices = this.convertValues(source["removedServices"], ServiceDiff);
	        this.hasChanges = source["hasChanges"];
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
	export class DiscoverResult {
	    source: string;
	    services: Record<string, workspace.Service>;
	    groups?: Record<string, Array<string>>;
	
	    static createFrom(source: any = {}) {
	        return new DiscoverResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.services = this.convertValues(source["services"], workspace.Service, true);
	        this.groups = source["groups"];
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
	export class LogLine {
	    workspace: string;
	    service: string;
	    line: string;
	    // Go type: time
	    timestamp: any;
	
	    static createFrom(source: any = {}) {
	        return new LogLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = source["workspace"];
	        this.service = source["service"];
	        this.line = source["line"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
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
	
	export class ServiceInfo {
	    name: string;
	    image?: string;
	    command?: string;
	    status: string;
	    port?: number;
	    dependsOn?: string[];
	    hasBuild: boolean;
	    buildStatus?: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.image = source["image"];
	        this.command = source["command"];
	        this.status = source["status"];
	        this.port = source["port"];
	        this.dependsOn = source["dependsOn"];
	        this.hasBuild = source["hasBuild"];
	        this.buildStatus = source["buildStatus"];
	    }
	}
	export class Settings {
	    autoRebuild: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.autoRebuild = source["autoRebuild"];
	    }
	}
	export class WorkspaceDetail {
	    name: string;
	    path: string;
	    services: ServiceInfo[];
	    profiles?: Record<string, Array<string>>;
	    groups?: Record<string, Array<string>>;
	    activeProfile?: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.services = this.convertValues(source["services"], ServiceInfo);
	        this.profiles = source["profiles"];
	        this.groups = source["groups"];
	        this.activeProfile = source["activeProfile"];
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
	export class WorkspaceInfo {
	    name: string;
	    path: string;
	    serviceCount: number;
	    runningCount: number;
	    activeProfile?: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.serviceCount = source["serviceCount"];
	        this.runningCount = source["runningCount"];
	        this.activeProfile = source["activeProfile"];
	    }
	}

}

export namespace ports {
	
	export class PortEntry {
	    workspace: string;
	    service: string;
	    port: number;
	    pinned?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PortEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = source["workspace"];
	        this.service = source["service"];
	        this.port = source["port"];
	        this.pinned = source["pinned"];
	    }
	}

}

export namespace workspace {
	
	export class Service {
	    Image: string;
	    Command: string;
	    Path: string;
	    WorkingDir: string;
	    Ports: number[];
	    PinPort: number;
	    Environment: Record<string, string>;
	    DependsOn: string[];
	    Healthcheck: any;
	    Volumes: string[];
	    EnvFile: string;
	    Build: string;
	    Dockerfile: string;
	    BuildFrom: string;
	    ForceBuild: boolean;
	    ResolvedEnvFile: string;
	
	    static createFrom(source: any = {}) {
	        return new Service(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Image = source["Image"];
	        this.Command = source["Command"];
	        this.Path = source["Path"];
	        this.WorkingDir = source["WorkingDir"];
	        this.Ports = source["Ports"];
	        this.PinPort = source["PinPort"];
	        this.Environment = source["Environment"];
	        this.DependsOn = source["DependsOn"];
	        this.Healthcheck = source["Healthcheck"];
	        this.Volumes = source["Volumes"];
	        this.EnvFile = source["EnvFile"];
	        this.Build = source["Build"];
	        this.Dockerfile = source["Dockerfile"];
	        this.BuildFrom = source["BuildFrom"];
	        this.ForceBuild = source["ForceBuild"];
	        this.ResolvedEnvFile = source["ResolvedEnvFile"];
	    }
	}
	export class Manifest {
	    Name: string;
	    Type: string;
	    Root: string;
	    Services: Record<string, Service>;
	    Groups: Record<string, Array<string>>;
	    Profiles: Record<string, Array<string>>;
	
	    static createFrom(source: any = {}) {
	        return new Manifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Type = source["Type"];
	        this.Root = source["Root"];
	        this.Services = this.convertValues(source["Services"], Service, true);
	        this.Groups = source["Groups"];
	        this.Profiles = source["Profiles"];
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

