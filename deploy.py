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
    # 1) run your local build.py first
    run(f"{sys.executable} build.py")

    # 2) make sure your built server binary is executable
    # adjust the path if yours is different
    run("chmod +x build/server/server")

    # 3) build docker image locally — delete old poker:latest if it exists
    run(
        f"docker image inspect {IMAGE_NAME} >/dev/null 2>&1 && docker rmi {IMAGE_NAME}; "
        f"docker buildx build --platform linux/amd64 -t {IMAGE_NAME} ."
    )

    # 4) ssh into EC2 to install and run docker
    run(
        f"ssh -i {SSH_KEY} {EC2_HOST} "
        "'sudo dnf install -y docker && sudo systemctl start docker && sudo systemctl enable docker'"
    )

    # 5) load image into EC2 (remove remote poker:latest first, then load stream)
    run(
        f"docker save {IMAGE_NAME} | bzip2 | "
        f"ssh -i {SSH_KEY} {EC2_HOST} "
        "'sudo docker rmi -f poker:latest 2>/dev/null || true && bunzip2 | sudo docker load'"
    )

    # 6) run image on EC2
    run(
        f"ssh -i {SSH_KEY} {EC2_HOST} "
        "'sudo docker rm -f {c} 2>/dev/null || true && "
        f"sudo docker run -d --name {CONTAINER_NAME} --restart=always -p 80:8080 {IMAGE_NAME}'"
    )

    print("\n✅ Deploy complete.")


if __name__ == "__main__":
    main()
