#!/usr/bin/env python3
"""Generate Microsoft Store MSIX visual assets (tiles/icons/splash) and Partner Center Store listing graphics."""
import os
import subprocess
import sys
from PIL import Image, ImageDraw, ImageFont

REPO_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
ICON_SVG = os.path.join(REPO_ROOT, "build", "appicon.svg")
ICON_PNG = os.path.join(REPO_ROOT, "build", "appicon.png")
ASSETS_DIR = os.path.join(REPO_ROOT, "packaging", "msix", "Assets")
STORE_DIR = os.path.join(REPO_ROOT, "packaging", "msix", "store-assets")

BG_DARK = (15, 23, 42, 255)      # #0f172a
BG_PANEL = (24, 24, 27, 255)     # #18181b
BG_CARD = (30, 41, 59, 255)      # #1e293b
BORDER = (51, 65, 85, 255)       # #334155
ACCENT = (16, 185, 129, 255)     # #10b981
ACCENT_BLUE = (56, 189, 248, 255)
AMBER = (251, 191, 36, 255)
TEXT_MAIN = (241, 245, 249, 255)
TEXT_MUTED = (148, 163, 184, 255)
SQ_LIGHT = (232, 214, 183, 255)
SQ_DARK = (166, 121, 82, 255)


def load_font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont:
    candidates = [
        "/usr/share/fonts/TTF/DejaVuSans-Bold.ttf" if bold else "/usr/share/fonts/TTF/DejaVuSans.ttf",
        "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf" if bold else "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
    ]
    for p in candidates:
        if os.path.exists(p):
            return ImageFont.truetype(p, size)
    return ImageFont.load_default()


def render_base_icon(size: int = 1024) -> Image.Image:
    os.makedirs(ASSETS_DIR, exist_ok=True)
    tmp_png = os.path.join(ASSETS_DIR, "_base_icon_tmp.png")
    if os.path.exists(ICON_SVG):
        subprocess.run(
            ["rsvg-convert", "-w", str(size), "-h", str(size), ICON_SVG, "-o", tmp_png],
            check=True,
        )
        img = Image.open(tmp_png).convert("RGBA")
        os.remove(tmp_png)
        return img
    return Image.open(ICON_PNG).convert("RGBA").resize((size, size), Image.Resampling.LANCZOS)


def make_centered_tile(icon: Image.Image, w: int, h: int, pad_ratio: float = 0.16, bg=BG_DARK) -> Image.Image:
    canvas = Image.new("RGBA", (w, h), bg)
    avail = int(min(w, h) * (1.0 - 2.0 * pad_ratio))
    resized = icon.resize((avail, avail), Image.Resampling.LANCZOS)
    x = (w - avail) // 2
    y = (h - avail) // 2
    canvas.alpha_composite(resized, (x, y))
    return canvas


def make_wide_or_splash(icon: Image.Image, w: int, h: int, title: str, subtitle: str) -> Image.Image:
    canvas = Image.new("RGBA", (w, h), BG_DARK)
    draw = ImageDraw.Draw(canvas)
    icon_sz = int(h * 0.62)
    resized = icon.resize((icon_sz, icon_sz), Image.Resampling.LANCZOS)
    ix = int(w * 0.10)
    iy = (h - icon_sz) // 2
    canvas.alpha_composite(resized, (ix, iy))

    tx = ix + icon_sz + int(w * 0.045)
    f_title = load_font(max(16, int(h * 0.21)), bold=True)
    f_sub = load_font(max(11, int(h * 0.105)), bold=False)
    draw.text((tx, int(h * 0.31)), title, font=f_title, fill=TEXT_MAIN)
    draw.text((tx, int(h * 0.56)), subtitle, font=f_sub, fill=TEXT_MUTED)
    return canvas


