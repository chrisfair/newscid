#!/usr/bin/env python3
"""Rasterize NewScid SVG tournament piece sets into 1152x96 RGBA spritesheets for pkg/tui/assets."""

from io import BytesIO
from pathlib import Path
import subprocess
from PIL import Image

PIECES = ["wK", "wQ", "wR", "wB", "wN", "wP", "bK", "bQ", "bR", "bB", "bN", "bP"]
SETS = ["cburnett", "merida", "staunty", "alpha", "leipzig", "cardinal"]
TILE = 96


def main() -> None:
    repo = Path(__file__).resolve().parents[2]
    src_root = repo / "frontend" / "src" / "assets" / "pieces"
    out_dir = repo / "pkg" / "tui" / "assets"
    out_dir.mkdir(parents=True, exist_ok=True)

    for pset in SETS:
        strip = Image.new("RGBA", (TILE * len(PIECES), TILE), (0, 0, 0, 0))
        for idx, code in enumerate(PIECES):
            svg_path = src_root / pset / f"{code}.svg"
            png_bytes = subprocess.check_output(
                ["rsvg-convert", "-w", str(TILE), "-h", str(TILE), str(svg_path)]
            )
            tile_img = Image.open(BytesIO(png_bytes)).convert("RGBA")
            strip.paste(tile_img, (idx * TILE, 0), tile_img)

        out_path = out_dir / f"{pset}.png"
        strip.save(out_path, format="PNG", optimize=True)
        print(f"Generated {out_path.relative_to(repo)} ({out_path.stat().st_size} bytes)")


if __name__ == "__main__":
    main()
