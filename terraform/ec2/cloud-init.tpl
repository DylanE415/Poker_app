#cloud-config
package_update: true
packages:
  - curl
  - unzip
  - jq
  - ca-certificates
runcmd:
  - |
    set -euo pipefail
    BUCKET="${bucket}"
    KEY="${key}"
    BIN_PATH="${binary_path}"
    SERVICE="${service_name}"
    APP_PORT="${app_port}"

    # Install AWS CLI v2 if not present
    if ! command -v aws >/dev/null 2>&1; then
      curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "/tmp/awscliv2.zip"
      unzip -o /tmp/awscliv2.zip -d /tmp
      /tmp/aws/install || true
    fi

    # Download binary from S3 using instance profile credentials
    /usr/local/bin/aws s3 cp "s3://${BUCKET}/${KEY}" "${BIN_PATH}"
    chmod +x "${BIN_PATH}"

    # Create systemd unit for the binary
    cat > /etc/systemd/system/${SERVICE}.service <<'EOF'
[Unit]
Description=${SERVICE}
After=network.target

[Service]
Type=simple
ExecStart=${BIN_PATH}
Restart=on-failure
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable --now ${SERVICE}.service || true

    # Install Caddy (Debian/Ubuntu)
    if ! command -v caddy >/dev/null 2>&1; then
      curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | apt-key add - || true
      curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' \
        | tee /etc/apt/sources.list.d/caddy-stable.list
      apt-get update
      apt-get install -y caddy || true
    fi

    # Write Caddyfile to reverse proxy to the app
    cat >/etc/caddy/Caddyfile <<CADDY
:80 {
  reverse_proxy 127.0.0.1:${APP_PORT}
}
CADDY

    systemctl restart caddy || true