def draw_chessboard(draw: ImageDraw.ImageDraw, x0: int, y0: int, size: int) -> None:
    sq = size // 8
    for r in range(8):
        for c in range(8):
            color = SQ_LIGHT if (r + c) % 2 == 0 else SQ_DARK
            draw.rectangle([x0 + c * sq, y0 + r * sq, x0 + (c + 1) * sq, y0 + (r + 1) * sq], fill=color)
    # Subtle highlight on e2->e4
    draw.rectangle([x0 + 4 * sq, y0 + 6 * sq, x0 + 5 * sq, y0 + 7 * sq], outline=ACCENT, width=4)
    draw.rectangle([x0 + 4 * sq, y0 + 4 * sq, x0 + 5 * sq, y0 + 5 * sq], outline=ACCENT, width=4)
    draw.rectangle([x0, y0, x0 + 8 * sq, y0 + 8 * sq], outline=BORDER, width=3)


def make_store_screenshot(icon: Image.Image, filename: str, badge: str, headline: str, bullets: list[str], metric_cards: list[tuple[str, str, tuple]]) -> None:
    w, h = 1920, 1080
    img = Image.new("RGBA", (w, h), BG_DARK)
    draw = ImageDraw.Draw(img)

    # Top header bar
    draw.rectangle([0, 0, w, 92], fill=(11, 17, 32, 255))
    draw.line([(0, 92), (w, 92)], fill=BORDER, width=2)
    sm_icon = icon.resize((52, 52), Image.Resampling.LANCZOS)
    img.alpha_composite(sm_icon, (40, 20))
    draw.text((108, 22), "NewScid Chess Database", font=load_font(28, True), fill=TEXT_MAIN)
    draw.text((108, 56), "Native Scid v4 (.si4/.sg4/.sn4) & PGN Workstation  |  Windows Store Edition", font=load_font(16), fill=TEXT_MUTED)

    # Badge pill top-right
    f_badge = load_font(20, True)
    draw.rounded_rectangle([w - 390, 24, w - 40, 68], radius=22, fill=(16, 185, 129, 38), outline=ACCENT, width=2)
    draw.text((w - 365, 34), badge, font=f_badge, fill=ACCENT)

    # Left: Chessboard workstation preview
    draw_chessboard(draw, 64, 150, 720)
    # Eval bar beside board
    draw.rectangle([798, 150, 822, 870], fill=(30, 41, 59, 255), outline=BORDER, width=2)
    draw.rectangle([800, 430, 820, 868], fill=(226, 232, 240, 255))
    draw.text((64, 892), "Stockfish Multi-PV  depth 28  |  +0.42  1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6", font=load_font(20), fill=ACCENT_BLUE)
    draw.text((64, 930), "7-Piece Syzygy Tablebase  |  Dockable Multi-Window Panels  |  Kitty & Sixel TUI Included", font=load_font(18), fill=TEXT_MUTED)

    # Right: Feature showcase panel
    rx = 885
    draw.rounded_rectangle([rx, 150, w - 64, 970], radius=20, fill=BG_PANEL, outline=BORDER, width=2)
    draw.text((rx + 42, 190), headline, font=load_font(42, True), fill=TEXT_MAIN)

    # Metric cards row
    card_w = (w - 64 - rx - 84 - 40) // 3
    for i, (label, val, col) in enumerate(metric_cards[:3]):
        cx = rx + 42 + i * (card_w + 20)
        cy = 278
        draw.rounded_rectangle([cx, cy, cx + card_w, cy + 135], radius=14, fill=BG_CARD, outline=BORDER, width=2)
        draw.text((cx + 22, cy + 20), label, font=load_font(18), fill=TEXT_MUTED)
        draw.text((cx + 22, cy + 58), val, font=load_font(34, True), fill=col)

    # Bullet list
    by = 460
    f_bul = load_font(26)
    for b in bullets:
        draw.ellipse([rx + 46, by + 10, rx + 60, by + 24], fill=ACCENT)
        draw.text((rx + 78, by), b, font=f_bul, fill=TEXT_MAIN)
        by += 68

    # Bottom footer inside card
    draw.rounded_rectangle([rx + 42, 855, w - 106, 932], radius=12, fill=(15, 23, 42, 255), outline=BORDER, width=1)
    draw.text((rx + 66, 880), "Free Edition on Microsoft Store  •  One-Time $9.99 Permanent Unlock  •  100% Local & Private", font=load_font(20, True), fill=AMBER)

    img.convert("RGB").save(os.path.join(STORE_DIR, filename), "PNG", optimize=True)


