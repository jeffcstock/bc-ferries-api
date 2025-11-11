#!/bin/bash
#
# BC Ferries API - Oracle Cloud Instance Setup Script
# This script configures security, installs Docker, Nginx, and prepares the server
#

set -e

echo "=================================================="
echo "BC Ferries API - Oracle Cloud Setup"
echo "=================================================="
echo ""

# Check if running as ubuntu user
if [ "$USER" != "ubuntu" ]; then
    echo "Error: This script must be run as the 'ubuntu' user"
    exit 1
fi

echo "Step 1: Updating system packages..."
sudo apt update
sudo apt upgrade -y

echo ""
echo "Step 2: Installing essential packages..."
sudo apt install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    software-properties-common \
    fail2ban \
    unattended-upgrades \
    git \
    htop

# Install ufw separately (conflicts with netfilter-persistent)
sudo apt install -y ufw

# Install iptables-persistent for saving iptables rules
echo iptables-persistent iptables-persistent/autosave_v4 boolean true | sudo debconf-set-selections
echo iptables-persistent iptables-persistent/autosave_v6 boolean true | sudo debconf-set-selections
sudo apt install -y iptables-persistent

echo ""
echo "Step 3: Configuring UFW Firewall..."
# Reset UFW to clean state
sudo ufw --force reset

# Set default policies
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH (critical!)
sudo ufw allow 22/tcp
sudo ufw allow OpenSSH

# Allow HTTP and HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Enable UFW
sudo ufw --force enable

echo "UFW Status:"
sudo ufw status verbose

echo ""
echo "Step 4: Fixing Oracle Cloud iptables rules..."
# Oracle Cloud blocks traffic by default at OS level
# We need to allow traffic through iptables
sudo iptables -I INPUT 1 -j ACCEPT
sudo netfilter-persistent save 2>/dev/null || sudo iptables-save | sudo tee /etc/iptables/rules.v4 > /dev/null

echo ""
echo "Step 5: Installing Docker..."
# Add Docker's official GPG key
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

# Add Docker repository
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Add ubuntu user to docker group
sudo usermod -aG docker ubuntu

# Enable Docker service
sudo systemctl enable docker
sudo systemctl start docker

echo "Docker version:"
docker --version

echo ""
echo "Step 6: Installing Nginx..."
sudo apt install -y nginx

# Enable and start Nginx
sudo systemctl enable nginx
sudo systemctl start nginx

echo "Nginx version:"
nginx -v

echo ""
echo "Step 7: Configuring Fail2Ban..."
sudo systemctl enable fail2ban
sudo systemctl start fail2ban

# Create SSH jail configuration
sudo tee /etc/fail2ban/jail.d/sshd.local > /dev/null <<EOF
[sshd]
enabled = true
port = 22
filter = sshd
logpath = /var/log/auth.log
maxretry = 5
bantime = 3600
findtime = 600
EOF

sudo systemctl restart fail2ban

echo "Fail2Ban status:"
sudo fail2ban-client status

echo ""
echo "Step 8: Enabling automatic security updates..."
sudo dpkg-reconfigure -plow unattended-upgrades

echo ""
echo "Step 9: Disabling Oracle Cloud Agent firewall management..."
sudo mkdir -p /etc/systemd/system/oracle-cloud-agent.service.d/
sudo tee /etc/systemd/system/oracle-cloud-agent.service.d/override.conf > /dev/null <<EOF
[Service]
Environment="OCA_DISABLE=true"
EOF

sudo systemctl daemon-reload
sudo systemctl restart oracle-cloud-agent 2>/dev/null || true

echo ""
echo "=================================================="
echo "Setup Complete!"
echo "=================================================="
echo ""
echo "Installed:"
echo "  ✓ Docker $(docker --version | awk '{print $3}')"
echo "  ✓ Nginx $(nginx -v 2>&1 | awk '{print $3}')"
echo "  ✓ UFW Firewall (enabled)"
echo "  ✓ Fail2Ban (monitoring SSH)"
echo "  ✓ Automatic security updates"
echo ""
echo "Next Steps:"
echo "  1. Log out and log back in for Docker group to take effect"
echo "  2. Clone your BC Ferries API repository"
echo "  3. Configure environment variables"
echo "  4. Deploy with docker compose"
echo ""
echo "IMPORTANT: Run 'exit' and then reconnect with SSH"
echo ""
