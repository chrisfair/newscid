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

