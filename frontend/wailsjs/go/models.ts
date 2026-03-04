export namespace databases {
	
	export class ServiceInfo {
	    type: string;
	    enabled: boolean;
	    running: boolean;
	    autostart: boolean;
	    port: number;
	
	    static createFrom(source: any = {}) {
	        return new ServiceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.enabled = source["enabled"];
	        this.running = source["running"];
	        this.autostart = source["autostart"];
	        this.port = source["port"];
	    }
	}

}

export namespace registry {
	
	export class Site {
	    path: string;
	    domain: string;
	    php_version?: string;
	    node_version?: string;
	    tls: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Site(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.domain = source["domain"];
	        this.php_version = source["php_version"];
	        this.node_version = source["node_version"];
	        this.tls = source["tls"];
	    }
	}

}

