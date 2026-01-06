# Deployment Guide

This guide explains how to deploy the OCPP Emulator to an Ubuntu server using GitHub Actions.

## Overview

The project includes three deployment options:

1. **Automated GitHub Actions** - CI/CD pipeline that deploys on tag push or manual trigger
2. **Manual Script** - Shell script for deploying from release or local builds
3. **Manual Installation** - Step-by-step manual deployment

## Prerequisites

### Server Requirements

- Ubuntu 20.04+ (or Debian-based distribution)
- MongoDB 7.0+ installed and running
- Nginx (for reverse proxy)
- SSH access with sudo privileges

### GitHub Repository Secrets

Configure these secrets in your repository settings (**Settings → Secrets and variables → Actions**):

| Secret | Description | Example |
|--------|-------------|---------|
| `SSH_PRIVATE_KEY` | SSH private key for server access | `-----BEGIN OPENSSH PRIVATE KEY-----...` |
| `SSH_USER` | SSH username | `deploy` |
| `SSH_HOST` | Server hostname or IP | `your-server.com` or `192.168.1.100` |

---

## Option 1: GitHub Actions Deployment

### Setting Up SSH Access

1. **Generate a deployment SSH key** (on your local machine):

   ```bash
   ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/deploy_key
   ```

2. **Add public key to server** (on the Ubuntu server):

   ```bash
   # Create deploy user (recommended)
   sudo useradd -m -s /bin/bash deploy
   sudo usermod -aG sudo deploy

   # Allow passwordless sudo for deploy user
   echo "deploy ALL=(ALL) NOPASSWD:ALL" | sudo tee /etc/sudoers.d/deploy

   # Add SSH key
   sudo mkdir -p /home/deploy/.ssh
   sudo nano /home/deploy/.ssh/authorized_keys
   # Paste the content of deploy_key.pub

   sudo chown -R deploy:deploy /home/deploy/.ssh
   sudo chmod 700 /home/deploy/.ssh
   sudo chmod 600 /home/deploy/.ssh/authorized_keys
   ```

3. **Configure GitHub Secrets**:

   Go to **Repository → Settings → Secrets and variables → Actions → New repository secret**

   - `SSH_PRIVATE_KEY`: Content of `~/.ssh/deploy_key` (private key)
   - `SSH_USER`: `deploy`
   - `SSH_HOST`: Your server hostname or IP

### Triggering Deployment

**Method 1: Tag Push**

```bash
git tag deploy-v1.0.0
git push origin deploy-v1.0.0
```

**Method 2: Manual Trigger**

1. Go to **Actions → Deploy to Ubuntu Server**
2. Click **Run workflow**
3. Select environment (production/staging)
4. Click **Run workflow**

### Deployment Workflow

The workflow performs these steps:

1. Builds Go binary for linux-amd64
2. Builds frontend with npm
3. Creates deployment package
4. Copies files to server via SCP
5. Stops existing service
6. Installs binary to `/usr/local/bin/ocpp-emu`
7. Deploys web files to `/var/www/html/ocpp-emu`
8. Installs/updates systemd service
9. Starts service and verifies health

---

## Option 2: Manual Script Deployment

### Download and Run

```bash
# Download deploy scripts from release
curl -LO https://github.com/ruslan-hut/ocpp-emu/releases/latest/download/ocpp-emu-deploy-scripts.tar.gz
tar -xzf ocpp-emu-deploy-scripts.tar.gz
cd deploy-scripts

# Deploy specific version
sudo ./deploy-ubuntu.sh --version v1.0.0

# Or deploy local builds
sudo ./deploy-ubuntu.sh --binary /path/to/ocpp-emu --web /path/to/web/dist
```

### Script Options

```bash
./deploy-ubuntu.sh [OPTIONS]

Options:
  --version VERSION    Version to download (e.g., v1.0.0)
  --binary PATH        Path to local binary
  --web PATH           Path to local web dist directory
  --config PATH        Path to custom config file
  --skip-service       Don't install/restart systemd service
  --skip-web           Don't deploy web files
  --uninstall          Remove installation
  -h, --help           Show help
```

### Examples

```bash
# Full deployment from release
sudo ./deploy-ubuntu.sh --version v1.0.0

# Update binary only
sudo ./deploy-ubuntu.sh --version v1.0.0 --skip-web

# Deploy with custom config
sudo ./deploy-ubuntu.sh --version v1.0.0 --config /path/to/my-config.yml

# Uninstall
sudo ./deploy-ubuntu.sh --uninstall
```

---

## Option 3: Manual Installation

### 1. Install Dependencies

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install MongoDB
curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc | sudo gpg -o /usr/share/keyrings/mongodb-server-7.0.gpg --dearmor
echo "deb [ signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/7.0 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-7.0.list
sudo apt update
sudo apt install -y mongodb-org
sudo systemctl enable mongod
sudo systemctl start mongod

# Install Nginx
sudo apt install -y nginx
```

### 2. Create Service User

```bash
sudo useradd --system --no-create-home --shell /bin/false ocpp-emu
```

### 3. Create Directories

```bash
sudo mkdir -p /etc/conf
sudo mkdir -p /var/www/html/ocpp-emu
sudo mkdir -p /var/log/ocpp-emu
sudo chown ocpp-emu:ocpp-emu /var/log/ocpp-emu
```

### 4. Download and Install Binary

```bash
# Download binary
VERSION="v1.0.0"
curl -LO "https://github.com/ruslan-hut/ocpp-emu/releases/download/${VERSION}/ocpp-emu-linux-amd64"

