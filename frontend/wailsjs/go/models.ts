export namespace blundervault {
	
	export class Puzzle {
	    id: string;
	    fen: string;
	    playerName: string;
	    userColor: string;
	    playedMove: string;
	    bestMove: string;
	    deltaCp: number;
	    event: string;
	    date: string;
	    explanation?: string;
	    repetitions: number;
	    intervalDays: number;
	    easeFactor: number;
	    // Go type: time
	    nextReview: any;
	    // Go type: time
	    lastReview: any;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new Puzzle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fen = source["fen"];
	        this.playerName = source["playerName"];
	        this.userColor = source["userColor"];
	        this.playedMove = source["playedMove"];
	        this.bestMove = source["bestMove"];
	        this.deltaCp = source["deltaCp"];
	        this.event = source["event"];
	        this.date = source["date"];
	        this.explanation = source["explanation"];
	        this.repetitions = source["repetitions"];
	        this.intervalDays = source["intervalDays"];
	        this.easeFactor = source["easeFactor"];
	        this.nextReview = this.convertValues(source["nextReview"], null);
	        this.lastReview = this.convertValues(source["lastReview"], null);
	        this.status = source["status"];
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

export namespace chess {
	
	export class Board {
	    Squares: number[];
	    Turn: number;
	    Castling: number;
	    EPSquare: number;
	    HalfMove: number;
	    FullMove: number;
	    PieceList: number[][];
	    PieceCount: number[];
	    PiecePos: number[];
	
	    static createFrom(source: any = {}) {
	        return new Board(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Squares = source["Squares"];
	        this.Turn = source["Turn"];
	        this.Castling = source["Castling"];
	        this.EPSquare = source["EPSquare"];
	        this.HalfMove = source["HalfMove"];
	        this.FullMove = source["FullMove"];
	        this.PieceList = source["PieceList"];
	        this.PieceCount = source["PieceCount"];
	        this.PiecePos = source["PiecePos"];
	    }
	}
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
	    clock?: string;
	    emt?: string;
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
	        this.clock = source["clock"];
	        this.emt = source["emt"];
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

export namespace chess960 {
	
	export class Position {
	    id: number;
	    fen: string;
	    pieces: string;
	    isClassic: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Position(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fen = source["fen"];
	        this.pieces = source["pieces"];
	        this.isClassic = source["isClassic"];
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
	export class SectionSummary {
	    letter: string;
	    title: string;
	    description: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new SectionSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.letter = source["letter"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.count = source["count"];
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
	export class PVLine {
	    multipv: number;
	    depth: number;
	    scoreCp: number;
	    mateIn: number;
	    nps: number;
	    nodes: number;
	    pv: string[];
	    moveUci: string;
	    san: string;
	    pvFormatted: string;
	
	    static createFrom(source: any = {}) {
	        return new PVLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.multipv = source["multipv"];
	        this.depth = source["depth"];
	        this.scoreCp = source["scoreCp"];
	        this.mateIn = source["mateIn"];
	        this.nps = source["nps"];
	        this.nodes = source["nodes"];
	        this.pv = source["pv"];
	        this.moveUci = source["moveUci"];
	        this.san = source["san"];
	        this.pvFormatted = source["pvFormatted"];
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
	    multiPv: number;
	    lines: PVLine[];
	
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
	        this.multiPv = source["multiPv"];
	        this.lines = this.convertValues(source["lines"], PVLine);
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

export namespace epd {
	
	export class EPDRecord {
	    fenCore: string;
	    fullFen: string;
	    opcodes: Record<string, string>;
	    bestMove?: string;
	    avoidMove?: string;
	    id?: string;
	    comment?: string;
	    evalCp?: number;
	
	    static createFrom(source: any = {}) {
	        return new EPDRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fenCore = source["fenCore"];
	        this.fullFen = source["fullFen"];
	        this.opcodes = source["opcodes"];
	        this.bestMove = source["bestMove"];
	        this.avoidMove = source["avoidMove"];
	        this.id = source["id"];
	        this.comment = source["comment"];
	        this.evalCp = source["evalCp"];
	    }
	}

}

export namespace explainer {
	
	export class Explanation {
	    moveUci: string;
	    moveSan: string;
	    assessment: string;
	    summary: string;
	    tacticalMotifs: string[];
	    strategicConcepts: string[];
	    detailedText: string;
	    bestAlternative?: string;
	
	    static createFrom(source: any = {}) {
	        return new Explanation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.moveUci = source["moveUci"];
	        this.moveSan = source["moveSan"];
	        this.assessment = source["assessment"];
	        this.summary = source["summary"];
	        this.tacticalMotifs = source["tacticalMotifs"];
	        this.strategicConcepts = source["strategicConcepts"];
	        this.detailedText = source["detailedText"];
	        this.bestAlternative = source["bestAlternative"];
	    }
	}

}

export namespace fairplay {
	
	export class BannedValidationSummary {
	    accountsChecked: number;
	    confirmedBannedCount: number;
	    gamesVsBanned: number;
	    bannedInTopTwoBands: number;
	    bannedTopTwoRatePct: number;
	    openTopTwoRatePct: number;
	    multiplierVsOpen: number;
	    confirmedBannedHandles?: string[];
	
	    static createFrom(source: any = {}) {
	        return new BannedValidationSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accountsChecked = source["accountsChecked"];
	        this.confirmedBannedCount = source["confirmedBannedCount"];
	        this.gamesVsBanned = source["gamesVsBanned"];
	        this.bannedInTopTwoBands = source["bannedInTopTwoBands"];
	        this.bannedTopTwoRatePct = source["bannedTopTwoRatePct"];
	        this.openTopTwoRatePct = source["openTopTwoRatePct"];
	        this.multiplierVsOpen = source["multiplierVsOpen"];
	        this.confirmedBannedHandles = source["confirmedBannedHandles"];
	    }
	}
	export class BenchmarkBandSummary {
	    bandIndex: number;
	    bandName: string;
	    description: string;
	    thresholdScore: number;
	    selfCount: number;
	    selfPct: number;
	    opponentCount: number;
	    opponentPct: number;
	    expectedOppCount: number;
	    excessOppCount: number;
	    bannedOppCount: number;
	    bannedOppPct: number;
	
	    static createFrom(source: any = {}) {
	        return new BenchmarkBandSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bandIndex = source["bandIndex"];
	        this.bandName = source["bandName"];
	        this.description = source["description"];
	        this.thresholdScore = source["thresholdScore"];
	        this.selfCount = source["selfCount"];
	        this.selfPct = source["selfPct"];
	        this.opponentCount = source["opponentCount"];
	        this.opponentPct = source["opponentPct"];
	        this.expectedOppCount = source["expectedOppCount"];
	        this.excessOppCount = source["excessOppCount"];
	        this.bannedOppCount = source["bannedOppCount"];
	        this.bannedOppPct = source["bannedOppPct"];
	    }
	}
	export class CrossControlDivergence {
	    hasMultipleControls: boolean;
	    baselineControl: string;
	    suspectControl: string;
	    iprDelta: number;
	    acplDelta: number;
	    cadenceCvDelta: number;
	    divergenceScore: number;
	    explanation: string;
	
	    static createFrom(source: any = {}) {
	        return new CrossControlDivergence(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasMultipleControls = source["hasMultipleControls"];
	        this.baselineControl = source["baselineControl"];
	        this.suspectControl = source["suspectControl"];
	        this.iprDelta = source["iprDelta"];
	        this.acplDelta = source["acplDelta"];
	        this.cadenceCvDelta = source["cadenceCvDelta"];
	        this.divergenceScore = source["divergenceScore"];
	        this.explanation = source["explanation"];
	    }
	}
	export class DailyFairPlaySummary {
	    date: string;
	    gamesCount: number;
	    timeControls: string[];
	    primaryTimeControl: string;
	    winRatePct: number;
	    avgAcpl: number;
	    top1AgreementPct: number;
	    avgMoveTimeCV: number;
	    intrinsicPerformance: number;
	    dailyIssueScore: number;
	    peakGameId: number;
	    classification: string;
	
	    static createFrom(source: any = {}) {
	        return new DailyFairPlaySummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.gamesCount = source["gamesCount"];
	        this.timeControls = source["timeControls"];
	        this.primaryTimeControl = source["primaryTimeControl"];
	        this.winRatePct = source["winRatePct"];
	        this.avgAcpl = source["avgAcpl"];
	        this.top1AgreementPct = source["top1AgreementPct"];
	        this.avgMoveTimeCV = source["avgMoveTimeCV"];
	        this.intrinsicPerformance = source["intrinsicPerformance"];
	        this.dailyIssueScore = source["dailyIssueScore"];
	        this.peakGameId = source["peakGameId"];
	        this.classification = source["classification"];
	    }
	}
	export class GameFairPlayRecord {
	    gameId: number;
	    date: string;
	    event: string;
	    site: string;
	    timeControlRaw: string;
	    timeControlCategory: string;
	    playerColor: string;
	    playerElo: number;
	    opponentName: string;
	    opponentElo: number;
	    result: string;
	    playerScore: number;
	    totalPlies: number;
	    nonBookMoves: number;
	    hasClockData: boolean;
	    hasEvalData: boolean;
	    avgMoveTimeSec: number;
	    moveTimeStdDevSec: number;
	    moveTimeCV: number;
	    roboticCadencePct: number;
	    delayedRecaptureCount: number;
	    acpl: number;
	    top1AgreementPct: number;
	    criticalAccuracyPct: number;
	    blunderCount: number;
	    intrinsicPerformanceElo: number;
	    gameIssueScore: number;
	    flags: string[];
	
	    static createFrom(source: any = {}) {
	        return new GameFairPlayRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gameId = source["gameId"];
	        this.date = source["date"];
	        this.event = source["event"];
	        this.site = source["site"];
	        this.timeControlRaw = source["timeControlRaw"];
	        this.timeControlCategory = source["timeControlCategory"];
	        this.playerColor = source["playerColor"];
	        this.playerElo = source["playerElo"];
	        this.opponentName = source["opponentName"];
	        this.opponentElo = source["opponentElo"];
	        this.result = source["result"];
	        this.playerScore = source["playerScore"];
	        this.totalPlies = source["totalPlies"];
	        this.nonBookMoves = source["nonBookMoves"];
	        this.hasClockData = source["hasClockData"];
	        this.hasEvalData = source["hasEvalData"];
	        this.avgMoveTimeSec = source["avgMoveTimeSec"];
	        this.moveTimeStdDevSec = source["moveTimeStdDevSec"];
	        this.moveTimeCV = source["moveTimeCV"];
	        this.roboticCadencePct = source["roboticCadencePct"];
	        this.delayedRecaptureCount = source["delayedRecaptureCount"];
	        this.acpl = source["acpl"];
	        this.top1AgreementPct = source["top1AgreementPct"];
	        this.criticalAccuracyPct = source["criticalAccuracyPct"];
	        this.blunderCount = source["blunderCount"];
	        this.intrinsicPerformanceElo = source["intrinsicPerformanceElo"];
	        this.gameIssueScore = source["gameIssueScore"];
	        this.flags = source["flags"];
	    }
	}
	export class HardMoveMoment {
	    ply: number;
	    moveNumber: number;
	    side: string;
	    playedSan: string;
	    bestSan: string;
	    foundTopChoice: boolean;
	    humanProb: number;
	    evalLossCp: number;
	    evalGapTop2Cp: number;
	    plausibleOptions: number;
	
	    static createFrom(source: any = {}) {
	        return new HardMoveMoment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ply = source["ply"];
	        this.moveNumber = source["moveNumber"];
	        this.side = source["side"];
	        this.playedSan = source["playedSan"];
	        this.bestSan = source["bestSan"];
	        this.foundTopChoice = source["foundTopChoice"];
	        this.humanProb = source["humanProb"];
	        this.evalLossCp = source["evalLossCp"];
	        this.evalGapTop2Cp = source["evalGapTop2Cp"];
	        this.plausibleOptions = source["plausibleOptions"];
	    }
	}
	export class GameHardMovePerformance {
	    gameId: number;
	    date: string;
	    timeControl: string;
	    side: string;
	    playerName: string;
	    anonymousLabel: string;
	    playerElo: number;
	    opponentName: string;
	    result: string;
	    totalNonBookMoves: number;
	    hardMoveCount: number;
	    hardMovesFound: number;
	    hardMoveHitRatePct: number;
	    expectedHardHits: number;
	    tailProbability: number;
	    unusualnessScore: number;
	    zScore: number;
	    bandIndex: number;
	    bandName: string;
	    isBannedOpponent: boolean;
	    banSource?: string;
	    hardMoments?: HardMoveMoment[];
	
	    static createFrom(source: any = {}) {
	        return new GameHardMovePerformance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gameId = source["gameId"];
	        this.date = source["date"];
	        this.timeControl = source["timeControl"];
	        this.side = source["side"];
	        this.playerName = source["playerName"];
	        this.anonymousLabel = source["anonymousLabel"];
	        this.playerElo = source["playerElo"];
	        this.opponentName = source["opponentName"];
	        this.result = source["result"];
	        this.totalNonBookMoves = source["totalNonBookMoves"];
	        this.hardMoveCount = source["hardMoveCount"];
	        this.hardMovesFound = source["hardMovesFound"];
	        this.hardMoveHitRatePct = source["hardMoveHitRatePct"];
	        this.expectedHardHits = source["expectedHardHits"];
	        this.tailProbability = source["tailProbability"];
	        this.unusualnessScore = source["unusualnessScore"];
	        this.zScore = source["zScore"];
	        this.bandIndex = source["bandIndex"];
	        this.bandName = source["bandName"];
	        this.isBannedOpponent = source["isBannedOpponent"];
	        this.banSource = source["banSource"];
	        this.hardMoments = this.convertValues(source["hardMoments"], HardMoveMoment);
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
	export class SelectiveCheatingPoint {
	    label: string;
	    swappedMovesSpec: string;
	    avgSwappedPerGame: number;
	    topTwoBandsCaughtPct: number;
	    topFivePctCaughtPct: number;
	    avgUnusualnessScore: number;
	
	    static createFrom(source: any = {}) {
	        return new SelectiveCheatingPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.swappedMovesSpec = source["swappedMovesSpec"];
	        this.avgSwappedPerGame = source["avgSwappedPerGame"];
	        this.topTwoBandsCaughtPct = source["topTwoBandsCaughtPct"];
	        this.topFivePctCaughtPct = source["topFivePctCaughtPct"];
	        this.avgUnusualnessScore = source["avgUnusualnessScore"];
	    }
	}
	export class HardMoveBenchmarkReport {
	    benchmarkPlayer: string;
	    timeControlFilter: string;
	    hardMoveProbCutoff: number;
	    totalGamesScanned: number;
	    selfGamesQualified: number;
	    opponentGamesQualified: number;
	    totalHardMovesAnalyzed: number;
	    selfAvgHardHitRatePct: number;
	    opponentAvgHardHitRatePct: number;
	    selfExpectedHitRatePct: number;
	    opponentExpectedHitRatePct: number;
	    topTwoBandThreshold: number;
	    topHalfPctThreshold: number;
	    selfInTopTwoBands: number;
	    selfTopTwoRatePct: number;
	    opponentInTopTwoBands: number;
	    opponentTopTwoRatePct: number;
	    expectedOpponentTopTwo: number;
	    excessOpponentGames: number;
	    excessGamesLowEstimate: number;
	    excessGamesHighEstimate: number;
	    excessPctOfAllGames: number;
	    excessPValue: number;
	    oddsOneIn: number;
	    selfInTop100Count: number;
	    opponentInTop100Count: number;
	    bands: BenchmarkBandSummary[];
	    selectiveSimulation: SelectiveCheatingPoint[];
	    bannedValidation: BannedValidationSummary;
	    top100Games: GameHardMovePerformance[];
	    verdictHeadline: string;
	    verdictFindings: string[];
	
	    static createFrom(source: any = {}) {
	        return new HardMoveBenchmarkReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.benchmarkPlayer = source["benchmarkPlayer"];
	        this.timeControlFilter = source["timeControlFilter"];
	        this.hardMoveProbCutoff = source["hardMoveProbCutoff"];
	        this.totalGamesScanned = source["totalGamesScanned"];
	        this.selfGamesQualified = source["selfGamesQualified"];
	        this.opponentGamesQualified = source["opponentGamesQualified"];
	        this.totalHardMovesAnalyzed = source["totalHardMovesAnalyzed"];
	        this.selfAvgHardHitRatePct = source["selfAvgHardHitRatePct"];
	        this.opponentAvgHardHitRatePct = source["opponentAvgHardHitRatePct"];
	        this.selfExpectedHitRatePct = source["selfExpectedHitRatePct"];
	        this.opponentExpectedHitRatePct = source["opponentExpectedHitRatePct"];
	        this.topTwoBandThreshold = source["topTwoBandThreshold"];
	        this.topHalfPctThreshold = source["topHalfPctThreshold"];
	        this.selfInTopTwoBands = source["selfInTopTwoBands"];
	        this.selfTopTwoRatePct = source["selfTopTwoRatePct"];
	        this.opponentInTopTwoBands = source["opponentInTopTwoBands"];
	        this.opponentTopTwoRatePct = source["opponentTopTwoRatePct"];
	        this.expectedOpponentTopTwo = source["expectedOpponentTopTwo"];
	        this.excessOpponentGames = source["excessOpponentGames"];
	        this.excessGamesLowEstimate = source["excessGamesLowEstimate"];
	        this.excessGamesHighEstimate = source["excessGamesHighEstimate"];
	        this.excessPctOfAllGames = source["excessPctOfAllGames"];
	        this.excessPValue = source["excessPValue"];
	        this.oddsOneIn = source["oddsOneIn"];
	        this.selfInTop100Count = source["selfInTop100Count"];
	        this.opponentInTop100Count = source["opponentInTop100Count"];
	        this.bands = this.convertValues(source["bands"], BenchmarkBandSummary);
	        this.selectiveSimulation = this.convertValues(source["selectiveSimulation"], SelectiveCheatingPoint);
	        this.bannedValidation = this.convertValues(source["bannedValidation"], BannedValidationSummary);
	        this.top100Games = this.convertValues(source["top100Games"], GameHardMovePerformance);
	        this.verdictHeadline = source["verdictHeadline"];
	        this.verdictFindings = source["verdictFindings"];
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
	
	export class TimeControlIssueProfile {
	    category: string;
	    gamesAnalyzed: number;
	    daysActive: number;
	    daysWithElevatedIssue: number;
	    dailyConsistencyPct: number;
	    winRatePct: number;
	    avgOfficialElo: number;
	    intrinsicPerformanceElo: number;
	    eloOverperformanceDelta: number;
	    avgAcpl: number;
	    top1AgreementPct: number;
	    criticalAccuracyPct: number;
	    blunderFreeGamePct: number;
	    avgMoveTimeSec: number;
	    avgMoveTimeCV: number;
	    roboticCadenceGamePct: number;
	    consistentIssueScore: number;
	    zScore: number;
	    consistencyClassification: string;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new TimeControlIssueProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.category = source["category"];
	        this.gamesAnalyzed = source["gamesAnalyzed"];
	        this.daysActive = source["daysActive"];
	        this.daysWithElevatedIssue = source["daysWithElevatedIssue"];
	        this.dailyConsistencyPct = source["dailyConsistencyPct"];
	        this.winRatePct = source["winRatePct"];
	        this.avgOfficialElo = source["avgOfficialElo"];
	        this.intrinsicPerformanceElo = source["intrinsicPerformanceElo"];
	        this.eloOverperformanceDelta = source["eloOverperformanceDelta"];
	        this.avgAcpl = source["avgAcpl"];
	        this.top1AgreementPct = source["top1AgreementPct"];
	        this.criticalAccuracyPct = source["criticalAccuracyPct"];
	        this.blunderFreeGamePct = source["blunderFreeGamePct"];
	        this.avgMoveTimeSec = source["avgMoveTimeSec"];
	        this.avgMoveTimeCV = source["avgMoveTimeCV"];
	        this.roboticCadenceGamePct = source["roboticCadenceGamePct"];
	        this.consistentIssueScore = source["consistentIssueScore"];
	        this.zScore = source["zScore"];
	        this.consistencyClassification = source["consistencyClassification"];
	        this.summary = source["summary"];
	    }
	}
	export class PlayerFairPlayDossier {
	    playerName: string;
	    totalGamesAnalyzed: number;
	    totalDaysActive: number;
	    overallIssueScore: number;
	    highestRiskTimeControl: string;
	    highestControlIssueScore: number;
	    overallClassification: string;
	    timeControlProfiles: TimeControlIssueProfile[];
	    dailyTimeline: DailyFairPlaySummary[];
	    crossDivergence: CrossControlDivergence;
	    flaggedGames: GameFairPlayRecord[];
	    allGames: GameFairPlayRecord[];
	    keyFindings: string[];
	    reportMarkdown: string;
	
	    static createFrom(source: any = {}) {
	        return new PlayerFairPlayDossier(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.playerName = source["playerName"];
	        this.totalGamesAnalyzed = source["totalGamesAnalyzed"];
	        this.totalDaysActive = source["totalDaysActive"];
	        this.overallIssueScore = source["overallIssueScore"];
	        this.highestRiskTimeControl = source["highestRiskTimeControl"];
	        this.highestControlIssueScore = source["highestControlIssueScore"];
	        this.overallClassification = source["overallClassification"];
	        this.timeControlProfiles = this.convertValues(source["timeControlProfiles"], TimeControlIssueProfile);
	        this.dailyTimeline = this.convertValues(source["dailyTimeline"], DailyFairPlaySummary);
	        this.crossDivergence = this.convertValues(source["crossDivergence"], CrossControlDivergence);
	        this.flaggedGames = this.convertValues(source["flaggedGames"], GameFairPlayRecord);
	        this.allGames = this.convertValues(source["allGames"], GameFairPlayRecord);
	        this.keyFindings = source["keyFindings"];
	        this.reportMarkdown = source["reportMarkdown"];
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

export namespace gameexport {
	
	export class HTMLExportOptions {
	    title?: string;
	    theme?: string;
	    includeDiagram: boolean;
	    diagramPly: number;
	    includeComments: boolean;
	    includeVariations: boolean;
	
	    static createFrom(source: any = {}) {
	        return new HTMLExportOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.theme = source["theme"];
	        this.includeDiagram = source["includeDiagram"];
	        this.diagramPly = source["diagramPly"];
	        this.includeComments = source["includeComments"];
	        this.includeVariations = source["includeVariations"];
	    }
	}
	export class LaTeXExportOptions {
	    documentClass?: string;
	    package?: string;
	    includeDiagram: boolean;
	    diagramPly: number;
	
	    static createFrom(source: any = {}) {
	        return new LaTeXExportOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.documentClass = source["documentClass"];
	        this.package = source["package"];
	        this.includeDiagram = source["includeDiagram"];
	        this.diagramPly = source["diagramPly"];
	    }
	}

}

export namespace imbalance {
	
	export class PawnStructureStats {
	    doubledPawns: number;
	    isolatedPawns: number;
	    passedPawns: number;
	    maxPassedRank: number;
	
	    static createFrom(source: any = {}) {
	        return new PawnStructureStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.doubledPawns = source["doubledPawns"];
	        this.isolatedPawns = source["isolatedPawns"];
	        this.passedPawns = source["passedPawns"];
	        this.maxPassedRank = source["maxPassedRank"];
	    }
	}
	export class SideEvaluation {
	    materialPoints: number;
	    hasBishopPair: boolean;
	    pawnStructure: PawnStructureStats;
	    kingSafety: number;
	    controlledTerritory: number;
	    openFilesControlled: number;
	
	    static createFrom(source: any = {}) {
	        return new SideEvaluation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.materialPoints = source["materialPoints"];
	        this.hasBishopPair = source["hasBishopPair"];
	        this.pawnStructure = this.convertValues(source["pawnStructure"], PawnStructureStats);
	        this.kingSafety = source["kingSafety"];
	        this.controlledTerritory = source["controlledTerritory"];
	        this.openFilesControlled = source["openFilesControlled"];
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
	export class Report {
	    fen: string;
	    white: SideEvaluation;
	    black: SideEvaluation;
	    materialDelta: number;
	    estimatedEvalCp: number;
	    positionalSummary: string;
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fen = source["fen"];
	        this.white = this.convertValues(source["white"], SideEvaluation);
	        this.black = this.convertValues(source["black"], SideEvaluation);
	        this.materialDelta = source["materialDelta"];
	        this.estimatedEvalCp = source["estimatedEvalCp"];
	        this.positionalSummary = source["positionalSummary"];
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
	
	export class SavedFloaterRect {
	    x: number;
	    y: number;
	    width: number;
	    height: number;
	    minimized: boolean;
	    zIndex: number;
	
	    static createFrom(source: any = {}) {
	        return new SavedFloaterRect(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.minimized = source["minimized"];
	        this.zIndex = source["zIndex"];
	    }
	}
	export class SavedPanelConfig {
	    mode: string;
	    dockZone: string;
	    targetWorkspace: string;
	    floaterRect: SavedFloaterRect;
	
	    static createFrom(source: any = {}) {
	        return new SavedPanelConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.dockZone = source["dockZone"];
	        this.targetWorkspace = source["targetWorkspace"];
	        this.floaterRect = this.convertValues(source["floaterRect"], SavedFloaterRect);
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
	export class AppPreferences {
	    defaultDatabasePath: string;
	    referenceDatabasePath: string;
	    autoOpenDefaultDb: boolean;
	    useReferenceDbForTree: boolean;
	    defaultPieceSet: string;
	    defaultBoardTheme: string;
	    defaultUITheme: string;
	    defaultNotationStyle: string;
	    soundEnabled: boolean;
	    showEvalBar: boolean;
	    threatModeDefault: boolean;
	    blindfoldDefault: boolean;
	    boardFlippedDefault: boolean;
	    compactLayoutMode: string;
	    leftColWidth: number;
	    bottomDrawerHeight: number;
	    bottomDrawerOpen: boolean;
	    mainWindowWorkspace: string;
	    autoRestoreWorkspaceWindows: boolean;
	    panelConfigs: Record<string, SavedPanelConfig>;
	    defaultEngine: string;
	    defaultMultiPv: number;
	    defaultThreads: number;
	    defaultHashMb: number;
	    annotateMoveTimeMs: number;
	    lichessUsername: string;
	    chessComUsername: string;
	    autoCapturePortalGames: boolean;
	    autoLoadCapturedGame: boolean;
	    defaultPortalPlatform: string;
	    portalWindowMode: string;
	    customBrowserCmd: string;
	    databasePolicies?: Record<string, string>;
	    customNotablePlayers?: scid.FamousPlayerEntry[];
	    hiddenFamousPlayers?: string[];
	
	    static createFrom(source: any = {}) {
	        return new AppPreferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaultDatabasePath = source["defaultDatabasePath"];
	        this.referenceDatabasePath = source["referenceDatabasePath"];
	        this.autoOpenDefaultDb = source["autoOpenDefaultDb"];
	        this.useReferenceDbForTree = source["useReferenceDbForTree"];
	        this.defaultPieceSet = source["defaultPieceSet"];
	        this.defaultBoardTheme = source["defaultBoardTheme"];
	        this.defaultUITheme = source["defaultUITheme"];
	        this.defaultNotationStyle = source["defaultNotationStyle"];
	        this.soundEnabled = source["soundEnabled"];
	        this.showEvalBar = source["showEvalBar"];
	        this.threatModeDefault = source["threatModeDefault"];
	        this.blindfoldDefault = source["blindfoldDefault"];
	        this.boardFlippedDefault = source["boardFlippedDefault"];
	        this.compactLayoutMode = source["compactLayoutMode"];
	        this.leftColWidth = source["leftColWidth"];
	        this.bottomDrawerHeight = source["bottomDrawerHeight"];
	        this.bottomDrawerOpen = source["bottomDrawerOpen"];
	        this.mainWindowWorkspace = source["mainWindowWorkspace"];
	        this.autoRestoreWorkspaceWindows = source["autoRestoreWorkspaceWindows"];
	        this.panelConfigs = this.convertValues(source["panelConfigs"], SavedPanelConfig, true);
	        this.defaultEngine = source["defaultEngine"];
	        this.defaultMultiPv = source["defaultMultiPv"];
	        this.defaultThreads = source["defaultThreads"];
	        this.defaultHashMb = source["defaultHashMb"];
	        this.annotateMoveTimeMs = source["annotateMoveTimeMs"];
	        this.lichessUsername = source["lichessUsername"];
	        this.chessComUsername = source["chessComUsername"];
	        this.autoCapturePortalGames = source["autoCapturePortalGames"];
	        this.autoLoadCapturedGame = source["autoLoadCapturedGame"];
	        this.defaultPortalPlatform = source["defaultPortalPlatform"];
	        this.portalWindowMode = source["portalWindowMode"];
	        this.customBrowserCmd = source["customBrowserCmd"];
	        this.databasePolicies = source["databasePolicies"];
	        this.customNotablePlayers = this.convertValues(source["customNotablePlayers"], scid.FamousPlayerEntry);
	        this.hiddenFamousPlayers = source["hiddenFamousPlayers"];
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
	export class BoardState {
	    fen: string;
	    turn: string;
	    inCheck: boolean;
	    isCheckmate: boolean;
	    isStalemate: boolean;
	    legalMoves: string[];
	    fullMove: number;
	    halfMove: number;
	    moveUci?: string;
	    moveSan?: string;
	
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
	        this.moveUci = source["moveUci"];
	        this.moveSan = source["moveSan"];
	    }
	}
	export class DatabaseInfo {
	    basePath: string;
	    name: string;
	    numGames: number;
	    description: string;
	    writePolicy: string;
	    isLocked: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.basePath = source["basePath"];
	        this.name = source["name"];
	        this.numGames = source["numGames"];
	        this.description = source["description"];
	        this.writePolicy = source["writePolicy"];
	        this.isLocked = source["isLocked"];
	    }
	}
	export class DatabaseRolesStatus {
	    activeDb?: DatabaseInfo;
	    defaultDbPath: string;
	    defaultDbName: string;
	    isActiveDefault: boolean;
	    referenceDb?: DatabaseInfo;
	    referenceDbPath: string;
	    referenceDbName: string;
	    useRefForTree: boolean;
	    clipbasePath: string;
	    isClipbase: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseRolesStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.activeDb = this.convertValues(source["activeDb"], DatabaseInfo);
	        this.defaultDbPath = source["defaultDbPath"];
	        this.defaultDbName = source["defaultDbName"];
	        this.isActiveDefault = source["isActiveDefault"];
	        this.referenceDb = this.convertValues(source["referenceDb"], DatabaseInfo);
	        this.referenceDbPath = source["referenceDbPath"];
	        this.referenceDbName = source["referenceDbName"];
	        this.useRefForTree = source["useRefForTree"];
	        this.clipbasePath = source["clipbasePath"];
	        this.isClipbase = source["isClipbase"];
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
	export class LivePortalCaptureStatus {
	    enabled: boolean;
	    autoLoadToBoard: boolean;
	    watcherActive: boolean;
	    intervalSeconds: number;
	    lichessUsername: string;
	    chessComUsername: string;
	    targetDatabasePath: string;
	    targetDatabaseName: string;
	    lastCheckAtMs: number;
	    lastCapturedAtMs: number;
	    lastCapturedGameId: number;
	    lastCapturedGameSummary: string;
	    sessionCapturedCount: number;
	    lastError?: string;
	
	    static createFrom(source: any = {}) {
	        return new LivePortalCaptureStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.autoLoadToBoard = source["autoLoadToBoard"];
	        this.watcherActive = source["watcherActive"];
	        this.intervalSeconds = source["intervalSeconds"];
	        this.lichessUsername = source["lichessUsername"];
	        this.chessComUsername = source["chessComUsername"];
	        this.targetDatabasePath = source["targetDatabasePath"];
	        this.targetDatabaseName = source["targetDatabaseName"];
	        this.lastCheckAtMs = source["lastCheckAtMs"];
	        this.lastCapturedAtMs = source["lastCapturedAtMs"];
	        this.lastCapturedGameId = source["lastCapturedGameId"];
	        this.lastCapturedGameSummary = source["lastCapturedGameSummary"];
	        this.sessionCapturedCount = source["sessionCapturedCount"];
	        this.lastError = source["lastError"];
	    }
	}
	export class LivePortalCaptureResult {
	    checkedAtMs: number;
	    newGamesImported: number;
	    lichessImported: number;
	    chessComImported: number;
	    duplicatesSkipped: number;
	    latestGameId: number;
	    latestGameSummary: string;
	    latestGamePgn?: string;
	    targetDatabasePath: string;
	    targetDatabaseName: string;
	    autoLoadToBoard: boolean;
	    status?: LivePortalCaptureStatus;
	
	    static createFrom(source: any = {}) {
	        return new LivePortalCaptureResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.checkedAtMs = source["checkedAtMs"];
	        this.newGamesImported = source["newGamesImported"];
	        this.lichessImported = source["lichessImported"];
	        this.chessComImported = source["chessComImported"];
	        this.duplicatesSkipped = source["duplicatesSkipped"];
	        this.latestGameId = source["latestGameId"];
	        this.latestGameSummary = source["latestGameSummary"];
	        this.latestGamePgn = source["latestGamePgn"];
	        this.targetDatabasePath = source["targetDatabasePath"];
	        this.targetDatabaseName = source["targetDatabaseName"];
	        this.autoLoadToBoard = source["autoLoadToBoard"];
	        this.status = this.convertValues(source["status"], LivePortalCaptureStatus);
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
	
	
	export class SharedWorkspaceState {
	    updatedAtMs: number;
	    sourceId: string;
	    fen: string;
	    lastMoveFrom: string;
	    lastMoveTo: string;
	    currentPly: number;
	    dbPath: string;
	    dbName: string;
	    activeGameId: number;
	    gameTags: Record<string, string>;
	    pendingMoveUci?: string;
	    pendingLoadId?: number;
	
	    static createFrom(source: any = {}) {
	        return new SharedWorkspaceState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.updatedAtMs = source["updatedAtMs"];
	        this.sourceId = source["sourceId"];
	        this.fen = source["fen"];
	        this.lastMoveFrom = source["lastMoveFrom"];
	        this.lastMoveTo = source["lastMoveTo"];
	        this.currentPly = source["currentPly"];
	        this.dbPath = source["dbPath"];
	        this.dbName = source["dbName"];
	        this.activeGameId = source["activeGameId"];
	        this.gameTags = source["gameTags"];
	        this.pendingMoveUci = source["pendingMoveUci"];
	        this.pendingLoadId = source["pendingLoadId"];
	    }
	}
	export class StandardImportRequest {
	    sourcePath: string;
	    pgnText: string;
	    archiveName: string;
	    importMode: string;
	    masterDbName: string;
	    userDbName: string;
	    targetDir?: string;
	    setMasterAsRef: boolean;
	    setUserAsDefault: boolean;
	    openAfterImport: boolean;
	
	    static createFrom(source: any = {}) {
	        return new StandardImportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourcePath = source["sourcePath"];
	        this.pgnText = source["pgnText"];
	        this.archiveName = source["archiveName"];
	        this.importMode = source["importMode"];
	        this.masterDbName = source["masterDbName"];
	        this.userDbName = source["userDbName"];
	        this.targetDir = source["targetDir"];
	        this.setMasterAsRef = source["setMasterAsRef"];
	        this.setUserAsDefault = source["setUserAsDefault"];
	        this.openAfterImport = source["openAfterImport"];
	    }
	}
	export class StandardImportResult {
	    importMode: string;
	    originalBasePath?: string;
	    originalName?: string;
	    originalGames: number;
	    masterBasePath?: string;
	    masterName?: string;
	    masterTotalGames: number;
	    masterAddedGames: number;
	    masterSkippedDupes: number;
	    userBasePath?: string;
	    userName?: string;
	    userTotalGames: number;
	    userAddedGames: number;
	    userSkippedDupes: number;
	    activeBasePath?: string;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new StandardImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.importMode = source["importMode"];
	        this.originalBasePath = source["originalBasePath"];
	        this.originalName = source["originalName"];
	        this.originalGames = source["originalGames"];
	        this.masterBasePath = source["masterBasePath"];
	        this.masterName = source["masterName"];
	        this.masterTotalGames = source["masterTotalGames"];
	        this.masterAddedGames = source["masterAddedGames"];
	        this.masterSkippedDupes = source["masterSkippedDupes"];
	        this.userBasePath = source["userBasePath"];
	        this.userName = source["userName"];
	        this.userTotalGames = source["userTotalGames"];
	        this.userAddedGames = source["userAddedGames"];
	        this.userSkippedDupes = source["userSkippedDupes"];
	        this.activeBasePath = source["activeBasePath"];
	        this.summary = source["summary"];
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
	export class WindowStartupConfig {
	    mode: string;
	    panelId: string;
	    workspace: string;
	    portalUrl: string;
	    portalTitle: string;
	    initialFen: string;
	    initialDb: string;
	
	    static createFrom(source: any = {}) {
	        return new WindowStartupConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.panelId = source["panelId"];
	        this.workspace = source["workspace"];
	        this.portalUrl = source["portalUrl"];
	        this.portalTitle = source["portalTitle"];
	        this.initialFen = source["initialFen"];
	        this.initialDb = source["initialDb"];
	    }
	}

}

export namespace online {
	
	export class ImportPlayerOptions {
	    playerName: string;
	    chessComUsername: string;
	    lichessUsername: string;
	    lichessToken?: string;
	    targetMode: string;
	    dedicatedPath: string;
	    blendedPath: string;
	    ratedOnly: boolean;
	    maxGames: number;
	    autoDeduplicate: boolean;
	    createProfile: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ImportPlayerOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.playerName = source["playerName"];
	        this.chessComUsername = source["chessComUsername"];
	        this.lichessUsername = source["lichessUsername"];
	        this.lichessToken = source["lichessToken"];
	        this.targetMode = source["targetMode"];
	        this.dedicatedPath = source["dedicatedPath"];
	        this.blendedPath = source["blendedPath"];
	        this.ratedOnly = source["ratedOnly"];
	        this.maxGames = source["maxGames"];
	        this.autoDeduplicate = source["autoDeduplicate"];
	        this.createProfile = source["createProfile"];
	    }
	}
	export class MegaDatabaseSource {
	    id: string;
	    name: string;
	    provider: string;
	    category: string;
	    format: string;
	    url: string;
	    sizeEstimate: string;
	    gamesEstimate: string;
	    description: string;
	    supportsLimit: boolean;
	    year: number;
	    isLatest: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MegaDatabaseSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.category = source["category"];
	        this.format = source["format"];
	        this.url = source["url"];
	        this.sizeEstimate = source["sizeEstimate"];
	        this.gamesEstimate = source["gamesEstimate"];
	        this.description = source["description"];
	        this.supportsLimit = source["supportsLimit"];
	        this.year = source["year"];
	        this.isLatest = source["isLatest"];
	    }
	}
	export class MegaImportOptions {
	    sourceId: string;
	    customUrl?: string;
	    maxGames: number;
	    minElo: number;
	    createNewDatabase: string;
	
	    static createFrom(source: any = {}) {
	        return new MegaImportOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceId = source["sourceId"];
	        this.customUrl = source["customUrl"];
	        this.maxGames = source["maxGames"];
	        this.minElo = source["minElo"];
	        this.createNewDatabase = source["createNewDatabase"];
	    }
	}
	export class MegaImportResult {
	    sourceId: string;
	    sourceName: string;
	    url: string;
	    gamesScanned: number;
	    gamesImported: number;
	    bytesDownloaded: number;
	    databasePath?: string;
	    durationMs: number;
	    cancelled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MegaImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceId = source["sourceId"];
	        this.sourceName = source["sourceName"];
	        this.url = source["url"];
	        this.gamesScanned = source["gamesScanned"];
	        this.gamesImported = source["gamesImported"];
	        this.bytesDownloaded = source["bytesDownloaded"];
	        this.databasePath = source["databasePath"];
	        this.durationMs = source["durationMs"];
	        this.cancelled = source["cancelled"];
	    }
	}
	export class Profile {
	    id: string;
	    name: string;
	    targetDatabase: string;
	    targetMode: string;
	    blendedDatabase: string;
	    isStudyTarget: boolean;
	    chessComUsername: string;
	    lichessUsername: string;
	    lichessToken?: string;
	    lastSyncTime: number;
	    lastSyncGames: number;
	    ratedOnly: boolean;
	    autoDeduplicate: boolean;
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.targetDatabase = source["targetDatabase"];
	        this.targetMode = source["targetMode"];
	        this.blendedDatabase = source["blendedDatabase"];
	        this.isStudyTarget = source["isStudyTarget"];
	        this.chessComUsername = source["chessComUsername"];
	        this.lichessUsername = source["lichessUsername"];
	        this.lichessToken = source["lichessToken"];
	        this.lastSyncTime = source["lastSyncTime"];
	        this.lastSyncGames = source["lastSyncGames"];
	        this.ratedOnly = source["ratedOnly"];
	        this.autoDeduplicate = source["autoDeduplicate"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class SyncResult {
	    profileId: string;
	    profileName: string;
	    totalDownloaded: number;
	    lichessGames: number;
	    chessComGames: number;
	    imported: number;
	    duplicatesSkipped: number;
	    durationMs: number;
	    timestamp: number;
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new SyncResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.profileName = source["profileName"];
	        this.totalDownloaded = source["totalDownloaded"];
	        this.lichessGames = source["lichessGames"];
	        this.chessComGames = source["chessComGames"];
	        this.imported = source["imported"];
	        this.duplicatesSkipped = source["duplicatesSkipped"];
	        this.durationMs = source["durationMs"];
	        this.timestamp = source["timestamp"];
	        this.errors = source["errors"];
	    }
	}
	export class TWICIssueResult {
	    issueNumber: number;
	    url: string;
	    gamesDownloaded: number;
	    gamesImported: number;
	    durationMs: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new TWICIssueResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.issueNumber = source["issueNumber"];
	        this.url = source["url"];
	        this.gamesDownloaded = source["gamesDownloaded"];
	        this.gamesImported = source["gamesImported"];
	        this.durationMs = source["durationMs"];
	        this.error = source["error"];
	    }
	}
	export class TWICBatchResult {
	    startIssue: number;
	    endIssue: number;
	    totalDownloaded: number;
	    totalImported: number;
	    durationMs: number;
	    issues: TWICIssueResult[];
	
	    static createFrom(source: any = {}) {
	        return new TWICBatchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.startIssue = source["startIssue"];
	        this.endIssue = source["endIssue"];
	        this.totalDownloaded = source["totalDownloaded"];
	        this.totalImported = source["totalImported"];
	        this.durationMs = source["durationMs"];
	        this.issues = this.convertValues(source["issues"], TWICIssueResult);
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

export namespace persona {
	
	export class AggregateResult {
	    personaId: string;
	    displayName: string;
	    dedicatedDatabasePath: string;
	    totalFound: number;
	    newlyAdded: number;
	    duplicatesSkipped: number;
	    totalPersonaGames: number;
	
	    static createFrom(source: any = {}) {
	        return new AggregateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.personaId = source["personaId"];
	        this.displayName = source["displayName"];
	        this.dedicatedDatabasePath = source["dedicatedDatabasePath"];
	        this.totalFound = source["totalFound"];
	        this.newlyAdded = source["newlyAdded"];
	        this.duplicatesSkipped = source["duplicatesSkipped"];
	        this.totalPersonaGames = source["totalPersonaGames"];
	    }
	}
	export class VulnerabilityNode {
	    ply: number;
	    eco: string;
	    openingName: string;
	    moveSan: string;
	    moveUci: string;
	    moveHistorySan: string[];
	    moveHistoryUci?: string[];
	    fen: string;
	    personaRecord: GameRecord;
	    exploitationScore: number;
	    tacticalInsight: string;
	    suggestedReplies: string[];
	
	    static createFrom(source: any = {}) {
	        return new VulnerabilityNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ply = source["ply"];
	        this.eco = source["eco"];
	        this.openingName = source["openingName"];
	        this.moveSan = source["moveSan"];
	        this.moveUci = source["moveUci"];
	        this.moveHistorySan = source["moveHistorySan"];
	        this.moveHistoryUci = source["moveHistoryUci"];
	        this.fen = source["fen"];
	        this.personaRecord = this.convertValues(source["personaRecord"], GameRecord);
	        this.exploitationScore = source["exploitationScore"];
	        this.tacticalInsight = source["tacticalInsight"];
	        this.suggestedReplies = source["suggestedReplies"];
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
	export class GameRecord {
	    total: number;
	    wins: number;
	    draws: number;
	    losses: number;
	    winRate: number;
	
	    static createFrom(source: any = {}) {
	        return new GameRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.wins = source["wins"];
	        this.draws = source["draws"];
	        this.losses = source["losses"];
	        this.winRate = source["winRate"];
	    }
	}
	export class AntiPersonaReport {
	    personaId: string;
	    displayName: string;
	    targetColor: string;
	    totalGamesAnalyzed: number;
	    personaRecord: GameRecord;
	    vulnerabilities: VulnerabilityNode[];
	    topRefutationLines: string[];
	    pgnRepertoire: string;
	    polyglotEntryCount: number;
	    generatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new AntiPersonaReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.personaId = source["personaId"];
	        this.displayName = source["displayName"];
	        this.targetColor = source["targetColor"];
	        this.totalGamesAnalyzed = source["totalGamesAnalyzed"];
	        this.personaRecord = this.convertValues(source["personaRecord"], GameRecord);
	        this.vulnerabilities = this.convertValues(source["vulnerabilities"], VulnerabilityNode);
	        this.topRefutationLines = source["topRefutationLines"];
	        this.pgnRepertoire = source["pgnRepertoire"];
	        this.polyglotEntryCount = source["polyglotEntryCount"];
	        this.generatedAt = source["generatedAt"];
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
	
	export class IdentifierStat {
	    identifier: string;
	    gameCount: number;
	    wins: number;
	    draws: number;
	    losses: number;
	    winRate: number;
	
	    static createFrom(source: any = {}) {
	        return new IdentifierStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.identifier = source["identifier"];
	        this.gameCount = source["gameCount"];
	        this.wins = source["wins"];
	        this.draws = source["draws"];
	        this.losses = source["losses"];
	        this.winRate = source["winRate"];
	    }
	}
	export class Persona {
	    id: string;
	    displayName: string;
	    identifiers: string[];
	    databasePath: string;
	    title?: string;
	    bio?: string;
	    avatarColor?: string;
	    estimatedElo?: number;
	    chessComUsername?: string;
	    lichessUsername?: string;
	    fideId?: string;
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Persona(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.displayName = source["displayName"];
	        this.identifiers = source["identifiers"];
	        this.databasePath = source["databasePath"];
	        this.title = source["title"];
	        this.bio = source["bio"];
	        this.avatarColor = source["avatarColor"];
	        this.estimatedElo = source["estimatedElo"];
	        this.chessComUsername = source["chessComUsername"];
	        this.lichessUsername = source["lichessUsername"];
	        this.fideId = source["fideId"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class PersonaStatistics {
	    personaId: string;
	    displayName: string;
	    totalGames: number;
	    whiteGames: number;
	    blackGames: number;
	    wins: number;
	    draws: number;
	    losses: number;
	    winRate: number;
	    peakElo: number;
	    averageElo: number;
	    topWhiteECOs: string[];
	    topBlackECOs: string[];
	    identifierStats: IdentifierStat[];
	    earliestDate: string;
	    latestDate: string;
	
	    static createFrom(source: any = {}) {
	        return new PersonaStatistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.personaId = source["personaId"];
	        this.displayName = source["displayName"];
	        this.totalGames = source["totalGames"];
	        this.whiteGames = source["whiteGames"];
	        this.blackGames = source["blackGames"];
	        this.wins = source["wins"];
	        this.draws = source["draws"];
	        this.losses = source["losses"];
	        this.winRate = source["winRate"];
	        this.peakElo = source["peakElo"];
	        this.averageElo = source["averageElo"];
	        this.topWhiteECOs = source["topWhiteECOs"];
	        this.topBlackECOs = source["topBlackECOs"];
	        this.identifierStats = this.convertValues(source["identifierStats"], IdentifierStat);
	        this.earliestDate = source["earliestDate"];
	        this.latestDate = source["latestDate"];
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

export namespace playerbot {
	
	export class BookMove {
	    uci: string;
	    san: string;
	    playedCount: number;
	    wins: number;
	    draws: number;
	    losses: number;
	    winRate: number;
	
	    static createFrom(source: any = {}) {
	        return new BookMove(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uci = source["uci"];
	        this.san = source["san"];
	        this.playedCount = source["playedCount"];
	        this.wins = source["wins"];
	        this.draws = source["draws"];
	        this.losses = source["losses"];
	        this.winRate = source["winRate"];
	    }
	}
	export class PlayerPersonality {
	    name: string;
	    totalGames: number;
	    whiteGames: number;
	    blackGames: number;
	    wins: number;
	    draws: number;
	    losses: number;
	    winRate: number;
	    estimatedElo: number;
	    peakElo: number;
	    topWhiteECOs: string[];
	    topBlackECOs: string[];
	    whiteBook: Record<string, Array<BookMove>>;
	    blackBook: Record<string, Array<BookMove>>;
	    avgGamePlies: number;
	    aggressionRating: number;
	    contemptValue: number;
	    playStyle: string;
	
	    static createFrom(source: any = {}) {
	        return new PlayerPersonality(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.totalGames = source["totalGames"];
	        this.whiteGames = source["whiteGames"];
	        this.blackGames = source["blackGames"];
	        this.wins = source["wins"];
	        this.draws = source["draws"];
	        this.losses = source["losses"];
	        this.winRate = source["winRate"];
	        this.estimatedElo = source["estimatedElo"];
	        this.peakElo = source["peakElo"];
	        this.topWhiteECOs = source["topWhiteECOs"];
	        this.topBlackECOs = source["topBlackECOs"];
	        this.whiteBook = this.convertValues(source["whiteBook"], Array<BookMove>, true);
	        this.blackBook = this.convertValues(source["blackBook"], Array<BookMove>, true);
	        this.avgGamePlies = source["avgGamePlies"];
	        this.aggressionRating = source["aggressionRating"];
	        this.contemptValue = source["contemptValue"];
	        this.playStyle = source["playStyle"];
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
	export class SimulatedMove {
	    uci: string;
	    san: string;
	    isBook: boolean;
	    playedCount: number;
	    winRate: number;
	    eloTarget: number;
	    commentary: string;
	
	    static createFrom(source: any = {}) {
	        return new SimulatedMove(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uci = source["uci"];
	        this.san = source["san"];
	        this.isBook = source["isBook"];
	        this.playedCount = source["playedCount"];
	        this.winRate = source["winRate"];
	        this.eloTarget = source["eloTarget"];
	        this.commentary = source["commentary"];
	    }
	}

}

export namespace polyglot {
	
	export class BookMove {
	    uci: string;
	    san: string;
	    weight: number;
	    percentage: number;
	    learn: number;
	
	    static createFrom(source: any = {}) {
	        return new BookMove(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uci = source["uci"];
	        this.san = source["san"];
	        this.weight = source["weight"];
	        this.percentage = source["percentage"];
	        this.learn = source["learn"];
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
	
	export class HighYieldLineSummary {
	    sanLine: string;
	    coveragePct: number;
	    movesCount: number;
	    keyIdea?: string;
	
	    static createFrom(source: any = {}) {
	        return new HighYieldLineSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sanLine = source["sanLine"];
	        this.coveragePct = source["coveragePct"];
	        this.movesCount = source["movesCount"];
	        this.keyIdea = source["keyIdea"];
	    }
	}
	export class PrunedBranchSummary {
	    sanLine: string;
	    depth: number;
	    frequencyPct: number;
	    totalGames: number;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new PrunedBranchSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sanLine = source["sanLine"];
	        this.depth = source["depth"];
	        this.frequencyPct = source["frequencyPct"];
	        this.totalGames = source["totalGames"];
	        this.reason = source["reason"];
	    }
	}
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
	export class ShortenerOptions {
	    repertoireId: string;
	    targetCoveragePct: number;
	    maxDepth: number;
	    minFrequencyPct: number;
	    preserveAnnotated: boolean;
	    saveAsNew: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ShortenerOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repertoireId = source["repertoireId"];
	        this.targetCoveragePct = source["targetCoveragePct"];
	        this.maxDepth = source["maxDepth"];
	        this.minFrequencyPct = source["minFrequencyPct"];
	        this.preserveAnnotated = source["preserveAnnotated"];
	        this.saveAsNew = source["saveAsNew"];
	    }
	}
	export class ShortenerResult {
	    originalId: string;
	    prunedRepertoireId: string;
	    originalMoveCount: number;
	    prunedMoveCount: number;
	    movesRemoved: number;
	    reductionPct: number;
	    coverageRetainedPct: number;
	    prunedBranches: PrunedBranchSummary[];
	    highYieldLines: HighYieldLineSummary[];
	    prunedRepertoire?: Repertoire;
	
	    static createFrom(source: any = {}) {
	        return new ShortenerResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.originalId = source["originalId"];
	        this.prunedRepertoireId = source["prunedRepertoireId"];
	        this.originalMoveCount = source["originalMoveCount"];
	        this.prunedMoveCount = source["prunedMoveCount"];
	        this.movesRemoved = source["movesRemoved"];
	        this.reductionPct = source["reductionPct"];
	        this.coverageRetainedPct = source["coverageRetainedPct"];
	        this.prunedBranches = this.convertValues(source["prunedBranches"], PrunedBranchSummary);
	        this.highYieldLines = this.convertValues(source["highYieldLines"], HighYieldLineSummary);
	        this.prunedRepertoire = this.convertValues(source["prunedRepertoire"], Repertoire);
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
	export class ConsolidateDatabaseOptions {
	    targetName: string;
	    targetDescription: string;
	    targetBasePath?: string;
	    sourceBasePaths: string[];
	    setAsReference: boolean;
	    setAsDefaultUser: boolean;
	    minElo?: number;
	    onlyMostRecentArchives: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConsolidateDatabaseOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetName = source["targetName"];
	        this.targetDescription = source["targetDescription"];
	        this.targetBasePath = source["targetBasePath"];
	        this.sourceBasePaths = source["sourceBasePaths"];
	        this.setAsReference = source["setAsReference"];
	        this.setAsDefaultUser = source["setAsDefaultUser"];
	        this.minElo = source["minElo"];
	        this.onlyMostRecentArchives = source["onlyMostRecentArchives"];
	    }
	}
	export class ConsolidateDatabaseResult {
	    targetBasePath: string;
	    targetName: string;
	    sourcesScanned: number;
	    totalGamesScanned: number;
	    uniqueGamesSaved: number;
	    duplicatesRemoved: number;
	    duplicatesById: number;
	    duplicatesByContent: number;
	    cancelled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConsolidateDatabaseResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetBasePath = source["targetBasePath"];
	        this.targetName = source["targetName"];
	        this.sourcesScanned = source["sourcesScanned"];
	        this.totalGamesScanned = source["totalGamesScanned"];
	        this.uniqueGamesSaved = source["uniqueGamesSaved"];
	        this.duplicatesRemoved = source["duplicatesRemoved"];
	        this.duplicatesById = source["duplicatesById"];
	        this.duplicatesByContent = source["duplicatesByContent"];
	        this.cancelled = source["cancelled"];
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
	    writePolicy: string;
	    isLocked: boolean;
	
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
	        this.writePolicy = source["writePolicy"];
	        this.isLocked = source["isLocked"];
	    }
	}
	export class DiscoveredNotablePlayer {
	    name: string;
	    title: string;
	    era: string;
	    aliases: string[];
	    matchedNames: string[];
	    gameCount: number;
	    peakElo: number;
	    yearStart: number;
	    yearEnd: number;
	    isCustom: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DiscoveredNotablePlayer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.era = source["era"];
	        this.aliases = source["aliases"];
	        this.matchedNames = source["matchedNames"];
	        this.gameCount = source["gameCount"];
	        this.peakElo = source["peakElo"];
	        this.yearStart = source["yearStart"];
	        this.yearEnd = source["yearEnd"];
	        this.isCustom = source["isCustom"];
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
	export class FamousPlayerEntry {
	    name: string;
	    title: string;
	    era: string;
	    aliases: string[];
	    isCustom: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FamousPlayerEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.era = source["era"];
	        this.aliases = source["aliases"];
	        this.isCustom = source["isCustom"];
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
	export class MaterialCriteria {
	    whiteQ: number;
	    whiteR: number;
	    whiteB: number;
	    whiteN: number;
	    whiteP: number;
	    blackQ: number;
	    blackR: number;
	    blackB: number;
	    blackN: number;
	    blackP: number;
	    diffQueens: number;
	    diffRooks: number;
	    diffBishops: number;
	    diffKnights: number;
	    diffPawns: number;
	    oppositeBishops: boolean;
	    matchEndOnly: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MaterialCriteria(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.whiteQ = source["whiteQ"];
	        this.whiteR = source["whiteR"];
	        this.whiteB = source["whiteB"];
	        this.whiteN = source["whiteN"];
	        this.whiteP = source["whiteP"];
	        this.blackQ = source["blackQ"];
	        this.blackR = source["blackR"];
	        this.blackB = source["blackB"];
	        this.blackN = source["blackN"];
	        this.blackP = source["blackP"];
	        this.diffQueens = source["diffQueens"];
	        this.diffRooks = source["diffRooks"];
	        this.diffBishops = source["diffBishops"];
	        this.diffKnights = source["diffKnights"];
	        this.diffPawns = source["diffPawns"];
	        this.oppositeBishops = source["oppositeBishops"];
	        this.matchEndOnly = source["matchEndOnly"];
	    }
	}
	export class MergeStats {
	    sourcesProcessed: number;
	    sourceNames: string[];
	    totalScanned: number;
	    imported: number;
	    duplicatesSkipped: number;
	    errors: number;
	    durationMs: number;
	    targetTotalGames: number;
	    cancelled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MergeStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourcesProcessed = source["sourcesProcessed"];
	        this.sourceNames = source["sourceNames"];
	        this.totalScanned = source["totalScanned"];
	        this.imported = source["imported"];
	        this.duplicatesSkipped = source["duplicatesSkipped"];
	        this.errors = source["errors"];
	        this.durationMs = source["durationMs"];
	        this.targetTotalGames = source["targetTotalGames"];
	        this.cancelled = source["cancelled"];
	    }
	}
	
	export class SearchCriteria {
	    white?: string;
	    black?: string;
	    player?: string;
	    players?: string[];
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
	    material?: MaterialCriteria;
	    pawnStructure?: string;
	    commentText?: string;
	    annotator?: string;
	    positionFen?: string;
	    openingMoves?: string[];
	
	    static createFrom(source: any = {}) {
	        return new SearchCriteria(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.white = source["white"];
	        this.black = source["black"];
	        this.player = source["player"];
	        this.players = source["players"];
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
	        this.material = this.convertValues(source["material"], MaterialCriteria);
	        this.pawnStructure = source["pawnStructure"];
	        this.commentText = source["commentText"];
	        this.annotator = source["annotator"];
	        this.positionFen = source["positionFen"];
	        this.openingMoves = source["openingMoves"];
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
	export class SpellCorrection {
	    original: string;
	    standardized: string;
	    gamesCount: number;
	
	    static createFrom(source: any = {}) {
	        return new SpellCorrection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.original = source["original"];
	        this.standardized = source["standardized"];
	        this.gamesCount = source["gamesCount"];
	    }
	}
	export class SpellCheckSummary {
	    totalPlayersChecked: number;
	    correctionsMade: number;
	    gamesUpdated: number;
	    corrections: SpellCorrection[];
	
	    static createFrom(source: any = {}) {
	        return new SpellCheckSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalPlayersChecked = source["totalPlayersChecked"];
	        this.correctionsMade = source["correctionsMade"];
	        this.gamesUpdated = source["gamesUpdated"];
	        this.corrections = this.convertValues(source["corrections"], SpellCorrection);
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
	
	export class SubDatabaseFilter {
	    targetName: string;
	    targetDescription: string;
	    targetBasePath?: string;
	    sourceBasePaths: string[];
	    players?: string[];
	    playerColor?: string;
	    ecoFilter?: string;
	    openingMoves?: string[];
	    minElo?: number;
	    yearFrom?: number;
	    yearTo?: number;
	    resultFilter?: string;
	    deduplicate: boolean;
	    maxGames?: number;
	
	    static createFrom(source: any = {}) {
	        return new SubDatabaseFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetName = source["targetName"];
	        this.targetDescription = source["targetDescription"];
	        this.targetBasePath = source["targetBasePath"];
	        this.sourceBasePaths = source["sourceBasePaths"];
	        this.players = source["players"];
	        this.playerColor = source["playerColor"];
	        this.ecoFilter = source["ecoFilter"];
	        this.openingMoves = source["openingMoves"];
	        this.minElo = source["minElo"];
	        this.yearFrom = source["yearFrom"];
	        this.yearTo = source["yearTo"];
	        this.resultFilter = source["resultFilter"];
	        this.deduplicate = source["deduplicate"];
	        this.maxGames = source["maxGames"];
	    }
	}
	export class SubDatabaseResult {
	    targetBasePath: string;
	    targetName: string;
	    sourcesScanned: number;
	    totalGamesScanned: number;
	    gamesMatched: number;
	    duplicatesSkipped: number;
	    cancelled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SubDatabaseResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetBasePath = source["targetBasePath"];
	        this.targetName = source["targetName"];
	        this.sourcesScanned = source["sourcesScanned"];
	        this.totalGamesScanned = source["totalGamesScanned"];
	        this.gamesMatched = source["gamesMatched"];
	        this.duplicatesSkipped = source["duplicatesSkipped"];
	        this.cancelled = source["cancelled"];
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

export namespace scout {
	
	export class GameRecord {
	    total: number;
	    wins: number;
	    draws: number;
	    losses: number;
	    winRate: number;
	
	    static createFrom(source: any = {}) {
	        return new GameRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.wins = source["wins"];
	        this.draws = source["draws"];
	        this.losses = source["losses"];
	        this.winRate = source["winRate"];
	    }
	}
	export class LinePerformance {
	    eco: string;
	    name?: string;
	    games: number;
	    wins: number;
	    losses: number;
	    draws: number;
	    winRate: number;
	
	    static createFrom(source: any = {}) {
	        return new LinePerformance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.eco = source["eco"];
	        this.name = source["name"];
	        this.games = source["games"];
	        this.wins = source["wins"];
	        this.losses = source["losses"];
	        this.draws = source["draws"];
	        this.winRate = source["winRate"];
	    }
	}
	export class Report {
	    playerName: string;
	    totalGames: number;
	    whiteRecord: GameRecord;
	    blackRecord: GameRecord;
	    topOpeningsAsWhite: LinePerformance[];
	    topOpeningsAsBlack: LinePerformance[];
	    sharpestLines: LinePerformance[];
	    weakestLines: LinePerformance[];
	    repertoireRecommendations: string[];
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.playerName = source["playerName"];
	        this.totalGames = source["totalGames"];
	        this.whiteRecord = this.convertValues(source["whiteRecord"], GameRecord);
	        this.blackRecord = this.convertValues(source["blackRecord"], GameRecord);
	        this.topOpeningsAsWhite = this.convertValues(source["topOpeningsAsWhite"], LinePerformance);
	        this.topOpeningsAsBlack = this.convertValues(source["topOpeningsAsBlack"], LinePerformance);
	        this.sharpestLines = this.convertValues(source["sharpestLines"], LinePerformance);
	        this.weakestLines = this.convertValues(source["weakestLines"], LinePerformance);
	        this.repertoireRecommendations = source["repertoireRecommendations"];
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

export namespace stats {
	
	export class HistogramBin {
	    label: string;
	    count: number;
	    pct: number;
	
	    static createFrom(source: any = {}) {
	        return new HistogramBin(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.count = source["count"];
	        this.pct = source["pct"];
	    }
	}
	export class DatabaseDemographics {
	    totalGames: number;
	    ratedGames: number;
	    avgRating: number;
	    avgMoves: number;
	    whiteWins: number;
	    draws: number;
	    blackWins: number;
	    whiteWinPct: number;
	    drawPct: number;
	    blackWinPct: number;
	    decadeDist: HistogramBin[];
	    yearDist: HistogramBin[];
	    eloDist: HistogramBin[];
	    ecoSectionDist: HistogramBin[];
	    topEcos: HistogramBin[];
	    moveLengthDist: HistogramBin[];
	
	    static createFrom(source: any = {}) {
	        return new DatabaseDemographics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalGames = source["totalGames"];
	        this.ratedGames = source["ratedGames"];
	        this.avgRating = source["avgRating"];
	        this.avgMoves = source["avgMoves"];
	        this.whiteWins = source["whiteWins"];
	        this.draws = source["draws"];
	        this.blackWins = source["blackWins"];
	        this.whiteWinPct = source["whiteWinPct"];
	        this.drawPct = source["drawPct"];
	        this.blackWinPct = source["blackWinPct"];
	        this.decadeDist = this.convertValues(source["decadeDist"], HistogramBin);
	        this.yearDist = this.convertValues(source["yearDist"], HistogramBin);
	        this.eloDist = this.convertValues(source["eloDist"], HistogramBin);
	        this.ecoSectionDist = this.convertValues(source["ecoSectionDist"], HistogramBin);
	        this.topEcos = this.convertValues(source["topEcos"], HistogramBin);
	        this.moveLengthDist = this.convertValues(source["moveLengthDist"], HistogramBin);
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
	export class ScrambleSummary {
	    whiteEnteredScramblePly: number;
	    blackEnteredScramblePly: number;
	    whiteScrambleMoves: number;
	    blackScrambleMoves: number;
	    whiteScrambleBlunders: number;
	    blackScrambleBlunders: number;
	
	    static createFrom(source: any = {}) {
	        return new ScrambleSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.whiteEnteredScramblePly = source["whiteEnteredScramblePly"];
	        this.blackEnteredScramblePly = source["blackEnteredScramblePly"];
	        this.whiteScrambleMoves = source["whiteScrambleMoves"];
	        this.blackScrambleMoves = source["blackScrambleMoves"];
	        this.whiteScrambleBlunders = source["whiteScrambleBlunders"];
	        this.blackScrambleBlunders = source["blackScrambleBlunders"];
	    }
	}
	export class PanicIncident {
	    ply: number;
	    moveNumber: number;
	    player: string;
	    color: string;
	    san: string;
	    timeSpentSec: number;
	    clockRemainingSec: number;
	    evalDropCp: number;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new PanicIncident(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ply = source["ply"];
	        this.moveNumber = source["moveNumber"];
	        this.player = source["player"];
	        this.color = source["color"];
	        this.san = source["san"];
	        this.timeSpentSec = source["timeSpentSec"];
	        this.clockRemainingSec = source["clockRemainingSec"];
	        this.evalDropCp = source["evalDropCp"];
	        this.description = source["description"];
	    }
	}
	export class TankIncident {
	    ply: number;
	    moveNumber: number;
	    player: string;
	    color: string;
	    san: string;
	    timeSpentSec: number;
	    clockBeforeSec: number;
	    clockAfterSec: number;
	    evalDropCp: number;
	    outcome: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new TankIncident(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ply = source["ply"];
	        this.moveNumber = source["moveNumber"];
	        this.player = source["player"];
	        this.color = source["color"];
	        this.san = source["san"];
	        this.timeSpentSec = source["timeSpentSec"];
	        this.clockBeforeSec = source["clockBeforeSec"];
	        this.clockAfterSec = source["clockAfterSec"];
	        this.evalDropCp = source["evalDropCp"];
	        this.outcome = source["outcome"];
	        this.description = source["description"];
	    }
	}
	export class SideTimeStats {
	    totalMoves: number;
	    totalTimeSpentSec: number;
	    avgMoveTimeSec: number;
	    medianMoveTimeSec: number;
	    openingAvgTimeSec: number;
	    middlegameAvgTimeSec: number;
	    endgameAvgTimeSec: number;
	    longestTankSec: number;
	    longestTankPly: number;
	    tankCount: number;
	    tankBlunderCount: number;
	    panicMoveCount: number;
	    timeTroubleMoves: number;
	    timeTroubleBlunders: number;
	    correlationTimeVsLoss: number;
	
	    static createFrom(source: any = {}) {
	        return new SideTimeStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalMoves = source["totalMoves"];
	        this.totalTimeSpentSec = source["totalTimeSpentSec"];
	        this.avgMoveTimeSec = source["avgMoveTimeSec"];
	        this.medianMoveTimeSec = source["medianMoveTimeSec"];
	        this.openingAvgTimeSec = source["openingAvgTimeSec"];
	        this.middlegameAvgTimeSec = source["middlegameAvgTimeSec"];
	        this.endgameAvgTimeSec = source["endgameAvgTimeSec"];
	        this.longestTankSec = source["longestTankSec"];
	        this.longestTankPly = source["longestTankPly"];
	        this.tankCount = source["tankCount"];
	        this.tankBlunderCount = source["tankBlunderCount"];
	        this.panicMoveCount = source["panicMoveCount"];
	        this.timeTroubleMoves = source["timeTroubleMoves"];
	        this.timeTroubleBlunders = source["timeTroubleBlunders"];
	        this.correlationTimeVsLoss = source["correlationTimeVsLoss"];
	    }
	}
	export class MoveTimeRecord {
	    ply: number;
	    moveNumber: number;
	    turn: string;
	    san: string;
	    uci: string;
	    clockRemainingSec: number;
	    timeSpentSec: number;
	    clockStr: string;
	    phase: string;
	    isTank: boolean;
	    isPanic: boolean;
	    isScramble: boolean;
	    evalCp: number;
	    evalDropCp: number;
	    evaluationClassification: string;
	    flag: string;
	
	    static createFrom(source: any = {}) {
	        return new MoveTimeRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ply = source["ply"];
	        this.moveNumber = source["moveNumber"];
	        this.turn = source["turn"];
	        this.san = source["san"];
	        this.uci = source["uci"];
	        this.clockRemainingSec = source["clockRemainingSec"];
	        this.timeSpentSec = source["timeSpentSec"];
	        this.clockStr = source["clockStr"];
	        this.phase = source["phase"];
	        this.isTank = source["isTank"];
	        this.isPanic = source["isPanic"];
	        this.isScramble = source["isScramble"];
	        this.evalCp = source["evalCp"];
	        this.evalDropCp = source["evalDropCp"];
	        this.evaluationClassification = source["evaluationClassification"];
	        this.flag = source["flag"];
	    }
	}
	export class GameTimeManagementReport {
	    gameId: number;
	    whitePlayer: string;
	    blackPlayer: string;
	    event: string;
	    date: string;
	    timeControl: string;
	    whiteHasClock: boolean;
	    blackHasClock: boolean;
	    moves: MoveTimeRecord[];
	    whiteStats: SideTimeStats;
	    blackStats: SideTimeStats;
	    tankDossier: TankIncident[];
	    panicIncidents: PanicIncident[];
	    scrambleSummary: ScrambleSummary;
	    recommendations: string[];
	
	    static createFrom(source: any = {}) {
	        return new GameTimeManagementReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gameId = source["gameId"];
	        this.whitePlayer = source["whitePlayer"];
	        this.blackPlayer = source["blackPlayer"];
	        this.event = source["event"];
	        this.date = source["date"];
	        this.timeControl = source["timeControl"];
	        this.whiteHasClock = source["whiteHasClock"];
	        this.blackHasClock = source["blackHasClock"];
	        this.moves = this.convertValues(source["moves"], MoveTimeRecord);
	        this.whiteStats = this.convertValues(source["whiteStats"], SideTimeStats);
	        this.blackStats = this.convertValues(source["blackStats"], SideTimeStats);
	        this.tankDossier = this.convertValues(source["tankDossier"], TankIncident);
	        this.panicIncidents = this.convertValues(source["panicIncidents"], PanicIncident);
	        this.scrambleSummary = this.convertValues(source["scrambleSummary"], ScrambleSummary);
	        this.recommendations = source["recommendations"];
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
	
	
	
	export class PlayerTimeManagementDossier {
	    playerName: string;
	    gamesAnalyzed: number;
	    gamesWithClock: number;
	    averageMoveTimeSec: number;
	    openingAvgSec: number;
	    middlegameAvgSec: number;
	    endgameAvgSec: number;
	    totalTanks: number;
	    tankSuccessRate: number;
	    panicMoveCount: number;
	    panicErrorRate: number;
	    scrambleFrequencyPct: number;
	    scrambleBlunderCount: number;
	    habitFlags: string[];
	    coachingDossier: string[];
	
	    static createFrom(source: any = {}) {
	        return new PlayerTimeManagementDossier(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.playerName = source["playerName"];
	        this.gamesAnalyzed = source["gamesAnalyzed"];
	        this.gamesWithClock = source["gamesWithClock"];
	        this.averageMoveTimeSec = source["averageMoveTimeSec"];
	        this.openingAvgSec = source["openingAvgSec"];
	        this.middlegameAvgSec = source["middlegameAvgSec"];
	        this.endgameAvgSec = source["endgameAvgSec"];
	        this.totalTanks = source["totalTanks"];
	        this.tankSuccessRate = source["tankSuccessRate"];
	        this.panicMoveCount = source["panicMoveCount"];
	        this.panicErrorRate = source["panicErrorRate"];
	        this.scrambleFrequencyPct = source["scrambleFrequencyPct"];
	        this.scrambleBlunderCount = source["scrambleBlunderCount"];
	        this.habitFlags = source["habitFlags"];
	        this.coachingDossier = source["coachingDossier"];
	    }
	}
	export class RatingPoint {
	    date: string;
	    year: number;
	    rating: number;
	    opponent: string;
	    opponentElo: number;
	    result: string;
	    event: string;
	    eco: string;
	    gameId: number;
	
	    static createFrom(source: any = {}) {
	        return new RatingPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.year = source["year"];
	        this.rating = source["rating"];
	        this.opponent = source["opponent"];
	        this.opponentElo = source["opponentElo"];
	        this.result = source["result"];
	        this.event = source["event"];
	        this.eco = source["eco"];
	        this.gameId = source["gameId"];
	    }
	}
	export class RatingTrackerResult {
	    player: string;
	    totalGames: number;
	    ratedGames: number;
	    startRating: number;
	    endRating: number;
	    peakRating: number;
	    peakDate: string;
	    minRating: number;
	    maxRating: number;
	    points: RatingPoint[];
	
	    static createFrom(source: any = {}) {
	        return new RatingTrackerResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player = source["player"];
	        this.totalGames = source["totalGames"];
	        this.ratedGames = source["ratedGames"];
	        this.startRating = source["startRating"];
	        this.endRating = source["endRating"];
	        this.peakRating = source["peakRating"];
	        this.peakDate = source["peakDate"];
	        this.minRating = source["minRating"];
	        this.maxRating = source["maxRating"];
	        this.points = this.convertValues(source["points"], RatingPoint);
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

export namespace support {
	
	export class FeatureEntitlement {
	    featureId: string;
	    unlocked: boolean;
	    enforceGate: boolean;
	    supporterActive: boolean;
	    bughouseOnlineSync: boolean;
	    tierRequired: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new FeatureEntitlement(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.featureId = source["featureId"];
	        this.unlocked = source["unlocked"];
	        this.enforceGate = source["enforceGate"];
	        this.supporterActive = source["supporterActive"];
	        this.bughouseOnlineSync = source["bughouseOnlineSync"];
	        this.tierRequired = source["tierRequired"];
	        this.message = source["message"];
	    }
	}
	export class SupportStatus {
	    supported: boolean;
	    isLifetime: boolean;
	    isAuthor: boolean;
	    authorId?: string;
	    deviceSignature?: string;
	    supporterTier?: string;
	    supportedAt?: number;
	    supportedUntil?: number;
	    dismissCount: number;
	    lastDismissedAt?: number;
	    stripeUrl: string;
	    bughouseOnlineDonorToken?: string;
	    tierLinks: Record<number, string>;
	    minAmountUsd: number;
	    shouldShowNag: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SupportStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supported = source["supported"];
	        this.isLifetime = source["isLifetime"];
	        this.isAuthor = source["isAuthor"];
	        this.authorId = source["authorId"];
	        this.deviceSignature = source["deviceSignature"];
	        this.supporterTier = source["supporterTier"];
	        this.supportedAt = source["supportedAt"];
	        this.supportedUntil = source["supportedUntil"];
	        this.dismissCount = source["dismissCount"];
	        this.lastDismissedAt = source["lastDismissedAt"];
	        this.stripeUrl = source["stripeUrl"];
	        this.bughouseOnlineDonorToken = source["bughouseOnlineDonorToken"];
	        this.tierLinks = source["tierLinks"];
	        this.minAmountUsd = source["minAmountUsd"];
	        this.shouldShowNag = source["shouldShowNag"];
	    }
	}

}

export namespace tablebase {
	
	export class TablebaseMove {
	    uci: string;
	    san: string;
	    dtz: number;
	    dtm: number;
	    category: string;
	    zeroing: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TablebaseMove(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uci = source["uci"];
	        this.san = source["san"];
	        this.dtz = source["dtz"];
	        this.dtm = source["dtm"];
	        this.category = source["category"];
	        this.zeroing = source["zeroing"];
	    }
	}
	export class TablebaseResult {
	    fen: string;
	    pieceCount: number;
	    category: string;
	    dtz: number;
	    dtm: number;
	    checkmate: boolean;
	    stalemate: boolean;
	    summaryText: string;
	    moves: TablebaseMove[];
	
	    static createFrom(source: any = {}) {
	        return new TablebaseResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fen = source["fen"];
	        this.pieceCount = source["pieceCount"];
	        this.category = source["category"];
	        this.dtz = source["dtz"];
	        this.dtm = source["dtm"];
	        this.checkmate = source["checkmate"];
	        this.stalemate = source["stalemate"];
	        this.summaryText = source["summaryText"];
	        this.moves = this.convertValues(source["moves"], TablebaseMove);
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

export namespace tournament {
	
	export class CrosstableCell {
	    result: string;
	    score: number;
	    color: string;
	    gameId: number;
	
	    static createFrom(source: any = {}) {
	        return new CrosstableCell(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.result = source["result"];
	        this.score = source["score"];
	        this.color = source["color"];
	        this.gameId = source["gameId"];
	    }
	}
	export class EngineConfig {
	    name: string;
	    path: string;
	    options?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new EngineConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.options = source["options"];
	    }
	}
	export class EngineGameResult {
	    round: number;
	    whiteEngine: string;
	    blackEngine: string;
	    result: string;
	    termination: string;
	    plies: number;
	    movesUci: string[];
	    movesSan: string[];
	    finalFen: string;
	    pgn: string;
	    durationMs: number;
	    eco?: string;
	
	    static createFrom(source: any = {}) {
	        return new EngineGameResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.round = source["round"];
	        this.whiteEngine = source["whiteEngine"];
	        this.blackEngine = source["blackEngine"];
	        this.result = source["result"];
	        this.termination = source["termination"];
	        this.plies = source["plies"];
	        this.movesUci = source["movesUci"];
	        this.movesSan = source["movesSan"];
	        this.finalFen = source["finalFen"];
	        this.pgn = source["pgn"];
	        this.durationMs = source["durationMs"];
	        this.eco = source["eco"];
	    }
	}
	export class EngineMatchConfig {
	    engine1: EngineConfig;
	    engine2: EngineConfig;
	    rounds: number;
	    timeControlMs: number;
	    maxPlies: number;
	    alternateColors: boolean;
	    openingFens?: string[];
	    adjudicateEvalCp?: number;
	    autoSaveToDb: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EngineMatchConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.engine1 = this.convertValues(source["engine1"], EngineConfig);
	        this.engine2 = this.convertValues(source["engine2"], EngineConfig);
	        this.rounds = source["rounds"];
	        this.timeControlMs = source["timeControlMs"];
	        this.maxPlies = source["maxPlies"];
	        this.alternateColors = source["alternateColors"];
	        this.openingFens = source["openingFens"];
	        this.adjudicateEvalCp = source["adjudicateEvalCp"];
	        this.autoSaveToDb = source["autoSaveToDb"];
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
	export class EngineMatchSummary {
	    engine1Name: string;
	    engine2Name: string;
	    rounds: number;
	    engine1Score: number;
	    engine2Score: number;
	    engine1Wins: number;
	    engine2Wins: number;
	    draws: number;
	    games: EngineGameResult[];
	    durationMs: number;
	
	    static createFrom(source: any = {}) {
	        return new EngineMatchSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.engine1Name = source["engine1Name"];
	        this.engine2Name = source["engine2Name"];
	        this.rounds = source["rounds"];
	        this.engine1Score = source["engine1Score"];
	        this.engine2Score = source["engine2Score"];
	        this.engine1Wins = source["engine1Wins"];
	        this.engine2Wins = source["engine2Wins"];
	        this.draws = source["draws"];
	        this.games = this.convertValues(source["games"], EngineGameResult);
	        this.durationMs = source["durationMs"];
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
	export class EventSummary {
	    event: string;
	    site: string;
	    date: string;
	    year: number;
	    numGames: number;
	    numPlayers: number;
	    rounds: number;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new EventSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.event = source["event"];
	        this.site = source["site"];
	        this.date = source["date"];
	        this.year = source["year"];
	        this.numGames = source["numGames"];
	        this.numPlayers = source["numPlayers"];
	        this.rounds = source["rounds"];
	        this.type = source["type"];
	    }
	}
	export class SwissRoundResult {
	    round: string;
	    opponentRank: number;
	    opponentName: string;
	    color: string;
	    result: string;
	    score: number;
	    gameId: number;
	
	    static createFrom(source: any = {}) {
	        return new SwissRoundResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.round = source["round"];
	        this.opponentRank = source["opponentRank"];
	        this.opponentName = source["opponentName"];
	        this.color = source["color"];
	        this.result = source["result"];
	        this.score = source["score"];
	        this.gameId = source["gameId"];
	    }
	}
	export class PlayerStanding {
	    rank: number;
	    playerName: string;
	    title?: string;
	    elo: number;
	    points: number;
	    played: number;
	    wins: number;
	    draws: number;
	    losses: number;
	    sonnebornBerger: number;
	    buchholz: number;
	    buchholzCut1: number;
	    performanceRating: number;
	    roundResults: SwissRoundResult[];
	    scoresAgainst: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new PlayerStanding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rank = source["rank"];
	        this.playerName = source["playerName"];
	        this.title = source["title"];
	        this.elo = source["elo"];
	        this.points = source["points"];
	        this.played = source["played"];
	        this.wins = source["wins"];
	        this.draws = source["draws"];
	        this.losses = source["losses"];
	        this.sonnebornBerger = source["sonnebornBerger"];
	        this.buchholz = source["buchholz"];
	        this.buchholzCut1 = source["buchholzCut1"];
	        this.performanceRating = source["performanceRating"];
	        this.roundResults = this.convertValues(source["roundResults"], SwissRoundResult);
	        this.scoresAgainst = source["scoresAgainst"];
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
	
	export class TournamentCrosstable {
	    event: string;
	    site: string;
	    date: string;
	    category: number;
	    averageElo: number;
	    numPlayers: number;
	    numGames: number;
	    numRounds: number;
	    type: string;
	    standings: PlayerStanding[];
	    matrix: CrosstableCell[][];
	    crossTableText: string;
	
	    static createFrom(source: any = {}) {
	        return new TournamentCrosstable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.event = source["event"];
	        this.site = source["site"];
	        this.date = source["date"];
	        this.category = source["category"];
	        this.averageElo = source["averageElo"];
	        this.numPlayers = source["numPlayers"];
	        this.numGames = source["numGames"];
	        this.numRounds = source["numRounds"];
	        this.type = source["type"];
	        this.standings = this.convertValues(source["standings"], PlayerStanding);
	        this.matrix = this.convertValues(source["matrix"], CrosstableCell);
	        this.crossTableText = source["crossTableText"];
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

export namespace training {
	
	export class PuzzleAttemptResult {
	    puzzleId: string;
	    solvedFirstTry: boolean;
	    timeTakenSec: number;
	    attemptedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new PuzzleAttemptResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.puzzleId = source["puzzleId"];
	        this.solvedFirstTry = source["solvedFirstTry"];
	        this.timeTakenSec = source["timeTakenSec"];
	        this.attemptedAt = source["attemptedAt"];
	    }
	}
	export class ActiveCycleProgress {
	    cycleNumber: number;
	    currentIndex: number;
	    elapsedSec: number;
	    firstTryCorrect: number;
	    totalAttempted: number;
	    results: Record<string, PuzzleAttemptResult>;
	
	    static createFrom(source: any = {}) {
	        return new ActiveCycleProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cycleNumber = source["cycleNumber"];
	        this.currentIndex = source["currentIndex"];
	        this.elapsedSec = source["elapsedSec"];
	        this.firstTryCorrect = source["firstTryCorrect"];
	        this.totalAttempted = source["totalAttempted"];
	        this.results = this.convertValues(source["results"], PuzzleAttemptResult, true);
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
	export class PieceSafety {
	    square: string;
	    piece: string;
	    color: string;
	    pieceValue: number;
	    attackers: string[];
	    defenders: string[];
	    status: string;
	    netLoss: number;
	    explanation: string;
	
	    static createFrom(source: any = {}) {
	        return new PieceSafety(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.square = source["square"];
	        this.piece = source["piece"];
	        this.color = source["color"];
	        this.pieceValue = source["pieceValue"];
	        this.attackers = source["attackers"];
	        this.defenders = source["defenders"];
	        this.status = source["status"];
	        this.netLoss = source["netLoss"];
	        this.explanation = source["explanation"];
	    }
	}
	export class BoardSafetyReport {
	    fen: string;
	    turn: string;
	    pieces: PieceSafety[];
	    hangingPieces: PieceSafety[];
	    underdefendedPieces: PieceSafety[];
	    loosePieces: PieceSafety[];
	    hangingSquares: string[];
	    totalHangingWhite: number;
	    totalHangingBlack: number;
	    criticalThreat?: PieceSafety;
	    blunderCount: number;
	
	    static createFrom(source: any = {}) {
	        return new BoardSafetyReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fen = source["fen"];
	        this.turn = source["turn"];
	        this.pieces = this.convertValues(source["pieces"], PieceSafety);
	        this.hangingPieces = this.convertValues(source["hangingPieces"], PieceSafety);
	        this.underdefendedPieces = this.convertValues(source["underdefendedPieces"], PieceSafety);
	        this.loosePieces = this.convertValues(source["loosePieces"], PieceSafety);
	        this.hangingSquares = source["hangingSquares"];
	        this.totalHangingWhite = source["totalHangingWhite"];
	        this.totalHangingBlack = source["totalHangingBlack"];
	        this.criticalThreat = this.convertValues(source["criticalThreat"], PieceSafety);
	        this.blunderCount = source["blunderCount"];
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
	export class CycleRun {
	    cycleNumber: number;
	    completedAt: string;
	    totalPuzzles: number;
	    firstTryCorrect: number;
	    totalSolved: number;
	    accuracyPct: number;
	    totalTimeSec: number;
	    targetTimeSec: number;
	    avgSecPerPuzzle: number;
	    speedupVsCycle1: number;
	    easyAccuracyPct: number;
	    mediumAccuracyPct: number;
	    hardAccuracyPct: number;
	
	    static createFrom(source: any = {}) {
	        return new CycleRun(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cycleNumber = source["cycleNumber"];
	        this.completedAt = source["completedAt"];
	        this.totalPuzzles = source["totalPuzzles"];
	        this.firstTryCorrect = source["firstTryCorrect"];
	        this.totalSolved = source["totalSolved"];
	        this.accuracyPct = source["accuracyPct"];
	        this.totalTimeSec = source["totalTimeSec"];
	        this.targetTimeSec = source["targetTimeSec"];
	        this.avgSecPerPuzzle = source["avgSecPerPuzzle"];
	        this.speedupVsCycle1 = source["speedupVsCycle1"];
	        this.easyAccuracyPct = source["easyAccuracyPct"];
	        this.mediumAccuracyPct = source["mediumAccuracyPct"];
	        this.hardAccuracyPct = source["hardAccuracyPct"];
	    }
	}
	export class CyclicalPuzzle {
	    id: string;
	    sourceGameId?: number;
	    white: string;
	    black: string;
	    event: string;
	    date: string;
	    ply: number;
	    fen: string;
	    turn: string;
	    solutionUci: string[];
	    solutionSan: string[];
	    playedBlunderUci?: string;
	    themes: string[];
	    primaryTheme: string;
	    difficultyTier: string;
	    difficultyOrder: number;
	    estimatedRating: number;
	    evalDeltaCp: number;
	    explanation: string;
	
	    static createFrom(source: any = {}) {
	        return new CyclicalPuzzle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sourceGameId = source["sourceGameId"];
	        this.white = source["white"];
	        this.black = source["black"];
	        this.event = source["event"];
	        this.date = source["date"];
	        this.ply = source["ply"];
	        this.fen = source["fen"];
	        this.turn = source["turn"];
	        this.solutionUci = source["solutionUci"];
	        this.solutionSan = source["solutionSan"];
	        this.playedBlunderUci = source["playedBlunderUci"];
	        this.themes = source["themes"];
	        this.primaryTheme = source["primaryTheme"];
	        this.difficultyTier = source["difficultyTier"];
	        this.difficultyOrder = source["difficultyOrder"];
	        this.estimatedRating = source["estimatedRating"];
	        this.evalDeltaCp = source["evalDeltaCp"];
	        this.explanation = source["explanation"];
	    }
	}
	export class CycleSet {
	    id: string;
	    name: string;
	    playerName: string;
	    themeFilter: string[];
	    createdAt: number;
	    currentCycle: number;
	    targetDays: number;
	    puzzles: CyclicalPuzzle[];
	    cycles: CycleRun[];
	    activeProgress?: ActiveCycleProgress;
	
	    static createFrom(source: any = {}) {
	        return new CycleSet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.playerName = source["playerName"];
	        this.themeFilter = source["themeFilter"];
	        this.createdAt = source["createdAt"];
	        this.currentCycle = source["currentCycle"];
	        this.targetDays = source["targetDays"];
	        this.puzzles = this.convertValues(source["puzzles"], CyclicalPuzzle);
	        this.cycles = this.convertValues(source["cycles"], CycleRun);
	        this.activeProgress = this.convertValues(source["activeProgress"], ActiveCycleProgress);
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
	export class ThemePerformanceStat {
	    themeKey: string;
	    themeName: string;
	    category: string;
	    attempts: number;
	    firstWins: number;
	    fixed: number;
	    failed: number;
	    accuracyPct: number;
	    avgTimeSec: number;
	    ratingAvg: number;
	    performanceRating: number;
	    historyRatings: number[];
	    totalTimeSec: number;
	    totalRatingSum: number;
	
	    static createFrom(source: any = {}) {
	        return new ThemePerformanceStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.themeKey = source["themeKey"];
	        this.themeName = source["themeName"];
	        this.category = source["category"];
	        this.attempts = source["attempts"];
	        this.firstWins = source["firstWins"];
	        this.fixed = source["fixed"];
	        this.failed = source["failed"];
	        this.accuracyPct = source["accuracyPct"];
	        this.avgTimeSec = source["avgTimeSec"];
	        this.ratingAvg = source["ratingAvg"];
	        this.performanceRating = source["performanceRating"];
	        this.historyRatings = source["historyRatings"];
	        this.totalTimeSec = source["totalTimeSec"];
	        this.totalRatingSum = source["totalRatingSum"];
	    }
	}
	export class CyclicalDashboard {
	    sets: CycleSet[];
	    themeStats: ThemePerformanceStat[];
	    radarAxes: ThemePerformanceStat[];
	    strongThemes: ThemePerformanceStat[];
	    weakThemes: ThemePerformanceStat[];
	    globalPerformanceRating: number;
	    totalPuzzlesAttempted: number;
	    totalCyclesCompleted: number;
	    improvementSummary: string;
	    methodAttribution: string;
	
	    static createFrom(source: any = {}) {
	        return new CyclicalDashboard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sets = this.convertValues(source["sets"], CycleSet);
	        this.themeStats = this.convertValues(source["themeStats"], ThemePerformanceStat);
	        this.radarAxes = this.convertValues(source["radarAxes"], ThemePerformanceStat);
	        this.strongThemes = this.convertValues(source["strongThemes"], ThemePerformanceStat);
	        this.weakThemes = this.convertValues(source["weakThemes"], ThemePerformanceStat);
	        this.globalPerformanceRating = source["globalPerformanceRating"];
	        this.totalPuzzlesAttempted = source["totalPuzzlesAttempted"];
	        this.totalCyclesCompleted = source["totalCyclesCompleted"];
	        this.improvementSummary = source["improvementSummary"];
	        this.methodAttribution = source["methodAttribution"];
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
	
	export class PerformanceSummary {
	    score: number;
	    maxScore: number;
	    percentage: number;
	    accuracyPct: number;
	    correctGuesses: number;
	    totalTurns: number;
	    hintsUsed: number;
	    ratingTitle: string;
	    feedback: string;
	
	    static createFrom(source: any = {}) {
	        return new PerformanceSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.score = source["score"];
	        this.maxScore = source["maxScore"];
	        this.percentage = source["percentage"];
	        this.accuracyPct = source["accuracyPct"];
	        this.correctGuesses = source["correctGuesses"];
	        this.totalTurns = source["totalTurns"];
	        this.hintsUsed = source["hintsUsed"];
	        this.ratingTitle = source["ratingTitle"];
	        this.feedback = source["feedback"];
	    }
	}
	export class GuessFeedback {
	    isMatch: boolean;
	    actualMoveSAN: string;
	    actualMoveUCI: string;
	    pointsEarned: number;
	    totalScore: number;
	    currentFEN: string;
	    opponentReplySAN?: string;
	    opponentReplyUCI?: string;
	    isFinished: boolean;
	    summary?: PerformanceSummary;
	
	    static createFrom(source: any = {}) {
	        return new GuessFeedback(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isMatch = source["isMatch"];
	        this.actualMoveSAN = source["actualMoveSAN"];
	        this.actualMoveUCI = source["actualMoveUCI"];
	        this.pointsEarned = source["pointsEarned"];
	        this.totalScore = source["totalScore"];
	        this.currentFEN = source["currentFEN"];
	        this.opponentReplySAN = source["opponentReplySAN"];
	        this.opponentReplyUCI = source["opponentReplyUCI"];
	        this.isFinished = source["isFinished"];
	        this.summary = this.convertValues(source["summary"], PerformanceSummary);
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
	export class IntuitionChallenge {
	    id: string;
	    fen: string;
	    whitePlayer: string;
	    blackPlayer: string;
	    event: string;
	    evaluationCp: number;
	    category: string;
	    keyFeature: string;
	    materialDelta: number;
	    timeLimitSec: number;
	
	    static createFrom(source: any = {}) {
	        return new IntuitionChallenge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fen = source["fen"];
	        this.whitePlayer = source["whitePlayer"];
	        this.blackPlayer = source["blackPlayer"];
	        this.event = source["event"];
	        this.evaluationCp = source["evaluationCp"];
	        this.category = source["category"];
	        this.keyFeature = source["keyFeature"];
	        this.materialDelta = source["materialDelta"];
	        this.timeLimitSec = source["timeLimitSec"];
	    }
	}
	export class IntuitionResult {
	    isCorrect: boolean;
	    userChoice: string;
	    actualCategory: string;
	    evaluationCp: number;
	    scoreFormatted: string;
	    pointsAwarded: number;
	    speedBonus: number;
	    explanation: string;
	    keyFeature: string;
	
	    static createFrom(source: any = {}) {
	        return new IntuitionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isCorrect = source["isCorrect"];
	        this.userChoice = source["userChoice"];
	        this.actualCategory = source["actualCategory"];
	        this.evaluationCp = source["evaluationCp"];
	        this.scoreFormatted = source["scoreFormatted"];
	        this.pointsAwarded = source["pointsAwarded"];
	        this.speedBonus = source["speedBonus"];
	        this.explanation = source["explanation"];
	        this.keyFeature = source["keyFeature"];
	    }
	}
	export class LichessThemeInfo {
	    id: number;
	    key: string;
	    name: string;
	    category: string;
	    description: string;
	    actionable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LichessThemeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.key = source["key"];
	        this.name = source["name"];
	        this.category = source["category"];
	        this.description = source["description"];
	        this.actionable = source["actionable"];
	    }
	}
	export class MoveRecord {
	    ply: number;
	    turn: string;
	    san: string;
	    uci: string;
	    fen: string;
	    comment?: string;
	
	    static createFrom(source: any = {}) {
	        return new MoveRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ply = source["ply"];
	        this.turn = source["turn"];
	        this.san = source["san"];
	        this.uci = source["uci"];
	        this.fen = source["fen"];
	        this.comment = source["comment"];
	    }
	}
	
	
	export class ProphylaxisDrill {
	    id: string;
	    title: string;
	    fen: string;
	    turn: string;
	    category: string;
	    categoryLabel: string;
	    prompt: string;
	    winningMoveUCI: string;
	    winningMoveSAN: string;
	    alternativeUCIs: string[];
	    opponentThreat: string;
	    threatArrowFrom: string;
	    threatArrowTo: string;
	    explanation: string;
	    source: string;
	    difficulty: string;
	
	    static createFrom(source: any = {}) {
	        return new ProphylaxisDrill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.fen = source["fen"];
	        this.turn = source["turn"];
	        this.category = source["category"];
	        this.categoryLabel = source["categoryLabel"];
	        this.prompt = source["prompt"];
	        this.winningMoveUCI = source["winningMoveUCI"];
	        this.winningMoveSAN = source["winningMoveSAN"];
	        this.alternativeUCIs = source["alternativeUCIs"];
	        this.opponentThreat = source["opponentThreat"];
	        this.threatArrowFrom = source["threatArrowFrom"];
	        this.threatArrowTo = source["threatArrowTo"];
	        this.explanation = source["explanation"];
	        this.source = source["source"];
	        this.difficulty = source["difficulty"];
	    }
	}
	export class ProphylaxisFeedback {
	    isCorrect: boolean;
	    pointsAwarded: number;
	    bestMoveUCI: string;
	    bestMoveSAN: string;
	    explanation: string;
	    opponentThreat: string;
	
	    static createFrom(source: any = {}) {
	        return new ProphylaxisFeedback(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isCorrect = source["isCorrect"];
	        this.pointsAwarded = source["pointsAwarded"];
	        this.bestMoveUCI = source["bestMoveUCI"];
	        this.bestMoveSAN = source["bestMoveSAN"];
	        this.explanation = source["explanation"];
	        this.opponentThreat = source["opponentThreat"];
	    }
	}
	
	export class SafetyDrill {
	    id: string;
	    title: string;
	    fen: string;
	    turn: string;
	    mode: string;
	    prompt: string;
	    targetColor: string;
	    correctSquares: string[];
	    winningMoveUCI?: string;
	    winningMoveSAN?: string;
	    explanation: string;
	    difficulty: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new SafetyDrill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.fen = source["fen"];
	        this.turn = source["turn"];
	        this.mode = source["mode"];
	        this.prompt = source["prompt"];
	        this.targetColor = source["targetColor"];
	        this.correctSquares = source["correctSquares"];
	        this.winningMoveUCI = source["winningMoveUCI"];
	        this.winningMoveSAN = source["winningMoveSAN"];
	        this.explanation = source["explanation"];
	        this.difficulty = source["difficulty"];
	        this.source = source["source"];
	    }
	}
	export class SafetyDrillFeedback {
	    isCorrect: boolean;
	    score: number;
	    correctSquares: string[];
	    missedSquares: string[];
	    wrongSquares: string[];
	    explanation: string;
	    bestMoveUCI?: string;
	    bestMoveSAN?: string;
	
	    static createFrom(source: any = {}) {
	        return new SafetyDrillFeedback(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isCorrect = source["isCorrect"];
	        this.score = source["score"];
	        this.correctSquares = source["correctSquares"];
	        this.missedSquares = source["missedSquares"];
	        this.wrongSquares = source["wrongSquares"];
	        this.explanation = source["explanation"];
	        this.bestMoveUCI = source["bestMoveUCI"];
	        this.bestMoveSAN = source["bestMoveSAN"];
	    }
	}
	export class SolitaireSession {
	    white: string;
	    black: string;
	    event: string;
	    date: string;
	    result: string;
	    userSide: number;
	    currentPlyIdx: number;
	    score: number;
	    correctGuesses: number;
	    totalTurns: number;
	    hintsUsed: number;
	    currentHints: number;
	    moves: MoveRecord[];
	    isFinished: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SolitaireSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.white = source["white"];
	        this.black = source["black"];
	        this.event = source["event"];
	        this.date = source["date"];
	        this.result = source["result"];
	        this.userSide = source["userSide"];
	        this.currentPlyIdx = source["currentPlyIdx"];
	        this.score = source["score"];
	        this.correctGuesses = source["correctGuesses"];
	        this.totalTurns = source["totalTurns"];
	        this.hintsUsed = source["hintsUsed"];
	        this.currentHints = source["currentHints"];
	        this.moves = this.convertValues(source["moves"], MoveRecord);
	        this.isFinished = source["isFinished"];
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

export namespace workstation {
	
	export class StoredAssetInfo {
	    name: string;
	    path: string;
	    category: string;
	    sizeBytes: number;
	    sizeFormatted: string;
	    modifiedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new StoredAssetInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.category = source["category"];
	        this.sizeBytes = source["sizeBytes"];
	        this.sizeFormatted = source["sizeFormatted"];
	        this.modifiedAt = source["modifiedAt"];
	    }
	}
	export class CategoryStorageInfo {
	    categoryId: number;
	    key: string;
	    label: string;
	    path: string;
	    description: string;
	    fileCount: number;
	    totalBytes: number;
	    sizeFormatted: string;
	    assets: StoredAssetInfo[];
	
	    static createFrom(source: any = {}) {
	        return new CategoryStorageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.categoryId = source["categoryId"];
	        this.key = source["key"];
	        this.label = source["label"];
	        this.path = source["path"];
	        this.description = source["description"];
	        this.fileCount = source["fileCount"];
	        this.totalBytes = source["totalBytes"];
	        this.sizeFormatted = source["sizeFormatted"];
	        this.assets = this.convertValues(source["assets"], StoredAssetInfo);
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
	export class FactoryResetOptions {
	    createOptInBackup: boolean;
	    backupDestDir?: string;
	    keepEngines?: boolean;
	    keepBooks?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FactoryResetOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.createOptInBackup = source["createOptInBackup"];
	        this.backupDestDir = source["backupDestDir"];
	        this.keepEngines = source["keepEngines"];
	        this.keepBooks = source["keepBooks"];
	    }
	}
	export class FactoryResetResult {
	    resetCompletedAt: string;
	    backupCreated: boolean;
	    backupArchivePath?: string;
	    wipedPaths: string[];
	    filesRemoved: number;
	    bytesFreed: number;
	    sizeFreedStr: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new FactoryResetResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resetCompletedAt = source["resetCompletedAt"];
	        this.backupCreated = source["backupCreated"];
	        this.backupArchivePath = source["backupArchivePath"];
	        this.wipedPaths = source["wipedPaths"];
	        this.filesRemoved = source["filesRemoved"];
	        this.bytesFreed = source["bytesFreed"];
	        this.sizeFreedStr = source["sizeFreedStr"];
	        this.message = source["message"];
	    }
	}
	export class StorageReport {
	    dataRootDir: string;
	    configRootDir: string;
	    totalBytes: number;
	    totalSizeStr: string;
	    totalFiles: number;
	    categories: CategoryStorageInfo[];
	
	    static createFrom(source: any = {}) {
	        return new StorageReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dataRootDir = source["dataRootDir"];
	        this.configRootDir = source["configRootDir"];
	        this.totalBytes = source["totalBytes"];
	        this.totalSizeStr = source["totalSizeStr"];
	        this.totalFiles = source["totalFiles"];
	        this.categories = this.convertValues(source["categories"], CategoryStorageInfo);
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

