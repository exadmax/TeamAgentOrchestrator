export namespace clients {
	
	export class Command {
	    cmd: string[];
	    prompt_flag: string;
	
	    static createFrom(source: any = {}) {
	        return new Command(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cmd = source["cmd"];
	        this.prompt_flag = source["prompt_flag"];
	    }
	}
	export class Detected {
	    id: string;
	    name: string;
	    exe: string;
	    npm_package: string;
	    color: string;
	    output: string;
	    // Go type: Command
	    headless: any;
	    // Go type: Command
	    interactive: any;
	    model_flag: string;
	    models: string[];
	    tiers: Record<string, string>;
	    env?: string[];
	    path: string;
	    installed: boolean;
	    version: string;
	    latest: string;
	
	    static createFrom(source: any = {}) {
	        return new Detected(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.exe = source["exe"];
	        this.npm_package = source["npm_package"];
	        this.color = source["color"];
	        this.output = source["output"];
	        this.headless = this.convertValues(source["headless"], null);
	        this.interactive = this.convertValues(source["interactive"], null);
	        this.model_flag = source["model_flag"];
	        this.models = source["models"];
	        this.tiers = source["tiers"];
	        this.env = source["env"];
	        this.path = source["path"];
	        this.installed = source["installed"];
	        this.version = source["version"];
	        this.latest = source["latest"];
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

export namespace main {
	
	export class AgentView {
	    id: string;
	    name: string;
	    role: string;
	    emblem: string;
	    face: string;
	    client: string;
	    model?: string;
	    tier?: string;
	    parent?: string;
	    instructions?: string;
	    paused?: boolean;
	    status: string;
	    model_in_use: string;
	    client_name: string;
	    client_color: string;
	    last_line: string;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.role = source["role"];
	        this.emblem = source["emblem"];
	        this.face = source["face"];
	        this.client = source["client"];
	        this.model = source["model"];
	        this.tier = source["tier"];
	        this.parent = source["parent"];
	        this.instructions = source["instructions"];
	        this.paused = source["paused"];
	        this.status = source["status"];
	        this.model_in_use = source["model_in_use"];
	        this.client_name = source["client_name"];
	        this.client_color = source["client_color"];
	        this.last_line = source["last_line"];
	        this.command = source["command"];
	    }
	}
	export class Line {
	    agent: string;
	    // Go type: time
	    at: any;
	    kind: string;
	    text: string;
	    tool?: string;
	    error?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Line(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent = source["agent"];
	        this.at = this.convertValues(source["at"], null);
	        this.kind = source["kind"];
	        this.text = source["text"];
	        this.tool = source["tool"];
	        this.error = source["error"];
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
	export class Snapshot {
	    project: string;
	    dir: string;
	    agents: AgentView[];
	    clients: clients.Detected[];
	    activity: Line[];
	    emblems: string[];
	    tiers: string[];
	    clocks: store.Clock[];
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.project = source["project"];
	        this.dir = source["dir"];
	        this.agents = this.convertValues(source["agents"], AgentView);
	        this.clients = this.convertValues(source["clients"], clients.Detected);
	        this.activity = this.convertValues(source["activity"], Line);
	        this.emblems = source["emblems"];
	        this.tiers = source["tiers"];
	        this.clocks = this.convertValues(source["clocks"], store.Clock);
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
	export class TokenRow {
	    agent: string;
	    name: string;
	    client: string;
	    color: string;
	    model: string;
	    status: string;
	    input: number;
	    output: number;
	    total: number;
	    cost_usd: number;
	    sessions: number;
	    share: number;
	
	    static createFrom(source: any = {}) {
	        return new TokenRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent = source["agent"];
	        this.name = source["name"];
	        this.client = source["client"];
	        this.color = source["color"];
	        this.model = source["model"];
	        this.status = source["status"];
	        this.input = source["input"];
	        this.output = source["output"];
	        this.total = source["total"];
	        this.cost_usd = source["cost_usd"];
	        this.sessions = source["sessions"];
	        this.share = source["share"];
	    }
	}

}

export namespace store {
	
	export class Clock {
	    client: string;
	    model: string;
	    // Go type: time
	    until: any;
	    left: string;
	    // Go type: time
	    recorded: any;
	    reason: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new Clock(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.client = source["client"];
	        this.model = source["model"];
	        this.until = this.convertValues(source["until"], null);
	        this.left = source["left"];
	        this.recorded = this.convertValues(source["recorded"], null);
	        this.reason = source["reason"];
	        this.source = source["source"];
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

export namespace team {
	
	export class Agent {
	    id: string;
	    name: string;
	    role: string;
	    emblem: string;
	    face: string;
	    client: string;
	    model?: string;
	    tier?: string;
	    parent?: string;
	    instructions?: string;
	    paused?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Agent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.role = source["role"];
	        this.emblem = source["emblem"];
	        this.face = source["face"];
	        this.client = source["client"];
	        this.model = source["model"];
	        this.tier = source["tier"];
	        this.parent = source["parent"];
	        this.instructions = source["instructions"];
	        this.paused = source["paused"];
	    }
	}

}

