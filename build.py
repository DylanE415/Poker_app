#!/usr/bin/env python3
# Requires Python 3.8+ and the Go toolchain (for build step)
import argparse
import os
import shutil
import subprocess
import sys
import stat
from pathlib import Path

# ====== Build defaults (env override, then CLI) ======
DEF_GOOS        = os.environ.get("GOOS", "linux")
DEF_GOARCH      = os.environ.get("GOARCH", "amd64")
DEF_CGO_ENABLED = os.environ.get("CGO_ENABLED", "0")
DEF_BIN_NAME    = os.environ.get("BIN_NAME", "server")

SERVER_DIR        = Path(os.environ.get("SERVER_DIR", "server"))
SERVER_STATIC_DIR = Path(os.environ.get("SERVER_STATIC_DIR", "server/static"))
USERS_DIR         = Path(os.environ.get("USERS_DIR", "users"))
BUILD_DIR         = Path(os.environ.get("BUILD_DIR", "build"))

# ====== Deploy defaults (can override via flags) ======
DEF_IP        = "54.183.19.220"
DEF_USER      = "ec2-user"
DEF_SERVICE   = "poker"
DEF_KEY_HINTS = ["poker_key.pem", "my-ec2-key.pem"]  # try these first in ~/.ssh

def which(cmd: str) -> bool:
    return shutil.which(cmd) is not None

def die(msg: str, code: int = 1):
    print(f"ERROR: {msg}", file=sys.stderr)
    sys.exit(code)

def warn(msg: str):
    print(f"WARN: {msg}")

def pick_pem_in_ssh(hints=None) -> str:
    ssh_dir = Path.home() / ".ssh"
    if not ssh_dir.is_dir():
        die(f"{ssh_dir} not found; create it and put your .pem there.")
    cands = []
    # prioritize hinted names if present
    hints = hints or []
    for name in hints:
        p = ssh_dir / name
        if p.is_file():
            cands.append(p)
    # then any *.pem
    for p in sorted(ssh_dir.glob("*.pem")):
        if p not in cands:
            cands.append(p)
    if not cands:
        die(f"No .pem keys found in {ssh_dir}")
    if len(cands) == 1:
        return str(cands[0])
    print("Multiple .pem keys found in ~/.ssh. Choose one:")
    for i, p in enumerate(cands, 1):
        print(f"  {i}) {p.name}")
    while True:
        sel = input(f"Select [1-{len(cands)}]: ").strip()
        if sel.isdigit() and 1 <= int(sel) <= len(cands):
            return str(cands[int(sel)-1])
        print("Invalid selection, try again.")

def ensure_strict_permissions(pem_path: str):
    if os.name == "posix":
        st = os.stat(pem_path)
        desired = stat.S_IRUSR
        if (st.st_mode & 0o777) != desired:
            try:
                os.chmod(pem_path, desired)
                print("🔒 Fixed key permissions to 0400.")
            except Exception as e:
                print(f"⚠️ Could not set 0400 permissions: {e}")

def copy_contents(src: Path, dest: Path):
    if not src.exists():
        return
    dest.mkdir(parents=True, exist_ok=True)
    for item in src.iterdir():
        dpath = dest / item.name
        if item.is_dir():
            shutil.copytree(item, dpath, dirs_exist_ok=True)
        else:
            shutil.copy2(item, dpath)

def print_layout(build_dir: Path):
    if which("tree"):
        subprocess.run(["tree", "-a", str(build_dir)], check=False)
        return
    for root, dirs, files in os.walk(build_dir):
        rel = os.path.relpath(root, build_dir)
        prefix = "." if rel == "." else rel
        print(prefix + "/")
        for d in sorted(dirs):
            print(f"{os.path.join(prefix, d)}/")
        for f in sorted(files):
            print(os.path.join(prefix, f))

def run(cmd: list[str], check=True):
    # stream output; raise on non-zero if check=True
    proc = subprocess.Popen(cmd)
    ret = proc.wait()
    if check and ret != 0:
        raise SystemExit(ret)
    return ret