# Install
sudo mv ocpp-emu-linux-amd64 /usr/local/bin/ocpp-emu
sudo chmod 755 /usr/local/bin/ocpp-emu
sudo chown root:root /usr/local/bin/ocpp-emu
```

### 5. Install Configuration

```bash
# Download config
curl -LO "https://github.com/ruslan-hut/ocpp-emu/releases/download/${VERSION}/ocpp-emu-configs.tar.gz"
tar -xzf ocpp-emu-configs.tar.gz

# Install config
sudo cp package/configs/config.yaml /etc/conf/ocpp-emu.yml
sudo chmod 640 /etc/conf/ocpp-emu.yml
sudo chown root:ocpp-emu /etc/conf/ocpp-emu.yml

# Edit config
sudo nano /etc/conf/ocpp-emu.yml
```

### 6. Install Web Files

```bash
# Download web files
curl -LO "https://github.com/ruslan-hut/ocpp-emu/releases/download/${VERSION}/ocpp-emu-web.tar.gz"

# Extract to web directory
sudo tar -xzf ocpp-emu-web.tar.gz -C /var/www/html/ocpp-emu
sudo chown -R www-data:www-data /var/www/html/ocpp-emu
sudo chmod -R 755 /var/www/html/ocpp-emu
```

### 7. Create Systemd Service

```bash
sudo tee /etc/systemd/system/ocpp-emu.service << 'EOF'
[Unit]
Description=OCPP Charging Station Emulator
Documentation=https://github.com/ruslan-hut/ocpp-emu
After=network.target mongodb.service
Wants=mongodb.service

[Service]
Type=simple
User=ocpp-emu
Group=ocpp-emu
WorkingDirectory=/usr/local/bin
ExecStart=/usr/local/bin/ocpp-emu --config /etc/conf/ocpp-emu.yml
Restart=on-failure
RestartSec=5s
TimeoutStopSec=30

NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/log/ocpp-emu

Environment=GIN_MODE=release

StandardOutput=journal
StandardError=journal
SyslogIdentifier=ocpp-emu

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable ocpp-emu
sudo systemctl start ocpp-emu
```

### 8. Configure Nginx

```bash
sudo tee /etc/nginx/sites-available/ocpp-emu << 'EOF'
server {
    listen 80;
    server_name _;

    root /var/www/html/ocpp-emu;
    index index.html;

    gzip on;
    gzip_types text/plain text/css application/json application/javascript;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;
    }
}
EOF

sudo ln -sf /etc/nginx/sites-available/ocpp-emu /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx
```

---

## Post-Deployment

### Verify Installation

```bash
# Check service status
sudo systemctl status ocpp-emu

# Check logs
sudo journalctl -u ocpp-emu -f

# Test API
curl http://localhost:8080/api/health

# Test web app
curl http://localhost/
```

### Update Configuration

```bash
# Edit config
sudo nano /etc/conf/ocpp-emu.yml

# Restart service
sudo systemctl restart ocpp-emu
```

### Important Configuration Changes

Edit `/etc/conf/ocpp-emu.yml`:

```yaml
auth:
  enabled: true
  jwt_secret: "generate-a-secure-random-string-at-least-32-chars"
  users:
    - username: "admin"
      # Generate new hash: go run cmd/tools/hashpw/main.go "your-password"
      password_hash: "$2a$10$YOUR_NEW_HASH_HERE"
      role: "admin"
      enabled: true
```

### Enable HTTPS (Recommended)

```bash
# Install Certbot
sudo apt install -y certbot python3-certbot-nginx

# Get certificate
sudo certbot --nginx -d your-domain.com

# Auto-renewal is configured automatically
```

---

## Troubleshooting

### Service Won't Start

```bash
# Check logs
sudo journalctl -u ocpp-emu -n 50 --no-pager

# Verify config
sudo -u ocpp-emu /usr/local/bin/ocpp-emu --config /etc/conf/ocpp-emu.yml

# Check MongoDB
sudo systemctl status mongod
```

### Permission Issues

```bash
# Fix log directory permissions
sudo chown -R ocpp-emu:ocpp-emu /var/log/ocpp-emu

# Fix config permissions
sudo chown root:ocpp-emu /etc/conf/ocpp-emu.yml
sudo chmod 640 /etc/conf/ocpp-emu.yml
```

### WebSocket Connection Issues

```bash
# Check nginx config
sudo nginx -t

# Verify proxy settings in nginx include WebSocket headers
# proxy_set_header Upgrade $http_upgrade;
# proxy_set_header Connection "upgrade";
```

### GitHub Actions Deployment Fails

1. **SSH Connection Failed**
   - Verify `SSH_HOST` is reachable from GitHub Actions
   - Ensure `SSH_PRIVATE_KEY` is correctly formatted (include full key with headers)
   - Check that the server allows SSH connections

2. **Permission Denied**
   - Verify deploy user has sudo access
   - Check `/etc/sudoers.d/deploy` exists
   - Ensure public key is in `/home/deploy/.ssh/authorized_keys`

3. **Service Start Failed**
   - Check MongoDB is running on server
   - Verify config file syntax

---

## File Locations

| Component | Path |
|-----------|------|
| Binary | `/usr/local/bin/ocpp-emu` |
| Config | `/etc/conf/ocpp-emu.yml` |
| Web App | `/var/www/html/ocpp-emu` |
| Systemd Service | `/etc/systemd/system/ocpp-emu.service` |
| Nginx Config | `/etc/nginx/sites-available/ocpp-emu` |
| Logs | `journalctl -u ocpp-emu` |

## Service Commands

```bash
# Status
sudo systemctl status ocpp-emu

# Start/Stop/Restart
sudo systemctl start ocpp-emu
sudo systemctl stop ocpp-emu
sudo systemctl restart ocpp-emu

# View logs
sudo journalctl -u ocpp-emu -f

# View last 100 lines
sudo journalctl -u ocpp-emu -n 100 --no-pager
```
