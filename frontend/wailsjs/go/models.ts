export namespace main {
	
	export class AppConfig {
	    api_key: string;
	    refresh_interval: number;
	    layout_mode: string;
	    display_items: string[];
	    window_width: number;
	    glass_mode: boolean;
	    auto_start: boolean;
	    window_mode: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.api_key = source["api_key"];
	        this.refresh_interval = source["refresh_interval"];
	        this.layout_mode = source["layout_mode"];
	        this.display_items = source["display_items"];
	        this.window_width = source["window_width"];
	        this.glass_mode = source["glass_mode"];
	        this.auto_start = source["auto_start"];
	        this.window_mode = source["window_mode"];
	    }
	}
	export class QuotaMonitorData {
	    level: string;
	    five_hour_used_pct: number;
	    five_hour_left_pct: number;
	    next_refresh_time: string;
	    weekly_used_pct: number;
	    weekly_left_pct: number;
	    mcp_current: number;
	    mcp_total: number;
	    mcp_left: number;
	    mcp_usage_pct: number;
	    last_error: string;
	
	    static createFrom(source: any = {}) {
	        return new QuotaMonitorData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.five_hour_used_pct = source["five_hour_used_pct"];
	        this.five_hour_left_pct = source["five_hour_left_pct"];
	        this.next_refresh_time = source["next_refresh_time"];
	        this.weekly_used_pct = source["weekly_used_pct"];
	        this.weekly_left_pct = source["weekly_left_pct"];
	        this.mcp_current = source["mcp_current"];
	        this.mcp_total = source["mcp_total"];
	        this.mcp_left = source["mcp_left"];
	        this.mcp_usage_pct = source["mcp_usage_pct"];
	        this.last_error = source["last_error"];
	    }
	}

}

