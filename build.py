#!/usr/bin/env python3
import os
import shutil
import subprocess
import sys
from pathlib import Path

# root dir (where build.py lives)
ROOT = Path(__file__).resolve().parent

DEV_DIR = ROOT / "dev"            # where your Go code is
DEV_STATIC_DIR = DEV_DIR / "static"
DEV_USERS_DIR = DEV_DIR / "users"

BUILD_DIR = ROOT / "build"
BUILD_SERVER_DIR = BUILD_DIR / "server"
BUILD_STATIC_DIR = BUILD_DIR / "static"
BUILD_USERS_DIR = BUILD_DIR / "users"

def die(msg: str, code: int = 1):
    print(f"ERROR: {msg}", file=sys.stderr)
    sys.exit(code)

def which(cmd: str) -> bool:
    return shutil.which(cmd) is not None

def copy_contents(src: Path, dest: Path):
    if not src.exists():
        return
    dest.mkdir(parents=True, exist_ok=True)
    for item in src.iterdir():
        target = dest / item.name
        if item.is_dir():
            shutil.copytree(item, target, dirs_exist_ok=True)
        else:
            shutil.copy2(item, target)

def main():
    # 1) sanity
    if not which("go"):
        die("Go toolchain not found on PATH")
    if not DEV_DIR.is_dir():
        die(f"dev/ not found at {DEV_DIR}")

    # 2) clean + create build/ tree
    if BUILD_DIR.exists():
        shutil.rmtree(BUILD_DIR)
    BUILD_SERVER_DIR.mkdir(parents=True, exist_ok=True)
    BUILD_STATIC_DIR.mkdir(parents=True, exist_ok=True)
    BUILD_USERS_DIR.mkdir(parents=True, exist_ok=True)

    # 3) copy static + users from dev/
    if DEV_STATIC_DIR.is_dir():
        copy_contents(DEV_STATIC_DIR, BUILD_STATIC_DIR)
    else:
        print("WARN: dev/static not found")

    if DEV_USERS_DIR.is_dir():
        copy_contents(DEV_USERS_DIR, BUILD_USERS_DIR)
    else:
        print("WARN: dev/users not found")

    # 4) build Go → build/server/server
    out_bin = (BUILD_SERVER_DIR / "server").resolve()
    print(f"🔨 Building Go binary → {out_bin}")

    env = os.environ.copy()
    env["GOOS"] = "linux"
    env["GOARCH"] = "amd64"
    env["CGO_ENABLED"] = "0"

    # run go build in dev/ (because main.go is there)
    subprocess.run(
        [
            "go", "build",
            "-trimpath",
            "-ldflags=-s -w",
            "-o", str(out_bin),
            "."   # build the current module/package
        ],
        cwd=str(DEV_DIR),
        check=True,
        env=env,
    )

    print("✅ Build complete. Contents of build/:")
    for root, dirs, files in os.walk(BUILD_DIR):
        rel = os.path.relpath(root, BUILD_DIR)
        prefix = "." if rel == "." else rel
        print(prefix + "/")
        for d in sorted(dirs):
            print(os.path.join(prefix, d) + "/")
        for f in sorted(files):
            print(os.path.join(prefix, f))

if __name__ == "__main__":
    main()
