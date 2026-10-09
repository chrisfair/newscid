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
	
	export class EngineConfig {
	    name: string;
	    cmd: string;
	    resolvedCmd: string;
	    args: string;
	    dir: string;
	    uci: boolean;
	    elo: number;
	    available: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EngineConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.cmd = source["cmd"];
	        this.resolvedCmd = source["resolvedCmd"];
	        this.args = source["args"];
	        this.dir = source["dir"];
	        this.uci = source["uci"];
	        this.elo = source["elo"];
	        this.available = source["available"];
	    }
	}
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
	export class MoveAnalysis {
	    ply: number;
	    san: string;
	    uci: string;
	    scoreCp: number;
	    mateIn: number;
	    bestMove: string;
	    deltaCp: number;
	    nag: number;
	    comment: string;
	
	    static createFrom(source: any = {}) {
	        return new MoveAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ply = source["ply"];
	        this.san = source["san"];
	        this.uci = source["uci"];
	        this.scoreCp = source["scoreCp"];
	        this.mateIn = source["mateIn"];
	        this.bestMove = source["bestMove"];
	        this.deltaCp = source["deltaCp"];
	        this.nag = source["nag"];
	        this.comment = source["comment"];
	    }
	}
	export class GameAnalysis {
	    whiteAccuracy: number;
	    blackAccuracy: number;
	    whiteBlunders: number;
	    whiteMistakes: number;
	    whiteInaccs: number;
	    blackBlunders: number;
	    blackMistakes: number;
	    blackInaccs: number;
	    moveAnalyses: MoveAnalysis[];
	
	    static createFrom(source: any = {}) {
	        return new GameAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.whiteAccuracy = source["whiteAccuracy"];
	        this.blackAccuracy = source["blackAccuracy"];
	        this.whiteBlunders = source["whiteBlunders"];
	        this.whiteMistakes = source["whiteMistakes"];
	        this.whiteInaccs = source["whiteInaccs"];
	        this.blackBlunders = source["blackBlunders"];
	        this.blackMistakes = source["blackMistakes"];
	        this.blackInaccs = source["blackInaccs"];
	        this.moveAnalyses = this.convertValues(source["moveAnalyses"], MoveAnalysis);
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
	export class RepertoireCheckResult {
	    isRepertoire: boolean;
	    playedMove?: repertoire.RepertoireMove;
	    alternatives: repertoire.RepertoireMove[];
	
	    static createFrom(source: any = {}) {
	        return new RepertoireCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isRepertoire = source["isRepertoire"];
	        this.playedMove = this.convertValues(source["playedMove"], repertoire.RepertoireMove);
	        this.alternatives = this.convertValues(source["alternatives"], repertoire.RepertoireMove);
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
	export class TacticsPuzzle {
	    gameId: number;
	    totalGames: number;
	    baseName: string;
	    event: string;
	    white: string;
	    black: string;
	    startFen: string;
	    moves: string[];
	    sanMoves: string[];
	    turn: string;
	    theme: string;
	
	    static createFrom(source: any = {}) {
	        return new TacticsPuzzle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gameId = source["gameId"];
	        this.totalGames = source["totalGames"];
	        this.baseName = source["baseName"];
	        this.event = source["event"];
	        this.white = source["white"];
	        this.black = source["black"];
	        this.startFen = source["startFen"];
	        this.moves = source["moves"];
	        this.sanMoves = source["sanMoves"];
	        this.turn = source["turn"];
	        this.theme = source["theme"];
	    }
	}

}

export namespace puzzleminer {
	
	export class MinedPuzzle {
	    id: string;
	    gameId: number;
	    white: string;
	    black: string;
	    date: string;
	    event: string;
	    ply: number;
	    startFen: string;
	    turn: string;
	    bestMoveUci: string;
	    bestMoveSan: string;
	    solution: string[];
	    theme: string;
	    evalDeltaCp: number;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new MinedPuzzle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.gameId = source["gameId"];
	        this.white = source["white"];
	        this.black = source["black"];
	        this.date = source["date"];
	        this.event = source["event"];
	        this.ply = source["ply"];
	        this.startFen = source["startFen"];
	        this.turn = source["turn"];
	        this.bestMoveUci = source["bestMoveUci"];
	        this.bestMoveSan = source["bestMoveSan"];
	        this.solution = source["solution"];
	        this.theme = source["theme"];
	        this.evalDeltaCp = source["evalDeltaCp"];
	        this.description = source["description"];
	    }
	}

}

export namespace repertoire {
	
	export class Repertoire {
	    id: string;
	    name: string;
	    color: string;
	    description: string;
	    positions: Record<string, Array<RepertoireMove>>;
	
	    static createFrom(source: any = {}) {
	        return new Repertoire(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.color = source["color"];
	        this.description = source["description"];
	        this.positions = this.convertValues(source["positions"], Array<RepertoireMove>, true);
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
	export class RepertoireMove {
	    uci: string;
	    san: string;
	    nextFen: string;
	    comment?: string;
	    practice: number;
	    correct: number;
	
	    static createFrom(source: any = {}) {
	        return new RepertoireMove(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uci = source["uci"];
	        this.san = source["san"];
	        this.nextFen = source["nextFen"];
	        this.comment = source["comment"];
	        this.practice = source["practice"];
	        this.correct = source["correct"];
	    }
	}
	export class RepertoireSummary {
	    id: string;
	    name: string;
	    color: string;
	    description: string;
	    numMoves: number;
	
	    static createFrom(source: any = {}) {
	        return new RepertoireSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.color = source["color"];
	        this.description = source["description"];
	        this.numMoves = source["numMoves"];
	    }
	}

}

export namespace scid {
	
	export class CompactStats {
	    originalGames: number;
	    compactedGames: number;
	    purgedGames: number;
	    bytesReclaimed: number;
	    originalSg4Size: number;
	    compactedSg4Size: number;
	
	    static createFrom(source: any = {}) {
	        return new CompactStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.originalGames = source["originalGames"];
	        this.compactedGames = source["compactedGames"];
	        this.purgedGames = source["purgedGames"];
	        this.bytesReclaimed = source["bytesReclaimed"];
	        this.originalSg4Size = source["originalSg4Size"];
	        this.compactedSg4Size = source["compactedSg4Size"];
	    }
	}
	export class NameCountPair {
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new NameCountPair(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class DatabaseStatistics {
	    totalGames: number;
	    activeGames: number;
	    deletedGames: number;
	    earliestYear: number;
	    latestYear: number;
	    whiteWins: number;
	    blackWins: number;
	    draws: number;
	    unknownResults: number;
	    averageElo: number;
	    maxElo: number;
	    minElo: number;
	    topOpenings: NameCountPair[];
	    topPlayers: NameCountPair[];
	    totalSizeBytes: number;
	    fileSizes: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseStatistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalGames = source["totalGames"];
	        this.activeGames = source["activeGames"];
	        this.deletedGames = source["deletedGames"];
	        this.earliestYear = source["earliestYear"];
	        this.latestYear = source["latestYear"];
	        this.whiteWins = source["whiteWins"];
	        this.blackWins = source["blackWins"];
	        this.draws = source["draws"];
	        this.unknownResults = source["unknownResults"];
	        this.averageElo = source["averageElo"];
	        this.maxElo = source["maxElo"];
	        this.minElo = source["minElo"];
	        this.topOpenings = this.convertValues(source["topOpenings"], NameCountPair);
	        this.topPlayers = this.convertValues(source["topPlayers"], NameCountPair);
	        this.totalSizeBytes = source["totalSizeBytes"];
	        this.fileSizes = source["fileSizes"];
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
	export class DiscoveredDatabase {
	    name: string;
	    basePath: string;
	    type: string;
	    category: string;
	    numGames: number;
	    description: string;
	    sizeStr: string;
	    exists: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DiscoveredDatabase(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.basePath = source["basePath"];
	        this.type = source["type"];
	        this.category = source["category"];
	        this.numGames = source["numGames"];
	        this.description = source["description"];
	        this.sizeStr = source["sizeStr"];
	        this.exists = source["exists"];
	    }
	}
	export class DuplicateCriteria {
	    exactMoves: boolean;
	    matchPlayers: boolean;
	    matchDate: boolean;
	    matchEvent: boolean;
	    matchResult: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DuplicateCriteria(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.exactMoves = source["exactMoves"];
	        this.matchPlayers = source["matchPlayers"];
	        this.matchDate = source["matchDate"];
	        this.matchEvent = source["matchEvent"];
	        this.matchResult = source["matchResult"];
	    }
	}
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
	export class DuplicateGroup {
	    key: string;
	    gameIds: number[];
	    count: number;
	    games: GameSummary[];
	
	    static createFrom(source: any = {}) {
	        return new DuplicateGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.gameIds = source["gameIds"];
	        this.count = source["count"];
	        this.games = this.convertValues(source["games"], GameSummary);
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
	
	export class ImportStats {
	    totalParsed: number;
	    imported: number;
	    skipped: number;
	    errors: number;
	    durationMs: number;
	    firstGameId: number;
	    lastGameId: number;
	
	    static createFrom(source: any = {}) {
	        return new ImportStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalParsed = source["totalParsed"];
	        this.imported = source["imported"];
	        this.skipped = source["skipped"];
	        this.errors = source["errors"];
	        this.durationMs = source["durationMs"];
	        this.firstGameId = source["firstGameId"];
	        this.lastGameId = source["lastGameId"];
	    }
	}
	
	export class SearchCriteria {
	    white?: string;
	    black?: string;
	    player?: string;
	    event?: string;
	    site?: string;
	    round?: string;
	    dateFrom?: string;
	    dateTo?: string;
	    yearFrom?: number;
	    yearTo?: number;
	    result?: string;
	    eco?: string;
	    ecoFrom?: string;
	    ecoTo?: string;
	    minElo?: number;
	    maxElo?: number;
	    whiteEloMin?: number;
	    whiteEloMax?: number;
	    blackEloMin?: number;
	    blackEloMax?: number;
	    minMoves?: number;
	    maxMoves?: number;
	    hasVariations?: boolean;
	    ignoreDeleted?: boolean;
	    onlyDeleted?: boolean;
	
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
	        this.round = source["round"];
	        this.dateFrom = source["dateFrom"];
	        this.dateTo = source["dateTo"];
	        this.yearFrom = source["yearFrom"];
	        this.yearTo = source["yearTo"];
	        this.result = source["result"];
	        this.eco = source["eco"];
	        this.ecoFrom = source["ecoFrom"];
	        this.ecoTo = source["ecoTo"];
	        this.minElo = source["minElo"];
	        this.maxElo = source["maxElo"];
	        this.whiteEloMin = source["whiteEloMin"];
	        this.whiteEloMax = source["whiteEloMax"];
	        this.blackEloMin = source["blackEloMin"];
	        this.blackEloMax = source["blackEloMax"];
	        this.minMoves = source["minMoves"];
	        this.maxMoves = source["maxMoves"];
	        this.hasVariations = source["hasVariations"];
	        this.ignoreDeleted = source["ignoreDeleted"];
	        this.onlyDeleted = source["onlyDeleted"];
	    }
	}
	export class SortCriteria {
	    field: string;
	    ascending: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SortCriteria(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.ascending = source["ascending"];
	    }
	}
	export class VerificationReport {
	    totalGames: number;
	    activeGames: number;
	    deletedGames: number;
	    corruptedGames: number;
	    status: string;
	    errors: string[];
	    warnings: string[];
	    fileSizes: Record<string, number>;
	    nameCounts: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new VerificationReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalGames = source["totalGames"];
	        this.activeGames = source["activeGames"];
	        this.deletedGames = source["deletedGames"];
	        this.corruptedGames = source["corruptedGames"];
	        this.status = source["status"];
	        this.errors = source["errors"];
	        this.warnings = source["warnings"];
	        this.fileSizes = source["fileSizes"];
	        this.nameCounts = source["nameCounts"];
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

export namespace version {
	
	export class Info {
	    version: string;
	    gitCommit: string;
	    buildDate: string;
	    channel: string;
	    goVersion: string;
	    platform: string;
	    compiler: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.gitCommit = source["gitCommit"];
	        this.buildDate = source["buildDate"];
	        this.channel = source["channel"];
	        this.goVersion = source["goVersion"];
	        this.platform = source["platform"];
	        this.compiler = source["compiler"];
	    }
	}

}

