#!/usr/bin/env python3
"""Build a Windows x64 NewScid .exe and unsigned Microsoft Store .msix directly on Linux."""
import argparse
import base64
import hashlib
import os
import re
import shutil
import subprocess
import sys
import xml.etree.ElementTree as ET
import zipfile

REPO_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
MSIX_DIR = os.path.join(REPO_ROOT, "packaging", "msix")
STAGE_DIR = os.path.join(REPO_ROOT, "build", "msix-stage")
DIST_DIR = os.path.join(REPO_ROOT, "build", "dist")

BLOCK_SIZE = 65536


def read_default_version() -> str:
    ver_file = os.path.join(REPO_ROOT, "VERSION")
    if os.path.exists(ver_file):
        v = open(ver_file, "r", encoding="utf-8").read().strip()
        parts = v.split(".")
        while len(parts) < 4:
            parts.append("0")
        return ".".join(parts[:4])
    return "2.0.0.0"


def build_frontend() -> None:
    print("==> Building Vue 3 frontend...")
    subprocess.run(["npm", "run", "build"], cwd=os.path.join(REPO_ROOT, "frontend"), check=True)


def cross_compile_windows_exe(out_exe: str, addon_store_id: str) -> None:
    print("==> Cross-compiling Windows x64 newscid.exe on Linux...")
    env = os.environ.copy()
    env["GOOS"] = "windows"
    env["GOARCH"] = "amd64"
    env["CGO_ENABLED"] = "0"
    ldflags = "-H windowsgui -s -w"
    if addon_store_id:
        ldflags += f" -X newscid/pkg/support.StoreAddOnStoreID={addon_store_id}"
    subprocess.run(
        ["go", "build", "-tags", "desktop,production", "-ldflags", ldflags, "-o", out_exe, "."],
        cwd=REPO_ROOT,
        env=env,
        check=True,
    )


def build_content_types_xml(rel_paths: list[str]) -> bytes:
    ext_map = {
        "exe": "application/x-msdownload",
        "dll": "application/x-msdownload",
        "png": "image/png",
        "svg": "image/svg+xml",
        "xml": "application/vnd.ms-appx.manifest+xml",
        "pri": "application/octet-stream",
    }
    seen_exts = set()
    lines = [
        '<?xml version="1.0" encoding="UTF-8"?>',
        '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">',
    ]
    for p in sorted(rel_paths):
        ext = os.path.splitext(p)[1].lstrip(".").lower()
        if ext and ext not in seen_exts and ext != "xml":
            seen_exts.add(ext)
            ctype = ext_map.get(ext, "application/octet-stream")
            lines.append(f'  <Default Extension="{ext}" ContentType="{ctype}"/>')
    lines.append('  <Override PartName="/AppxManifest.xml" ContentType="application/vnd.ms-appx.manifest+xml"/>')
    lines.append('  <Override PartName="/AppxBlockMap.xml" ContentType="application/vnd.ms-appx.blockmap+xml"/>')
    lines.append("</Types>")
    return ("\n".join(lines) + "\n").encode("utf-8")


def build_blockmap_xml(stage_dir: str, rel_paths: list[str], compressed_sizes: dict[str, int]) -> bytes:
    ns = "http://schemas.microsoft.com/appx/2010/blockmap"
    ET.register_namespace("", ns)
    root = ET.Element(f"{{{ns}}}BlockMap", {"HashMethod": "http://www.w3.org/2001/04/xmlenc#sha256"})
    for rel in sorted(rel_paths):
        full = os.path.join(stage_dir, rel)
        size = os.path.getsize(full)
        win_name = rel.replace("/", "\\")
        attrs = {
            "Name": win_name,
            "Size": str(size),
            "LfhSize": str(30 + len(rel.encode("utf-8"))),
        }
        f_el = ET.SubElement(root, f"{{{ns}}}File", attrs)
        with open(full, "rb") as fh:
            while True:
                chunk = fh.read(BLOCK_SIZE)
                if not chunk:
                    break
                digest = base64.b64encode(hashlib.sha256(chunk).digest()).decode("ascii")
                b_attrs = {"Hash": digest}
                if rel in compressed_sizes and size <= BLOCK_SIZE:
                    b_attrs["Size"] = str(compressed_sizes[rel])
                ET.SubElement(f_el, f"{{{ns}}}Block", b_attrs)
    return b'<?xml version="1.0" encoding="UTF-8" standalone="no"?>\n' + ET.tostring(root, encoding="utf-8")


