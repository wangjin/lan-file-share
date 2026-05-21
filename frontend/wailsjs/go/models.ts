export namespace model {
	
	export class TransferTask {
	    id: string;
	    type: number;
	    state: number;
	    file_name: string;
	    file_size: number;
	    peer_id: string;
	    peer_name: string;
	    bytes_transferred: number;
	    speed: number;
	    file_md5: string;
	    chunks_total: number;
	    chunks_completed: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    completed_at?: any;
	
	    static createFrom(source: any = {}) {
	        return new TransferTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.state = source["state"];
	        this.file_name = source["file_name"];
	        this.file_size = source["file_size"];
	        this.peer_id = source["peer_id"];
	        this.peer_name = source["peer_name"];
	        this.bytes_transferred = source["bytes_transferred"];
	        this.speed = source["speed"];
	        this.file_md5 = source["file_md5"];
	        this.chunks_total = source["chunks_total"];
	        this.chunks_completed = source["chunks_completed"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.completed_at = this.convertValues(source["completed_at"], null);
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

