#!/usr/bin/env python3
"""
RPC integration test runner for newscid-server.
"""

import json
import urllib.request
import os
import sys
import subprocess
import tempfile
import time

URL = "http://127.0.0.1:9124/api/rpc"

def rpc(method, *params):
    payload = json.dumps({
        "id": 1,
        "method": method,
        "params": list(params)
    }).encode("utf-8")
    
    req = urllib.request.Request(URL, data=payload, headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            if data.get("error"):
                return False, data["error"]
            return True, data.get("result")
    except Exception as e:
        return False, str(e)

def ensure_server():
    ok, _ = rpc("GetVersionInfo")
    if ok:
        return None
    binary = "./build/bin/newscid-server"
    if not os.path.exists(binary):
        print(f"Building {binary}...")
        subprocess.run(["go", "build", "-o", binary, "."], check=True)
    proc = subprocess.Popen([binary, "serve", "--port", "9124", "--host", "127.0.0.1"],
                            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    for _ in range(30):
        time.sleep(0.2)
        ok, _ = rpc("GetVersionInfo")
        if ok:
            return proc
    raise RuntimeError("Failed to start newscid-server on port 9124")

def run_test(test_name, method, *params, expect_error=False):
    ok, res = rpc(method, *params)
    if expect_error:
        if not ok:
            print(f"[+] PASSED (Expected Error) {test_name}: {res}")
            return True, res
        else:
            print(f"[-] FAILED {test_name}: expected error, but succeeded with result: {res}")
            return False, res
    else:
        if not ok:
            print(f"[-] FAILED {test_name}: method {method} failed with error: {res}")
            return False, res
        print(f"[+] PASSED {test_name}")
        return True, res

def main():
    print("Running newscid-server RPC integration tests...")
    failures = []

    server_proc = ensure_server()
    if server_proc:
        import atexit
        def _cleanup():
            server_proc.terminate()
            try:
                server_proc.wait(timeout=2)
            except Exception:
                server_proc.kill()
        atexit.register(_cleanup)

    def check(name, ok, condition=True, extra_msg=""):
        if not ok or not condition:
            msg = f"{name} failed" if not extra_msg else f"{name} failed: {extra_msg}"
            failures.append(msg)
            print(f"    --> DEFECT DETECTED: {msg}")

    # -------------------------------------------------------------
    # 1. System, Visual Config, Themes, Piece Sets
    # -------------------------------------------------------------
    print("\n--- 1. System, Visual Config, Themes & Piece Sets ---")
    ok, ver = run_test("GetVersionInfo", "GetVersionInfo")
    check("GetVersionInfo", ok, ver and "version" in ver)

    ok, sets = run_test("GetPieceSets", "GetPieceSets")
    check("GetPieceSets", ok, sets and len(sets) >= 1)

    ok, themes = run_test("GetBoardThemes", "GetBoardThemes")
    check("GetBoardThemes", ok, themes and len(themes) >= 1)

    ok, scid_avail = run_test("IsScidDriverAvailable", "IsScidDriverAvailable")
    check("IsScidDriverAvailable", ok, scid_avail is not None)

    ok, disc_dbs = run_test("GetDiscoveredDatabases", "GetDiscoveredDatabases")
    check("GetDiscoveredDatabases", ok, isinstance(disc_dbs, list))

    ok, sys_bases = run_test("GetScidSystemBases", "GetScidSystemBases")
    check("GetScidSystemBases", ok, isinstance(sys_bases, list))

    # -------------------------------------------------------------
    # 2. Board State, Move Generation, Special Rules & Notation
    # -------------------------------------------------------------
    print("\n--- 2. Board State, Special Rules & Notation ---")
    start_fen = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
    ok, b_state = run_test("ParseFEN: Initial Position", "ParseFEN", start_fen)
    check("ParseFEN: Initial", ok, b_state and b_state.get("turn") == "w" and len(b_state.get("legalMoves", [])) == 20)

    # Test Invalid FEN handling
    ok, _ = run_test("ParseFEN: Invalid FEN", "ParseFEN", "invalid fen string 123", expect_error=True)
    check("ParseFEN: Invalid", ok)

    ok, moves = run_test("GetLegalMoves", "GetLegalMoves", start_fen)
    check("GetLegalMoves", ok, moves and len(moves) == 20)

    # Move: e2e4
    ok, m1 = run_test("MakeMove: e2e4", "MakeMove", start_fen, "e2e4")
    check("MakeMove: e2e4", ok, m1 and m1.get("turn") == "b")
    fen_after_e4 = m1.get("fen")

    # En Passant Test: 1. e4 Nf6 2. e5 d5 3. exd6 (en passant)
    fen_ep_setup = "rnbqkb1r/ppp1pppp/5n2/3pP3/8/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 3"
    ok, m_ep = run_test("MakeMove: En Passant (e5d6)", "MakeMove", fen_ep_setup, "e5d6")
    check("MakeMove: En Passant", ok, m_ep and m_ep.get("turn") == "b" and "3P1n2" in m_ep.get("fen", ""))

    # Castling Test: Kingside & Queenside
    fen_white_castling = "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1"
    ok, m_castle_k = run_test("MakeMove: White O-O (e1g1)", "MakeMove", fen_white_castling, "e1g1")
    check("MakeMove: White O-O", ok, m_castle_k and m_castle_k.get("turn") == "b")
    ok, m_castle_q = run_test("MakeMove: White O-O-O (e1c1)", "MakeMove", fen_white_castling, "e1c1")
    check("MakeMove: White O-O-O", ok, m_castle_q and m_castle_q.get("turn") == "b")

    # Promotion Test: Queen, Rook, Bishop, Knight
    fen_promo = "8/4P3/8/8/8/8/8/4K2k w - - 0 1"
    for promo_piece, promo_char in [("Queen", "e7e8q"), ("Rook", "e7e8r"), ("Bishop", "e7e8b"), ("Knight", "e7e8n")]:
        ok, m_promo = run_test(f"MakeMove: Promotion to {promo_piece} ({promo_char})", "MakeMove", fen_promo, promo_char)
        check(f"MakeMove: Promotion {promo_piece}", ok, m_promo and m_promo.get("turn") == "b")

    # Checkmate Position Test: Scholar's mate
    fen_scholars_mate = "r1bqkb1r/pppp1Qpp/2n5/4p3/2B1n3/8/PPPP1PPP/RNB1K1NR b KQkq - 0 4"
    ok, b_mate = run_test("ParseFEN: Checkmate Position", "ParseFEN", fen_scholars_mate)
    check("ParseFEN: Checkmate Position", ok, b_mate and b_mate.get("isCheckmate") is True and len(b_mate.get("legalMoves", [])) == 0)

    # Stalemate Position Test
    fen_stalemate = "k7/8/1Q6/8/8/8/8/4K3 b - - 0 1"
    ok, b_stale = run_test("ParseFEN: Stalemate Position", "ParseFEN", fen_stalemate)
    check("ParseFEN: Stalemate Position", ok, b_stale and b_stale.get("isStalemate") is True and len(b_stale.get("legalMoves", [])) == 0)

    # Notation Styling & Formatting
    ok, _ = run_test("SetNotationStyle: san", "SetNotationStyle", "san")
    ok, not_style = run_test("GetNotationStyle", "GetNotationStyle")
    check("GetNotationStyle", ok, not_style == "san")

    ok, fmt_move = run_test("FormatMove: e2e4 -> e4", "FormatMove", start_fen, "e2e4", "san")
    check("FormatMove", ok, fmt_move == "e4")

    ok, fmt_seq = run_test("FormatMoveSequence", "FormatMoveSequence", start_fen, ["e2e4", "e7e5", "g1f3"], "san")
    check("FormatMoveSequence", ok, fmt_seq and "Nf3" in fmt_seq)

    # ECO Classification
    ok, eco_res = run_test("ClassifyOpening (Ruy Lopez)", "ClassifyOpening", ["e4", "e5", "Nf3", "Nc6", "Bb5"])
    check("ClassifyOpening", ok, eco_res and "code" in eco_res and eco_res["code"].startswith("C"))

    ok, eco_secs = run_test("GetECOSections", "GetECOSections")
    check("GetECOSections", ok, eco_secs and len(eco_secs) == 5)

    ok, eco_search = run_test("SearchECO: Sicilian", "SearchECO", "B", "Sicilian", 10)
    check("SearchECO", ok, isinstance(eco_search, list) and len(eco_search) > 0)

    ok, class_moves = run_test("ClassifyMoves", "ClassifyMoves", ["e4", "c5"])
    check("ClassifyMoves", ok, class_moves and class_moves.get("code") == "B20")

    # Chess960 (Fischer Random Chess)
    ok, c960_classic = run_test("GetChess960Position: 518 (Classic)", "GetChess960Position", 518)
    check("GetChess960Position: 518", ok, c960_classic and c960_classic.get("pieces") == "RNBQKBNR" and c960_classic.get("isClassic") is True)

    ok, c960_rand = run_test("GenerateRandomChess960", "GenerateRandomChess960")
    check("GenerateRandomChess960", ok, c960_rand and len(c960_rand.get("pieces", "")) == 8)

    # EPD (Extended Position Description)
    epd_sample = 'r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/8/PPPP1PPP/RNBQK1NR b KQkq - bm d5; id "Test-01"; c0 "Fork threat";'
    ok, epd_rec = run_test("ParseEPD", "ParseEPD", epd_sample)
    check("ParseEPD", ok, epd_rec and epd_rec.get("bestMove") == "d5" and epd_rec.get("id") == "Test-01")

    ok, epd_str = run_test("ExportEPD", "ExportEPD", start_fen, {"bm": "e4", "id": "Start-01"})
    check("ExportEPD", ok, epd_str and "bm e4;" in epd_str)

    # Positional Imbalance & Material Radar
    ok, imb_rep = run_test("GetPositionalImbalance", "GetPositionalImbalance", start_fen)
    check("GetPositionalImbalance", ok, imb_rep and imb_rep.get("materialDelta") == 0 and imb_rep.get("white", {}).get("hasBishopPair") is True)

    # Natural Language Move Explainer
    fen_expl = "r1bqk2r/pppp1ppp/2n2n2/2b1p3/2B1P3/3P1N2/PPP2PPP/RNBQK2R w KQkq - 1 5"
    ok, expl = run_test("ExplainMove: Castling", "ExplainMove", fen_expl, "e1g1", 40, "e1g1")
    check("ExplainMove: Castling", ok, expl and "castles" in expl.get("detailedText", "").lower() and "Castling" in expl.get("tacticalMotifs", []))

    # Personal Blunder Vault (SM-2 Spaced Repetition)
    blunder_p = {
        "id": "func-test-blunder",
        "fen": start_fen,
        "playerName": "FuncTester",
        "userColor": "white",
        "playedMove": "f2f3",
        "bestMove": "e2e4",
        "deltaCp": -190,
        "event": "Online Blitz",
        "date": "2026.10.10"
    }
    ok, _ = run_test("AddBlunderToVault", "AddBlunderToVault", blunder_p)
    check("AddBlunderToVault", ok, True)

    ok, due_blunders = run_test("GetDueBlunderPuzzles", "GetDueBlunderPuzzles")
    check("GetDueBlunderPuzzles", ok, due_blunders is not None and len(due_blunders) >= 1)

    ok, reviewed_p = run_test("RecordBlunderReview", "RecordBlunderReview", "func-test-blunder", 4)
    check("RecordBlunderReview", ok, reviewed_p and reviewed_p.get("repetitions") == 1)

    # -------------------------------------------------------------
    # 3. Database Lifecycle, Game CRUD, Searching & Maintenance
    # -------------------------------------------------------------
    print("\n--- 3. Database Lifecycle & Game Operations ---")
    tmp_db_base = tempfile.mktemp(prefix="newscid_full_test_")
    ok, db_info = run_test("CreateDatabase", "CreateDatabase", tmp_db_base, "Full Playthrough Database")
    check("CreateDatabase", ok, db_info and db_info.get("numGames") == 0)

    ok, curr_info = run_test("GetDatabaseInfo", "GetDatabaseInfo")
    check("GetDatabaseInfo", ok, curr_info and curr_info.get("basePath") == tmp_db_base)

    sample_pgn = """[Event "London Tournament"]
[Site "London ENG"]
[Date "1851.06.21"]
[Round "1"]
[White "Anderssen, Adolf"]
[Black "Kieseritzky, Lionel"]
[Result "1-0"]
[ECO "C33"]
[WhiteElo "2600"]
[BlackElo "2550"]

1. e4 e5 2. f4 exf4 3. Bc4 Qh4+ 4. Kf1 b5 5. Bxb5 Nf6 6. Nf3 Qh6 7. d3 Nh5 8. Nh4 Qg5 9. Nf5 c6 10. g4 Nf6 11. Rg1 cxb5 12. h4 Qg6 13. h5 Qg5 14. Qf3 Ng8 15. Bxf4 Qf6 16. Nc3 Bc5 17. Nd5 Qxb2 18. Bd6 Bxg1 19. e5 Qxa1+ 20. Ke2 Na6 21. Nxg7+ Kd8 22. Qf6+ Nxf6 23. Be7# 1-0

[Event "Morphy Opera Game"]
[Site "Paris FRA"]
[Date "1858.10.21"]
[Round "1"]
[White "Morphy, Paul"]
[Black "Duke of Brunswick / Count Isouard"]
[Result "1-0"]
[ECO "C41"]
[WhiteElo "2700"]
[BlackElo "2300"]

1. e4 e5 2. Nf3 d6 3. d4 Bg4 4. dxe5 Bxf3 5. Qxf3 dxe5 6. Bc4 Nf6 7. Qb3 Qe7 8. Nc3 c6 9. Bg5 b5 10. Nxb5 cxb5 11. Bxb5+ Nbd7 12. O-O-O Rd8 13. Rxd7 Rxd7 14. Rd1 Qe6 15. Bxd7+ Nxd7 16. Qb8+ Nxb8 17. Rd8# 1-0

[Event "Casual Blitz Match"]
[Site "Internet"]
[Date "2023.01.15"]
[Round "1"]
[White "Carlsen, Magnus"]
[Black "Nakamura, Hikaru"]
[Result "1/2-1/2"]
[ECO "B50"]
[WhiteElo "2850"]
[BlackElo "2820"]

1. e4 c5 2. Nf3 d6 3. c3 Nf6 4. Be2 g6 5. O-O Bg7 6. Re1 O-O 7. Bf1 e5 8. d4 cxd4 9. cxd4 exd4 10. Nxd4 d5 1/2-1/2
"""

    ok, imp_stats = run_test("ImportPGNBatch (3 games)", "ImportPGNBatch", sample_pgn)
    check("ImportPGNBatch", ok, imp_stats and imp_stats.get("imported") == 3)

    # Query games list
    ok, games_page = run_test("GetGames", "GetGames", 0, 50, {})
    check("GetGames", ok, games_page and games_page.get("total") == 3)

    # Search criteria count
    ok, count_match = run_test("CountSearchMatches (White=Anderssen)", "CountSearchMatches", {"white": "Anderssen"})
    check("CountSearchMatches", ok, count_match == 1)

    # Get single game
    ok, g1 = run_test("GetGame 1 (Anderssen)", "GetGame", 1)
    check("GetGame 1", ok, g1 and g1.get("plyCount", 0) > 0)

    # Update headers
    ok, up_res = run_test("UpdateGameHeaders", "UpdateGameHeaders", 1, {"Event": "Immortal Game 1851", "Round": "Semi-Final"})
    check("UpdateGameHeaders", ok, up_res is True or up_res is None or isinstance(up_res, dict))

    # Search Position
    ok, pos_matches = run_test("SearchPosition (Anderssen opening e4 e5)", "SearchPosition", "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 2")
    check("SearchPosition", ok, isinstance(pos_matches, list) and len(pos_matches) >= 1)

    # Database Players & Statistics
    ok, players = run_test("GetDatabasePlayers", "GetDatabasePlayers")
    check("GetDatabasePlayers", ok, isinstance(players, list) and len(players) >= 3)

    ok, db_stats = run_test("GetDatabaseStatistics", "GetDatabaseStatistics")
    check("GetDatabaseStatistics", ok, db_stats is not None)

    ok, db_demo = run_test("GetDatabaseDemographics", "GetDatabaseDemographics")
    check("GetDatabaseDemographics", ok, db_demo is not None)

    ok, spell_summary = run_test("StandardizePlayerNames", "StandardizePlayerNames")
    check("StandardizePlayerNames", ok, spell_summary is not None)

    ok, rating_res = run_test("GetRatingEvolution (Carlsen, Magnus)", "GetRatingEvolution", "Carlsen, Magnus")
    check("GetRatingEvolution", ok, rating_res is not None)

    ok, scout_res = run_test("GenerateOpponentScoutingReport (Morphy)", "GenerateOpponentScoutingReport", "Morphy")
    check("GenerateOpponentScoutingReport", ok, scout_res and scout_res.get("totalGames") >= 1 and scout_res.get("whiteRecord", {}).get("wins") >= 1)

    # Duplicate check & Verification
    ok, dups = run_test("FindDuplicateGames", "FindDuplicateGames", {"matchMoves": True})
    check("FindDuplicateGames", ok, isinstance(dups, list))

    ok, ver_rep = run_test("VerifyDatabase", "VerifyDatabase")
    check("VerifyDatabase", ok, ver_rep and ver_rep.get("totalGames") == 3)

    # Sort Database
    ok, _ = run_test("SortDatabase by Date", "SortDatabase", {"primaryField": "date", "ascending": True})

    # Delete game & Compact Database
    ok, _ = run_test("DeleteGame (Mark game 3 deleted)", "DeleteGame", 3, True)
    ok, comp_stats = run_test("CompactDatabase", "CompactDatabase")
    check("CompactDatabase", ok, comp_stats and comp_stats.get("purgedGames") == 1)

    # -------------------------------------------------------------
    # 4. Variations, Comments & NAGs
    # -------------------------------------------------------------
    print("\n--- 4. Variations, Comments & NAGs ---")
    ok, _ = run_test("AddGameComment: Ply 1 (Opening comment)", "AddGameComment", 1, "The King's Gambit is played!")
    ok, _ = run_test("SetGameNAG: Ply 1 (NAG 1 = !)", "SetGameNAG", 1, 1)
    ok, _ = run_test("ClearGameNAGs: Ply 1", "ClearGameNAGs", 1)

    # -------------------------------------------------------------
    # 5. Solitaire Chess / Guess the Move
    # -------------------------------------------------------------
    print("\n--- 5. Guess the Move / Solitaire Chess ---")
    ok, gtm_session = run_test("StartGuessTheMove", "StartGuessTheMove", 1, "White")
    check("StartGuessTheMove", ok, gtm_session and len(gtm_session.get("moves", [])) > 0)

    ok, gtm_curr = run_test("GetGuessTheMoveState", "GetGuessTheMoveState")
    check("GetGuessTheMoveState", ok, gtm_curr and gtm_curr.get("isFinished") is False)

    ok, hint_res = run_test("GetGuessTheMoveHint", "GetGuessTheMoveHint")
    check("GetGuessTheMoveHint", ok, hint_res is not None)

    # Submit wrong move
    ok, wrong_feedback = run_test("SubmitGuessTheMove (Wrong move: a4)", "SubmitGuessTheMove", "a4")
    check("SubmitGuessTheMove (Wrong)", ok, wrong_feedback and wrong_feedback.get("isMatch") is False)

    # Submit correct move (e4)
    ok, right_feedback = run_test("SubmitGuessTheMove (Correct move: e4)", "SubmitGuessTheMove", "e4")
    check("SubmitGuessTheMove (Correct)", ok, right_feedback and right_feedback.get("isMatch") is True)

    # Skip move
    ok, skip_feedback = run_test("SkipGuessTheMove", "SkipGuessTheMove")
    check("SkipGuessTheMove", ok, skip_feedback is not None)

    # -------------------------------------------------------------
    # 6. Opening Repertoire & Polyglot
    # -------------------------------------------------------------
    print("\n--- 6. Opening Repertoire & Polyglot ---")
    test_rep = {
        "id": "rep_ruy_lopez",
        "name": "Ruy Lopez Repertoire",
        "color": "w",
        "description": "Solid white repertoire"
    }
    ok, _ = run_test("SaveRepertoire", "SaveRepertoire", test_rep)
    ok, reps = run_test("GetRepertoires", "GetRepertoires")
    check("GetRepertoires", ok, isinstance(reps, list) and len(reps) >= 1)

    ok, loaded_rep = run_test("GetRepertoire", "GetRepertoire", "rep_ruy_lopez")
    check("GetRepertoire", ok, loaded_rep and loaded_rep.get("name") == "Ruy Lopez Repertoire")

    ok, _ = run_test("AddRepertoireMove: e2e4", "AddRepertoireMove", "rep_ruy_lopez", start_fen, "e2e4", "e4", fen_after_e4, "Main open game")

    ok, chk_res = run_test("VerifyRepertoireMove: e2e4", "VerifyRepertoireMove", "rep_ruy_lopez", start_fen, "e2e4")
    check("VerifyRepertoireMove", ok, chk_res and chk_res.get("isRepertoire") is True)

    ok, _ = run_test("RecordRepertoirePractice", "RecordRepertoirePractice", "rep_ruy_lopez", start_fen, "e2e4", True)
    ok, _ = run_test("DeleteRepertoire", "DeleteRepertoire", "rep_ruy_lopez")

    # Polyglot Books
    ok, p_books = run_test("GetPolyglotBooks", "GetPolyglotBooks")
    check("GetPolyglotBooks", ok, isinstance(p_books, list))

    # Opening Tree
    ok, tree_res = run_test("GetOpeningTree (start position)", "GetOpeningTree", start_fen)
    check("GetOpeningTree", ok, tree_res and "moves" in tree_res)

    # -------------------------------------------------------------
    # 7. Personas & Player Simulation
    # -------------------------------------------------------------
    print("\n--- 7. Personas & Player Simulation ---")
    test_persona = {
        "id": "persona_paul_morphy",
        "displayName": "Paul Morphy",
        "title": "Historical Legend",
        "bio": "Pride and Sorrow of Chess",
        "identifiers": ["Morphy, Paul", "Morphy"],
        "chessComUsername": "",
        "lichessUsername": ""
    }
    ok, _ = run_test("SavePersona", "SavePersona", test_persona)
    ok, personas = run_test("GetPersonas", "GetPersonas")
    check("GetPersonas", ok, isinstance(personas, list) and any(p.get("id") == "persona_paul_morphy" for p in personas))

    ok, p_obj = run_test("GetPersona", "GetPersona", "persona_paul_morphy")
    check("GetPersona", ok, p_obj and p_obj.get("displayName") == "Paul Morphy")

    ok, _ = run_test("AddPersonaIdentifier", "AddPersonaIdentifier", "persona_paul_morphy", "P. Morphy")
    ok, _ = run_test("RemovePersonaIdentifier", "RemovePersonaIdentifier", "persona_paul_morphy", "P. Morphy")

    ok, cand = run_test("SuggestPersonaIdentifiers", "SuggestPersonaIdentifiers", "Morphy", ["Morphy, Paul"])
    check("SuggestPersonaIdentifiers", ok, isinstance(cand, list))

    ok, p_personality = run_test("GetPersonaPersonality", "GetPersonaPersonality", "persona_paul_morphy")
    check("GetPersonaPersonality", ok, p_personality is not None)

    ok, p_sim_move = run_test("GetPersonaSimulatedMove (start pos)", "GetPersonaSimulatedMove", "persona_paul_morphy", start_fen, True)
    check("GetPersonaSimulatedMove", ok, p_sim_move and ("uci" in p_sim_move or "san" in p_sim_move))

    ok, p_stats = run_test("GetPersonaStatistics", "GetPersonaStatistics", "persona_paul_morphy")
    check("GetPersonaStatistics", ok, p_stats is not None)

    ok, agg_res = run_test("AggregatePersonaGames", "AggregatePersonaGames", "persona_paul_morphy")
    check("AggregatePersonaGames", ok, agg_res is not None)

    ok, _ = run_test("DeletePersona", "DeletePersona", "persona_paul_morphy")

    # -------------------------------------------------------------
    # 8. Sync Profiles & Online (offline-safe RPC verification)
    # -------------------------------------------------------------
    print("\n--- 8. Sync Profiles & Online Tools ---")
    test_profile = {
        "id": "sync_test_profile",
        "name": "Test Lichess Profile",
        "lichessUsername": "thibault",
        "chessComUsername": "",
        "targetMode": "active"
    }
    ok, _ = run_test("SaveSyncProfile", "SaveSyncProfile", test_profile)
    ok, profiles = run_test("GetSyncProfiles", "GetSyncProfiles")
    check("GetSyncProfiles", ok, isinstance(profiles, list) and any(p.get("id") == "sync_test_profile" for p in profiles))

    ok, prof = run_test("GetSyncProfile", "GetSyncProfile", "sync_test_profile")
    check("GetSyncProfile", ok, prof and prof.get("lichessUsername") == "thibault")

    ok, _ = run_test("DeleteSyncProfile", "DeleteSyncProfile", "sync_test_profile")

    # TWIC checks (with timeout safety)
    ok, twic_issue = run_test("CheckTWICIssue: 1500", "CheckTWICIssue", 1500)
    check("CheckTWICIssue", ok, isinstance(twic_issue, bool))

    # -------------------------------------------------------------
    # 9. Tournament Tools & Crosstables
    # -------------------------------------------------------------
    print("\n--- 9. Tournament Tools & Crosstables ---")
    ok, events = run_test("GetTournamentEvents", "GetTournamentEvents")
    check("GetTournamentEvents", ok, isinstance(events, list))

    if events and len(events) > 0:
        ev_name = events[0].get("name", "")
        ev_site = events[0].get("site", "")
        ev_date = events[0].get("date", "")
        ok, xtab = run_test(f"GetTournamentCrosstable: {ev_name}", "GetTournamentCrosstable", ev_name, ev_site, ev_date)
        check("GetTournamentCrosstable", ok, xtab is not None)

        ok, xtab_txt = run_test("ExportCrosstableText", "ExportCrosstableText", ev_name, ev_site, ev_date)
        check("ExportCrosstableText", ok, xtab_txt and len(xtab_txt) > 0)

    # -------------------------------------------------------------
    # 10. Engine Configuration & Tablebase
    # -------------------------------------------------------------
    print("\n--- 10. Engine Configuration & Tablebase ---")
    ok, avail_engs = run_test("GetAvailableEngines", "GetAvailableEngines")
    check("GetAvailableEngines", ok, isinstance(avail_engs, list))

    ok, disc_engs = run_test("GetDiscoveredEngines", "GetDiscoveredEngines")
    check("GetDiscoveredEngines", ok, isinstance(disc_engs, list))

    ok, _ = run_test("SetEngineMultiPV: 3", "SetEngineMultiPV", 3)
    ok, mpv_val = run_test("GetEngineMultiPV", "GetEngineMultiPV")
    check("GetEngineMultiPV", ok, mpv_val == 3)

    ok, _ = run_test("SetEngineThreads: 2", "SetEngineThreads", 2)
    ok, _ = run_test("SetEngineHash: 64", "SetEngineHash", 64)

    ok, is_match = run_test("IsEngineMatchRunning", "IsEngineMatchRunning")
    check("IsEngineMatchRunning", ok, is_match is False)

    # Tablebase probe (KQK endgame)
    fen_tb = "8/8/8/8/8/5k2/8/4K1Q1 w - - 0 1"
    ok, tb_res = run_test("ProbeTablebase (KQK position)", "ProbeTablebase", fen_tb)
    check("ProbeTablebase", ok, tb_res is not None)

    # Full Game Annotation (AnnotatePGN)
    test_pgn = '[Event "Fool\'s Mate Test"]\n[White "White"]\n[Black "Black"]\n[Result "0-1"]\n\n1. f3 e5 2. g4 Qh4# 0-1\n'
    ok, ann_res = run_test("AnnotatePGN (Fool's Mate)", "AnnotatePGN", test_pgn, 100)
    check("AnnotatePGN", ok, ann_res and len(ann_res.get("moveAnalyses", [])) == 4)

    # -------------------------------------------------------------
    # 11. Export Formats (PGN, HTML, LaTeX)
    # -------------------------------------------------------------
    print("\n--- 11. Game Exporting (PGN, HTML, LaTeX) ---")
    ok, pgn_export = run_test("ExportPGN (Game 1)", "ExportPGN", g1)
    check("ExportPGN", ok, pgn_export and "Anderssen" in pgn_export)

    html_opts = {
        "includeDiagrams": True,
        "includeComments": True,
        "includeVariations": True,
        "theme": "default"
    }
    ok, html_export = run_test("ExportGameHTML (Game 1)", "ExportGameHTML", 1, html_opts)
    check("ExportGameHTML", ok, html_export and "<html" in html_export.lower())

    latex_opts = {
        "includeDiagrams": False,
        "includeComments": True,
        "includeVariations": True,
        "fontSize": "11pt"
    }
    ok, latex_export = run_test("ExportGameLaTeX (Game 1)", "ExportGameLaTeX", 1, latex_opts)
    check("ExportGameLaTeX", ok, latex_export and "\\documentclass" in latex_export)

    tmp_export_file = tempfile.mktemp(prefix="newscid_export_", suffix=".html")
    ok, _ = run_test("SaveExportFile", "SaveExportFile", tmp_export_file, html_export or "test")
    check("SaveExportFile", ok, os.path.exists(tmp_export_file))
    if os.path.exists(tmp_export_file):
        os.remove(tmp_export_file)

    # -------------------------------------------------------------
    # 12. Novel Training & Analytical Tools
    # -------------------------------------------------------------
    print("\n--- 12. Novel Training & Analytical Tools ---")
    # Intuition Storm
    ok, intuition_batch = run_test("GetIntuitionDrillBatch", "GetIntuitionDrillBatch", 3)
    check("GetIntuitionDrillBatch", ok, isinstance(intuition_batch, list) and len(intuition_batch) > 0)

    ok, intuition_sub = run_test("SubmitIntuitionAnswer", "SubmitIntuitionAnswer", "int-1", "equal", 1200)
    check("SubmitIntuitionAnswer", ok, intuition_sub and "isCorrect" in intuition_sub)

    # Prophylaxis Drills
    ok, pro_batch = run_test("GetProphylaxisDrillBatch", "GetProphylaxisDrillBatch", 3, "all")
    check("GetProphylaxisDrillBatch", ok, isinstance(pro_batch, list) and len(pro_batch) > 0)

    ok, pro_sub = run_test("SubmitProphylaxisAnswer", "SubmitProphylaxisAnswer", "pro-01-karpov-kh1", "g1h1")
    check("SubmitProphylaxisAnswer", ok, pro_sub and "isCorrect" in pro_sub)

    # Time Management Analysis (Game 1)
    ok, tm_res = run_test("AnalyzeGameTimeManagement (Game 1)", "AnalyzeGameTimeManagement", 1)
    check("AnalyzeGameTimeManagement", ok, tm_res is not None)

    # Repertoire Shortener
    short_opts = {
        "repertoireId": "white-openings-italian",
        "targetCoveragePct": 80.0,
        "minFrequencyPct": 5.0,
        "maxDepth": 6,
        "preserveAnnotated": True,
        "saveAsNew": False
    }
    ok, short_res = run_test("ShortenRepertoire", "ShortenRepertoire", short_opts)
    check("ShortenRepertoire", ok, short_res and "originalMoveCount" in short_res)

    # -------------------------------------------------------------
    # 13. Cleanup & Database Close
    # -------------------------------------------------------------
    print("\n--- 13. Cleanup ---")
    ok, _ = run_test("CloseDatabase", "CloseDatabase")
    for ext in [".si4", ".sn4", ".sg4"]:
        fpath = tmp_db_base + ext
        if os.path.exists(fpath):
            try:
                os.remove(fpath)
            except:
                pass


    if failures:
        print(f"[-] {len(failures)} test failure(s):")
        for idx, f in enumerate(failures, 1):
            print(f"  {idx}. {f}")
        sys.exit(1)
    else:
        print("[+] All RPC integration checks passed.")
        sys.exit(0)

if __name__ == "__main__":
    main()