def pack_msix(stage_dir: str, out_msix: str) -> None:
    rel_paths = []
    for root_dir, _, files in os.walk(stage_dir):
        for fname in sorted(files):
            full = os.path.join(root_dir, fname)
            rel = os.path.relpath(full, stage_dir).replace("\\", "/")
            rel_paths.append(rel)

    compressed_sizes: dict[str, int] = {}
    with zipfile.ZipFile(out_msix, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=6) as zf:
        for rel in sorted(rel_paths):
            full = os.path.join(stage_dir, rel)
            zf.write(full, arcname=rel)
            compressed_sizes[rel] = zf.getinfo(rel).compress_size

        blockmap = build_blockmap_xml(stage_dir, rel_paths, compressed_sizes)
        zf.writestr("AppxBlockMap.xml", blockmap)

        ctypes = build_content_types_xml(rel_paths)
        zf.writestr("[Content_Types].xml", ctypes)


def main() -> int:
    parser = argparse.ArgumentParser(description="Build NewScid Windows x64 MSIX directly on Linux.")
    parser.add_argument("--version", default=read_default_version(), help="4-part MSIX version (default: from VERSION)")
    parser.add_argument("--publisher", default="CN=REPLACE-WITH-PARTNER-CENTER-PUBLISHER-ID", help="Partner Center Publisher CN")
    parser.add_argument("--addon-store-id", default="", help="Partner Center $9.99 add-on Store ID")
    parser.add_argument("--stockfish", default="", help="Optional path to Windows stockfish.exe to bundle")
    parser.add_argument("--skip-frontend", action="store_true", help="Skip npm run build")
    args = parser.parse_args()

    if not args.skip_frontend:
        build_frontend()

    os.makedirs(DIST_DIR, exist_ok=True)
    if os.path.exists(STAGE_DIR):
        shutil.rmtree(STAGE_DIR)
    os.makedirs(STAGE_DIR, exist_ok=True)

    exe_path = os.path.join(STAGE_DIR, "newscid.exe")
    cross_compile_windows_exe(exe_path, args.addon_store_id)

    shutil.copytree(os.path.join(MSIX_DIR, "Assets"), os.path.join(STAGE_DIR, "Assets"))

    if args.stockfish and os.path.exists(args.stockfish):
        eng_dir = os.path.join(STAGE_DIR, "engines")
        os.makedirs(eng_dir, exist_ok=True)
        shutil.copy2(args.stockfish, os.path.join(eng_dir, "stockfish.exe"))

    manifest_src = open(os.path.join(MSIX_DIR, "AppxManifest.xml"), "r", encoding="utf-8").read()
    manifest_src = re.sub(r'Version="[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"', f'Version="{args.version}"', manifest_src)
    manifest_src = re.sub(r'Publisher="[^"]+"', f'Publisher="{args.publisher}"', manifest_src)
    with open(os.path.join(STAGE_DIR, "AppxManifest.xml"), "w", encoding="utf-8") as fh:
        fh.write(manifest_src)

    out_msix = os.path.join(DIST_DIR, f"NewScid-{args.version}-x64.msix")
    print(f"==> Packing {out_msix}...")
    pack_msix(STAGE_DIR, out_msix)
    shutil.rmtree(STAGE_DIR, ignore_errors=True)
    size_mb = os.path.getsize(out_msix) / (1024 * 1024)
    print(f"DONE: {out_msix} ({size_mb:.2f} MB)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