def make_poster_art(icon: Image.Image) -> None:
    w, h = 720, 1080
    img = Image.new("RGBA", (w, h), BG_DARK)
    draw = ImageDraw.Draw(img)
    draw.rectangle([24, 24, w - 24, h - 24], outline=BORDER, width=3)
    ic = icon.resize((360, 360), Image.Resampling.LANCZOS)
    img.alpha_composite(ic, ((w - 360) // 2, 130))
    draw.text((130, 540), "NewScid", font=load_font(86, True), fill=TEXT_MAIN)
    draw.text((95, 650), "Chess Database & Analysis", font=load_font(38, True), fill=ACCENT)
    draw.text((95, 725), "• Native Scid v4 (.si4) & PGN", font=load_font(26), fill=TEXT_MUTED)
    draw.text((95, 775), "• Multi-PV UCI Engine & Syzygy", font=load_font(26), fill=TEXT_MUTED)
    draw.text((95, 825), "• Fair-Play Radar Forensics", font=load_font(26), fill=TEXT_MUTED)
    draw.text((95, 875), "• Opening Prep & Tactics Drills", font=load_font(26), fill=TEXT_MUTED)
    img.convert("RGB").save(os.path.join(STORE_DIR, "PosterArt-720x1080.png"), "PNG", optimize=True)


def make_hero_art(icon: Image.Image) -> None:
    img = make_wide_or_splash(
        icon,
        1920,
        1080,
        "NewScid Chess Database",
        "Native Scid v4 (.si4) • Multi-PV Engine • Fair-Play Forensics • Opening Prep",
    )
    img.convert("RGB").save(os.path.join(STORE_DIR, "HeroArt-1920x1080.png"), "PNG", optimize=True)


def main() -> int:
    os.makedirs(ASSETS_DIR, exist_ok=True)
    os.makedirs(STORE_DIR, exist_ok=True)
    base_icon = render_base_icon(1024)

    # Required MSIX package visual assets (scale-100 and scale-200)
    square_specs = [
        ("StoreLogo.png", 50, 50, 0.08),
        ("StoreLogo.scale-100.png", 50, 50, 0.08),
        ("StoreLogo.scale-200.png", 100, 100, 0.08),
        ("Square44x44Logo.png", 44, 44, 0.06),
        ("Square44x44Logo.scale-100.png", 44, 44, 0.06),
        ("Square44x44Logo.scale-200.png", 88, 88, 0.06),
        ("Square44x44Logo.targetsize-48.png", 48, 48, 0.04),
        ("Square44x44Logo.targetsize-256.png", 256, 256, 0.04),
        ("Square71x71Logo.png", 71, 71, 0.14),
        ("Square71x71Logo.scale-100.png", 71, 71, 0.14),
        ("Square71x71Logo.scale-200.png", 142, 142, 0.14),
        ("Square150x150Logo.png", 150, 150, 0.16),
        ("Square150x150Logo.scale-100.png", 150, 150, 0.16),
        ("Square150x150Logo.scale-200.png", 300, 300, 0.16),
        ("Square310x310Logo.png", 310, 310, 0.18),
        ("Square310x310Logo.scale-100.png", 310, 310, 0.18),
        ("Square310x310Logo.scale-200.png", 620, 620, 0.18),
    ]
    for name, w, h, pad in square_specs:
        make_centered_tile(base_icon, w, h, pad).save(os.path.join(ASSETS_DIR, name), "PNG", optimize=True)

    wide_specs = [
        ("Wide310x150Logo.png", 310, 150),
        ("Wide310x150Logo.scale-100.png", 310, 150),
        ("Wide310x150Logo.scale-200.png", 620, 300),
        ("SplashScreen.png", 620, 300),
        ("SplashScreen.scale-100.png", 620, 300),
        ("SplashScreen.scale-200.png", 1240, 600),
    ]
    for name, w, h in wide_specs:
        make_wide_or_splash(base_icon, w, h, "NewScid", "Chess Database & Analysis").save(
            os.path.join(ASSETS_DIR, name), "PNG", optimize=True
        )

    # Partner Center Store listing graphics
    make_centered_tile(base_icon, 300, 300, 0.08).convert("RGB").save(
        os.path.join(STORE_DIR, "StoreIcon-300x300.png"), "PNG", optimize=True
    )
    make_poster_art(base_icon)
    make_hero_art(base_icon)

    make_store_screenshot(
        base_icon,
        "Screenshot-1-Workstation-1920x1080.png",
        "ANALYSIS WORKSTATION",
        "Native Scid v4 & Multi-PV Analysis",
        [
            "Reads & writes .si4/.sg4/.sn4, .pgn, and .pgn.zst directly",
            "Multi-PV UCI engine analysis with live evaluation graph",
            "Position, material, pattern, ECO & header filters",
            "7 dockable panels with multi-monitor & tiling support",
            "Incremental Lichess & Chess.com game synchronization",
        ],
        [
            ("FORMATS", "SI4 + PGN", ACCENT),
            ("TABLEBASE", "7-Piece", ACCENT_BLUE),
            ("WORKSPACES", "4 Modes", AMBER),
        ],
    )
    make_store_screenshot(
        base_icon,
        "Screenshot-2-FairPlay-1920x1080.png",
        "FAIR-PLAY FORENSICS",
        "6-Signal Fair-Play Radar & Hard-Move Audit",
        [
            "Composite Risk Score (CRS 0-100) + Cheat Likelihood %",
            "Platform-calibrated Chess.com & Lichess accuracy bands",
            "Move-time cadence CV, micro-burst & clutch-toggle checks",
            "Poisson-binomial hard-move benchmark & OTB baselines",
            "Shared Community Telemetry Network & Ban-Watch ledger",
        ],
        [
            ("SIGNALS", "6 Core", ACCENT),
            ("BASELINES", "9.7M Games", ACCENT_BLUE),
            ("TELEMETRY", "Live Sync", AMBER),
        ],
    )
    make_store_screenshot(
        base_icon,
        "Screenshot-3-OpeningPrep-1920x1080.png",
        "OPENING PREPARATION",
        "Opponent Scouting & Polyglot Books",
        [
            "Live opening tree with win/draw/loss statistics",
            "Opponent scouting reports & vulnerability finder",
            "Export targeted refutation books to Polyglot (.bin)",
            "Spaced-repetition (SM-2) opening repertoire trainer",
            "Spar against historical player repertoires + engine",
        ],
        [
            ("OPENING BOOK", "Polyglot", ACCENT),
            ("REPERTOIRE", "SM-2 SRS", ACCENT_BLUE),
            ("PERSONAS", "Multi-Site", AMBER),
        ],
    )
    make_store_screenshot(
        base_icon,
        "Screenshot-4-Training-1920x1080.png",
        "TRAINING & TUI",
        "Tactical Drills, Blunder Vault & Terminal UI",
        [
            "Mine tactical puzzles directly from your own games",
            "Cyclical repetition, Solitaire Chess & blindfold drills",
            "Clock & time-management tank/panic diagnostics",
            "Export annotated games to PGN, HTML, LaTeX & PDF",
            "Full Kitty & Sixel graphics Terminal UI included",
        ],
        [
            ("DRILLS", "6 Modes", ACCENT),
            ("EXPORTS", "PDF + TeX", ACCENT_BLUE),
            ("INTERFACES", "GUI + TUI", AMBER),
        ],
    )

    print(f"Generated {len(os.listdir(ASSETS_DIR))} MSIX assets in {ASSETS_DIR}")
    print(f"Generated {len(os.listdir(STORE_DIR))} Store listing graphics in {STORE_DIR}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
