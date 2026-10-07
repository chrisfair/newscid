export namespace chess {
	
	export class Move {
	    From: number;
	    To: number;
	    Promo: number;
	    Capture: number;
	
	    static createFrom(source: any = {}) {
	        return new Move(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.From = source["From"];
	        this.To = source["To"];
	        this.Promo = source["Promo"];
	        this.Capture = source["Capture"];
	    }
	}
	export class MoveNode {
	    move: Move;
	    san: string;
	    uci: string;
	    fen: string;
	    comment?: string;
	    nags?: number[];
	    variations?: MoveNode[];
	    next?: MoveNode;
	
	    static createFrom(source: any = {}) {
	        return new MoveNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.move = this.convertValues(source["move"], Move);
	        this.san = source["san"];
	        this.uci = source["uci"];
	        this.fen = source["fen"];
	        this.comment = source["comment"];
	        this.nags = source["nags"];
	        this.variations = this.convertValues(source["variations"], MoveNode);
	        this.next = this.convertValues(source["next"], MoveNode);
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
	export class Game {
	    id: number;
	    tags: Record<string, string>;
	    startFen: string;
	    rootMove?: MoveNode;
	    result: string;
	    plyCount: number;
	
	    static createFrom(source: any = {}) {
	        return new Game(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.tags = source["tags"];
	        this.startFen = source["startFen"];
	        this.rootMove = this.convertValues(source["rootMove"], MoveNode);
	        this.result = source["result"];
	        this.plyCount = source["plyCount"];
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

export namespace eco {
	
	export class Entry {
	    code: string;
	    name: string;
	    moves: string;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.name = source["name"];
	        this.moves = source["moves"];
	    }
	}

}

export namespace engine {
	
	export class EvalInfo {
	    depth: number;
	    scoreCp: number;
	    mateIn: number;
	    nps: number;
	    nodes: number;
	    pv: string[];
	    bestMove: string;
	
	    static createFrom(source: any = {}) {
	        return new EvalInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.depth = source["depth"];
	        this.scoreCp = source["scoreCp"];
	        this.mateIn = source["mateIn"];
	        this.nps = source["nps"];
	        this.nodes = source["nodes"];
	        this.pv = source["pv"];
	        this.bestMove = source["bestMove"];
	    }
	}

}

export namespace main {
	
	export class BoardState {
	    fen: string;
	    turn: string;
	    inCheck: boolean;
	    isCheckmate: boolean;
	    isStalemate: boolean;
	    legalMoves: string[];
	    fullMove: number;
	    halfMove: number;
	
	    static createFrom(source: any = {}) {
	        return new BoardState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fen = source["fen"];
	        this.turn = source["turn"];
	        this.inCheck = source["inCheck"];
	        this.isCheckmate = source["isCheckmate"];
	        this.isStalemate = source["isStalemate"];
	        this.legalMoves = source["legalMoves"];
	        this.fullMove = source["fullMove"];
	        this.halfMove = source["halfMove"];
	    }
	}
	export class DatabaseInfo {
	    basePath: string;
	    name: string;
	    numGames: number;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.basePath = source["basePath"];
	        this.name = source["name"];
	        this.numGames = source["numGames"];
	        this.description = source["description"];
	    }
	}
	export class GamesPage {
	    total: number;
	    games: scid.GameSummary[];
	
	    static createFrom(source: any = {}) {
	        return new GamesPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.games = this.convertValues(source["games"], scid.GameSummary);
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

export namespace scid {
	
	export class GameSummary {
	    id: number;
	    white: string;
	    black: string;
	    result: string;
	    date: string;
	    event: string;
	    site: string;
	    round: string;
	    whiteElo: number;
	    blackElo: number;
	    eco: string;
	    moves: number;
	    deleted: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GameSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.white = source["white"];
	        this.black = source["black"];
	        this.result = source["result"];
	        this.date = source["date"];
	        this.event = source["event"];
	        this.site = source["site"];
	        this.round = source["round"];
	        this.whiteElo = source["whiteElo"];
	        this.blackElo = source["blackElo"];
	        this.eco = source["eco"];
	        this.moves = source["moves"];
	        this.deleted = source["deleted"];
	    }
	}
	export class SearchCriteria {
	    white?: string;
	    black?: string;
	    player?: string;
	    event?: string;
	    site?: string;
	    dateFrom?: string;
	    dateTo?: string;
	    result?: string;
	    eco?: string;
	    minElo?: number;
	    maxElo?: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchCriteria(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.white = source["white"];
	        this.black = source["black"];
	        this.player = source["player"];
	        this.event = source["event"];
	        this.site = source["site"];
	        this.dateFrom = source["dateFrom"];
	        this.dateTo = source["dateTo"];
	        this.result = source["result"];
	        this.eco = source["eco"];
	        this.minElo = source["minElo"];
	        this.maxElo = source["maxElo"];
	    }
	}

}

export namespace sets {
	
	export class BoardThemeInfo {
	    id: string;
	    name: string;
	    light: string;
	    dark: string;
	    highlight: string;
	
	    static createFrom(source: any = {}) {
	        return new BoardThemeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.light = source["light"];
	        this.dark = source["dark"];
	        this.highlight = source["highlight"];
	    }
	}
	export class PieceSetInfo {
	    id: string;
	    name: string;
	    isScidNative: boolean;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new PieceSetInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.isScidNative = source["isScidNative"];
	        this.description = source["description"];
	    }
	}

}

export namespace tree {
	
	export class MoveStat {
	    move: string;
	    uci: string;
	    count: number;
	    whiteWins: number;
	    draws: number;
	    blackWins: number;
	    scorePct: number;
	    avgElo: number;
	    eco?: string;
	
	    static createFrom(source: any = {}) {
	        return new MoveStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.move = source["move"];
	        this.uci = source["uci"];
	        this.count = source["count"];
	        this.whiteWins = source["whiteWins"];
	        this.draws = source["draws"];
	        this.blackWins = source["blackWins"];
	        this.scorePct = source["scorePct"];
	        this.avgElo = source["avgElo"];
	        this.eco = source["eco"];
	    }
	}
	export class TreeResult {
	    fen: string;
	    totalGames: number;
	    moves: MoveStat[];
	
	    static createFrom(source: any = {}) {
	        return new TreeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fen = source["fen"];
	        this.totalGames = source["totalGames"];
	        this.moves = this.convertValues(source["moves"], MoveStat);
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

