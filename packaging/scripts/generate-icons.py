#!/usr/bin/env python3
"""
Generate standard application icon sizes from source high-res icon.
Outputs PNG icons to the specified directory:
  16x16, 32x32, 48x48, 64x64, 128x128, 256x256, 512x512, 1024x1024
"""

import sys
import os
import shutil
import subprocess

SIZES = [16, 32, 48, 64, 128, 256, 512, 1024]

def main():
    src_icon = sys.argv[1] if len(sys.argv) > 1 else "build/appicon.png"
    out_dir = sys.argv[2] if len(sys.argv) > 2 else "build/staging/icons"

    if not os.path.isfile(src_icon):
        print(f"Error: Source icon not found at {src_icon}", file=sys.stderr)
        sys.exit(1)

    os.makedirs(out_dir, exist_ok=True)

    # Try PIL / Pillow first
    try:
        from PIL import Image
        img = Image.open(src_icon)
        for size in SIZES:
            out_file = os.path.join(out_dir, f"{size}x{size}.png")
            resized = img.resize((size, size), Image.Resampling.LANCZOS)
            resized.save(out_file, format="PNG")
        print(f"Generated {len(SIZES)} icon sizes in {out_dir} via PIL")
        return
    except ImportError:
        pass

    # Try ImageMagick convert / magick
    magick_cmd = shutil.which("magick") or shutil.which("convert")
    if magick_cmd:
        for size in SIZES:
            out_file = os.path.join(out_dir, f"{size}x{size}.png")
            subprocess.run([magick_cmd, src_icon, "-resize", f"{size}x{size}", out_file], check=True)
        print(f"Generated {len(SIZES)} icon sizes in {out_dir} via ImageMagick")
        return

    # Fallback: copy original to each size
    for size in SIZES:
        out_file = os.path.join(out_dir, f"{size}x{size}.png")
        shutil.copyfile(src_icon, out_file)
    print(f"Copied base icon to {len(SIZES)} sizes in {out_dir} (fallback)")

if __name__ == "__main__":
    main()
