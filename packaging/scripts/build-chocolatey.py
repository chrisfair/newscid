#!/usr/bin/env python3
"""
Build Chocolatey (.nupkg) package for NewScid.
Supports both native 'choco pack' on Windows and standalone OPC NuGet/ZIP packaging
on Linux and macOS without external dependencies.
"""

import sys
import os
import shutil
import subprocess
import zipfile

def main():
    version = os.environ.get("APP_VERSION", "1.0.0")
    dist_dir = sys.argv[1] if len(sys.argv) > 1 else "build/dist"
    staging_dir = "build/staging/chocolatey"

    print(f"==> Packaging Chocolatey package for NewScid v{version}...")

    os.makedirs(staging_dir, exist_ok=True)
    os.makedirs(os.path.join(staging_dir, "tools"), exist_ok=True)
    os.makedirs(dist_dir, exist_ok=True)

    # 1. Update and copy .nuspec
    nuspec_src = "packaging/chocolatey/newscid.nuspec"
    nuspec_dest = os.path.join(staging_dir, "newscid.nuspec")
    with open(nuspec_src, "r", encoding="utf-8") as f:
        content = f.read()

    # Replace version
    import re
    content = re.sub(r"<version>[^<]+</version>", f"<version>{version}</version>", content)
    with open(nuspec_dest, "w", encoding="utf-8") as f:
        f.write(content)

    # 2. Copy tools
    tools_src = "packaging/chocolatey/tools"
    for item in os.listdir(tools_src):
        src_path = os.path.join(tools_src, item)
        dst_path = os.path.join(staging_dir, "tools", item)
        if os.path.isfile(src_path):
            with open(src_path, "r", encoding="utf-8") as f:
                tcontent = f.read()
            tcontent = tcontent.replace("v1.0.0", f"v{version}")
            with open(dst_path, "w", encoding="utf-8") as f:
                f.write(tcontent)

    output_nupkg = os.path.join(dist_dir, f"newscid.{version}.nupkg")

    # 3. If choco is installed, run native choco pack
    if shutil.which("choco"):
        print("--> Building with native 'choco pack'...")
        cmd = ["choco", "pack", nuspec_dest, f"--outputdirectory={dist_dir}"]
        subprocess.run(cmd, check=True)
    else:
        print("--> choco not detected; creating standard NuGet OPC .nupkg archive...")
        content_types_xml = """<?xml version="1.0" encoding="utf-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml" />
  <Default Extension="ps1" ContentType="application/octet-stream" />
  <Default Extension="txt" ContentType="text/plain" />
  <Default Extension="nuspec" ContentType="application/octet-stream" />
</Types>"""

        rels_xml = """<?xml version="1.0" encoding="utf-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Type="http://schemas.microsoft.com/packaging/2010/07/manifest" Target="/newscid.nuspec" Id="R1" />
</Relationships>"""

        with zipfile.ZipFile(output_nupkg, "w", zipfile.ZIP_DEFLATED) as zf:
            zf.writestr("[Content_Types].xml", content_types_xml)
            zf.writestr("_rels/.rels", rels_xml)
            zf.write(nuspec_dest, arcname="newscid.nuspec")
            for root, _, files in os.walk(os.path.join(staging_dir, "tools")):
                for file in files:
                    file_path = os.path.join(root, file)
                    rel_path = os.path.relpath(file_path, staging_dir)
                    zf.write(file_path, arcname=rel_path)

    size = os.path.getsize(output_nupkg)
    print(f"✔ Chocolatey package successfully built: {output_nupkg} ({size / 1024:.1f} KB)")

if __name__ == "__main__":
    main()
