#!/usr/bin/env python3
import subprocess
import sys
from pathlib import Path

EC2_HOST = "ec2-user@54.183.19.220"
SSH_KEY = Path("~/.ssh/poker_key.pem").expanduser()
IMAGE_NAME = "poker:latest"
CONTAINER_NAME = "poker"


def run(cmd: str):
    print(f"\n>>> {cmd}")
    subprocess.run(cmd, shell=True, check=True)


def main():
    # 1) build your app locally
    run(f"{sys.executable} build.py")

    # 2) make server binary executable
    run("chmod +x build/server/server")

    # 3) build docker image locally — delete old local poker:latest if present
    run(
        f"docker image inspect {IMAGE_NAME} >/dev/null 2>&1 && docker rmi {IMAGE_NAME}; "
        f"docker buildx build --platform linux/amd64 -t {IMAGE_NAME} ."
    )

    # 4) make sure docker is installed & running on EC2
    run(
        f"ssh -i {SSH_KEY} {EC2_HOST} "
        "'sudo dnf install -y docker && sudo systemctl start docker && sudo systemctl enable docker'"
    )

    # 5) send image to EC2 (remove remote image first, then load)
    run(
        f"docker save {IMAGE_NAME} | bzip2 | "
        f"ssh -i {SSH_KEY} {EC2_HOST} "
        "'sudo docker rmi -f poker:latest 2>/dev/null || true && bunzip2 | sudo docker load'"
    )

    # 6) stop old container (named 'poker') and run new one
    remote_cmd = (
        f"sudo docker rm -f {CONTAINER_NAME} 2>/dev/null || true && "
        f"sudo docker run -d --name {CONTAINER_NAME} --restart=always -p 80:8080 {IMAGE_NAME}"
    )
    run(
        f"ssh -i {SSH_KEY} {EC2_HOST} '{remote_cmd}'"
    )

    print("\n✅ Deploy complete.")


if __name__ == "__main__":
    main()