def go_build(goos: str, goarch: str, cgo_enabled: str, bin_name: str) -> Path:
    if not which("go"):
        die("Go toolchain not found on PATH")
    if not SERVER_DIR.is_dir():
        die(f"{SERVER_DIR} not found")
    if not SERVER_STATIC_DIR.is_dir():
        warn(f"{SERVER_STATIC_DIR} not found (static will be empty)")
    if not USERS_DIR.is_dir():
        warn(f"{USERS_DIR} not found (users will be empty)")

    if BUILD_DIR.exists():
        shutil.rmtree(BUILD_DIR)
    (BUILD_DIR / "static").mkdir(parents=True, exist_ok=True)
    (BUILD_DIR / "users").mkdir(parents=True, exist_ok=True)
    (BUILD_DIR / "server").mkdir(parents=True, exist_ok=True)

    if SERVER_STATIC_DIR.is_dir():
        copy_contents(SERVER_STATIC_DIR, BUILD_DIR / "static")
    if USERS_DIR.is_dir():
        copy_contents(USERS_DIR, BUILD_DIR / "users")

    bin_ext = "" if goos != "windows" else ".exe"
    out_path = BUILD_DIR / "server" / f"{bin_name}{bin_ext}"
    print(f"🔨 Building {goos}/{goarch} (CGO_ENABLED={cgo_enabled}) -> {out_path}")

    env = os.environ.copy()
    env["GOOS"] = goos
    env["GOARCH"] = goarch
    env["CGO_ENABLED"] = cgo_enabled

    subprocess.run(
        ["go", "build", "-trimpath", "-ldflags=-s -w", "-o", str(out_path)],
        cwd=str(SERVER_DIR),
        check=True,
        env=env,
    )

    print("📦 Build layout:")
    print_layout(BUILD_DIR)
    return out_path

def scp_build_contents(pem: str, user: str, host: str, remote_app_dir: str):
    # ensure remote dir exists
    run(["ssh", "-i", pem, "-o", "StrictHostKeyChecking=accept-new",
         f"{user}@{host}", f"mkdir -p {remote_app_dir}"], check=True)

    # upload contents of build/ one-by-one to preserve layout without nesting build/
    print("⬆️  Uploading build/ contents ...")
    for item in sorted(BUILD_DIR.iterdir(), key=lambda p: p.name):
        src = str(item)
        dst = f"{user}@{host}:{remote_app_dir}/"
        # always use -r to handle dirs; scp ignores -r for files
        run(["scp", "-r", "-i", pem, "-o", "StrictHostKeyChecking=accept-new", src, dst], check=True)

def main():
    parser = argparse.ArgumentParser(description="Build & deploy Poker app to EC2.")
    parser.add_argument("--os",   choices=["linux", "darwin", "windows"], default=DEF_GOOS)
    parser.add_argument("--arch", choices=["amd64", "arm64"], default=DEF_GOARCH)
    parser.add_argument("--name", default=DEF_BIN_NAME)
    parser.add_argument("--cgo",  default=DEF_CGO_ENABLED)
    parser.add_argument("--ip",   default=DEF_IP, help="EC2 public IP")
    parser.add_argument("--user", default=DEF_USER, help="SSH username")
    parser.add_argument("--service", default=DEF_SERVICE, help="systemd service name")
    parser.add_argument("--remote-app-dir", default="/home/ec2-user/app", help="Remote app dir")
    parser.add_argument("--key", default=None, help="Override pem filename in ~/.ssh (optional)")
    args = parser.parse_args()

    # 1) Build
    out_bin = go_build(args.os, args.arch, args.cgo, args.name)

    # 2) SSH key
    pem = (Path.home() / ".ssh" / args.key).as_posix() if args.key else pick_pem_in_ssh(DEF_KEY_HINTS)
    ensure_strict_permissions(pem)
    target = f"{args.user}@{args.ip}"
    remote_app_dir = args.remote_app_dir
    remote_bin = f"{remote_app_dir}/server/{args.name}"

    print(f"\n🔗 Using key: {pem}")
    print(f"🎯 Target: {target}")
    print(f"📁 Remote app dir: {remote_app_dir}\n")

    # 3) Stop/disable service before replacing files
    print(f"🛑 Disabling & stopping systemd service: {args.service}")
    run(["ssh", "-i", pem, "-o", "StrictHostKeyChecking=accept-new", target,
         f"sudo systemctl disable --now {args.service} || true"], check=False)

    # 4) Upload build/ contents
    scp_build_contents(pem, args.user, args.ip, remote_app_dir)

    # 5) Post-copy remote commands
    post_cmds = [
        f"sudo setcap 'cap_net_bind_service=+ep' {remote_bin}",
        f"ln -sfn {remote_app_dir}/users /home/{args.user}/users",
        f"chmod +x {remote_bin}",
        "sudo systemctl daemon-reload",
        f"sudo systemctl enable --now {args.service}",
    ]
    joined = " && ".join(post_cmds)
    print("▶️  Finalizing on remote (capabilities, symlink, permissions, systemd)...")
    run(["ssh", "-i", pem, "-o", "StrictHostKeyChecking=accept-new", target, joined], check=True)

    print("\n✅ Deploy complete.")

if __name__ == "__main__":
    try:
        main()
    except subprocess.CalledProcessError as e:
        die(f"Command failed with exit code {e.returncode}")
    except KeyboardInterrupt:
        print("\nAborted by user.")
