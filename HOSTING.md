# BC Ferries API - Hosting Documentation

## Current Setup (As of November 2025)

**API URL:** `http://192.18.154.57`

This API is hosted on **Oracle Cloud Always Free Tier** at **$0/month forever** (no domain, no HTTPS - personal use only).

### Quick Reference

- **API Base URL:** `http://192.18.154.57`
- **Health Check:** `http://192.18.154.57/healthcheck/`
- **All Routes:** `http://192.18.154.57/v2/`
- **Capacity Routes:** `http://192.18.154.57/v2/capacity/`
- **Non-capacity Routes:** `http://192.18.154.57/v2/noncapacity/`

### Server Details

- **Provider:** Oracle Cloud (Always Free Tier)
- **Region:** Toronto (ca-toronto-1)
- **Instance Type:** VM.Standard.E2.1.Micro
- **Resources:** 1 OCPU, 1GB RAM, 47GB boot volume
- **Public IP:** 192.18.154.57
- **Cost:** $0/month (Always Free - no trial, no expiration)
- **OS:** Ubuntu 24.04 LTS

---

## Architecture

```
Internet (HTTP only)
    ↓
Nginx Reverse Proxy (Port 80) - 192.18.154.57
    ↓
Docker Container - API (localhost:8080)
    ↓
Docker Container - PostgreSQL (localhost:5432, internal only)
```

**Components:**

- **Oracle Cloud VM** - Always Free tier compute instance
- **Docker + Docker Compose** - Containerized API and database
- **Nginx** - Reverse proxy (HTTP only, no SSL)
- **No Domain** - Using IP address directly
- **No SSL** - HTTP only (acceptable for public ferry schedule data)

---

## How It Works

### Docker Containers

The application runs in two Docker containers managed by `docker-compose.yml`:

1. **API Container (`api`):**

   - Built from `Dockerfile`
   - Go 1.23 application
   - Includes Chromium for web scraping
   - Exposes port 8080 internally
   - Environment variables from `.env`

2. **Database Container (`db`):**
   - PostgreSQL 13
   - Port 5432 (internal only, not exposed to internet)
   - Persistent storage via Docker volume `db_data`
   - Initialized with `init.sql` on first run

### Nginx Reverse Proxy

Nginx runs on the host (not in Docker) and:

- Listens on port 80 (public)
- Forwards all requests to `localhost:8080` (Docker API container)
- Handles all internet traffic

### Environment Configuration

The `.env` file on the server contains:

```env
DB_USER=bcferries_prod
DB_PASS=YzuLCYHPL0qHVnWmjZPzumr7Etp9XTuv
DB_NAME=bcferries
DB_HOST=db
DB_PORT=5432
DB_SSL=disable
```

**Note:** Password is stored here for reference. DB_SSL is disabled for internal Docker network communication.

---

## Accessing the Server

### SSH Connection

**Option 1: Direct connection**

```bash
ssh -i ~/.ssh/oracle-bc-ferries.key ubuntu@192.18.154.57
```

**Option 2: Using SSH config (recommended)**

If you've set up `~/.ssh/config` with:

```
Host bc-ferries-api
    HostName 192.18.154.57
    User ubuntu
    IdentityFile ~/.ssh/oracle-bc-ferries.key
```

Then you can simply use:

```bash
ssh bc-ferries-api
```

**Note:** SSH key is stored in `~/.ssh/` (NOT in the project directory) for security.

### SSH Key Security

**Why the key is NOT in this repository:**

- SSH keys are authentication credentials and should never be committed to git
- Even with `.gitignore`, accidental commits can expose keys in git history
- Keys belong in `~/.ssh/` (standard secure location with proper permissions)
- The `.gitignore` includes `*.key` and `*.pem` as a safety net, but keys should never be in the project directory

**Key permissions:**

```bash
# SSH keys must have restricted permissions
chmod 600 ~/.ssh/oracle-bc-ferries.key
chmod 600 ~/.ssh/config
```

---

## Deployment & Management

### Current Deployment Status

The API is currently deployed and running. Docker containers are managed by Docker Compose.

### Checking Status

```bash
# SSH into server
ssh bc-ferries-api

# Check container status
cd ~/bc-ferries-api
docker compose ps

# View API logs
docker compose logs -f api

# View database logs
docker compose logs -f db
```

### Deploying Updates

When code changes are pushed to GitHub:

```bash
# SSH into server
ssh bc-ferries-api

# Navigate to project
cd ~/bc-ferries-api

# use deploy script
./deploy.sh

That's it! The script will:
  - Pull latest from master
  - Stop containers
  - Rebuild and restart them
  - Show container status
  - Stream API logs (Ctrl+C to exit)

# Check logs for errors
docker compose logs -f api
```

**Deployment takes 2-5 minutes** (building Go app + pulling dependencies).

---

## Original Setup Process (For Reference)

This was a one-time setup. The server is now configured and running.

### Phase 1: Oracle Cloud Instance Creation

1. **Created VCN (Virtual Cloud Network):**

   - Used VCN Wizard with "Create VCN with Internet Connectivity"
   - Name: `bcferries-vcn`
   - CIDR: 10.0.0.0/16
   - Public subnet: 10.0.0.0/24

2. **Created Compute Instance:**

   - Name: `bc-ferries-api-free`
   - Region: Toronto (ca-toronto-1)
   - Availability Domain: AD-1
   - **Shape: VM.Standard.E2.1.Micro (Always Free)**
   - Image: Ubuntu 24.04
   - Networking: Selected `bcferries-vcn` with public subnet
   - Assigned public IPv4: 192.18.154.57
   - SSH key: Generated and downloaded

3. **Configured Security Lists:**
   - Added ingress rules for ports 22 (SSH), 80 (HTTP), 443 (HTTPS)

**Note:** Tried VM.Standard.A1.Flex (ARM - 4 OCPU + 24GB free) but got "Out of capacity" error. Used AMD Always Free instead.

---

### Phase 2: Server Configuration

#### 2.1 Connect to Server

```bash
ssh -i ~/.ssh/oracle-bc-ferries.key ubuntu@192.18.154.57
# Or: ssh bc-ferries-api (if SSH config is set up)
```

#### 2.2 Install Required Software

Uploaded and ran `oracle-cloud-setup.sh` script which attempted to install:

- Docker
- Nginx
- UFW firewall
- Fail2Ban
- Automatic security updates

**Issue:** Script failed due to UFW/iptables-persistent conflict.

**Manual completion:**

```bash
# Fix Oracle Cloud iptables (critical for allowing HTTP traffic)
sudo iptables -I INPUT 1 -j ACCEPT
sudo netfilter-persistent save

# Install Docker using convenience script
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker ubuntu

# Install Nginx
sudo apt install -y nginx
sudo systemctl enable nginx
sudo systemctl start nginx

# Install Fail2Ban
sudo apt install -y fail2ban
sudo systemctl enable fail2ban
sudo systemctl start fail2ban

# Exit and reconnect for docker group to take effect
exit
```

**Oracle Cloud Firewall Note:** Oracle's default iptables rules block all traffic except SSH. The `iptables -I INPUT 1 -j ACCEPT` command allows all traffic and was necessary for HTTP/HTTPS to work.

---

### Phase 3: Deploy API

#### 3.1 Clone Repository

```bash
cd ~
git clone https://github.com/jeffcstock/bc-ferries-api.git
cd bc-ferries-api
```

#### 3.2 Configure Environment

```bash
nano .env
```

Created `.env` file with:

```env
DB_USER=bcferries_prod
DB_PASS=YzuLCYHPL0qHVnWmjZPzumr7Etp9XTuv
DB_NAME=bcferries
DB_HOST=db
DB_PORT=5432
DB_SSL=disable
```

**Note:** `DB_HOST=db` works because Docker Compose creates an internal network where containers can reference each other by service name.

#### 3.3 Build and Start Containers

```bash
docker compose up -d --build
```

This command:

- Builds the Go application (from `Dockerfile`)
- Starts PostgreSQL database container
- Starts API container
- Takes 2-5 minutes on first run

#### 3.4 Verify Deployment

```bash
# Check container status
docker compose ps

# Test health check
curl http://localhost:8080/healthcheck/

# Test main endpoint
curl http://localhost:8080/v2/ | head -50
```

---

### Phase 4: Configure Nginx Reverse Proxy

#### 4.1 Create Nginx Configuration

```bash
sudo mkdir -p /etc/nginx/sites-available
sudo mkdir -p /etc/nginx/sites-enabled
sudo nano /etc/nginx/sites-available/bcferries-api
```

Created configuration file with:

```nginx
server {
    listen 80;
    listen [::]:80;

    server_name 192.18.154.57;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}
```

**Note:** Using IP address (192.18.154.57) instead of domain name. No SSL configured (HTTP only).

#### 4.2 Enable Configuration

```bash
sudo ln -s /etc/nginx/sites-available/bcferries-api /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

#### 4.3 Test Public Access

Visit `http://192.18.154.57/healthcheck/` in browser - should return: `{"status":"API is running"}`

---

## API is Live! 🎉

### Available Endpoints

```bash
# Health check
curl http://192.18.154.57/healthcheck/

# All routes
curl http://192.18.154.57/v2/

# Capacity routes only
curl http://192.18.154.57/v2/capacity/

# Non-capacity routes only
curl http://192.18.154.57/v2/noncapacity/

# V1 API example (specific route)
curl http://192.18.154.57/api/TSA/SWB
```

---

## Maintenance & Operations

### Viewing Logs

```bash
# API logs
docker-compose logs -f api

# Database logs
docker-compose logs -f db

# Nginx access logs
tail -f /var/log/nginx/access.log

# Nginx error logs
tail -f /var/log/nginx/error.log
```

### Updating Your API

When you push changes to your repository:

```bash
# SSH into server
ssh root@your-server-ip

# Navigate to project
cd /home/deploy/bc-ferries-api

# Pull latest changes
git pull

# Rebuild and restart containers
docker-compose down
docker-compose up -d --build

# Check logs for errors
docker-compose logs -f api
```

### Restarting Services

```bash
# Restart API only
docker-compose restart api

# Restart database (may cause brief downtime)
docker-compose restart db

# Restart all containers
docker-compose restart

# Restart Nginx
systemctl restart nginx
```

### Database Backups

**Create a backup script:**

```bash
# Create backup directory
mkdir -p /home/deploy/backups

# Create backup script
nano /home/deploy/backup-db.sh
```

**Paste:**

```bash
#!/bin/bash
BACKUP_DIR="/home/deploy/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DB_CONTAINER="bc-ferries-api-db-1"  # Check with: docker ps

docker exec $DB_CONTAINER pg_dump -U bcferries_prod bcferries > $BACKUP_DIR/backup_$TIMESTAMP.sql

# Keep only last 7 days of backups
find $BACKUP_DIR -name "backup_*.sql" -mtime +7 -delete

echo "Backup completed: backup_$TIMESTAMP.sql"
```

**Make executable and test:**

```bash
chmod +x /home/deploy/backup-db.sh
/home/deploy/backup-db.sh
```

**Setup daily backups with cron:**

```bash
# Edit crontab
crontab -e

# Add this line (runs daily at 2 AM)
0 2 * * * /home/deploy/backup-db.sh
```

### Restore from Backup

```bash
# Stop API container
docker-compose stop api

# Restore database
cat /home/deploy/backups/backup_TIMESTAMP.sql | docker exec -i bc-ferries-api-db-1 psql -U bcferries_prod bcferries

# Restart API
docker-compose start api
```

---

## Monitoring & Performance

### Check Resource Usage

```bash
# Overall system stats
htop  # Install with: apt install htop

# Docker container stats
docker stats

# Disk usage
df -h

# Check memory
free -h
```

### Optional: Setup Monitoring

**Simple uptime monitoring (free):**

- [UptimeRobot](https://uptimerobot.com) - Pings your API every 5 minutes
- [StatusCake](https://www.statuscake.com) - Free tier available

**Add health check URL:** `https://api.yourdomain.com/healthcheck/`

### Rate Limiting (Optional)

If you want to prevent abuse, add to your Nginx config:

```nginx
# Add before server block
limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;

# Inside server block
location / {
    limit_req zone=api_limit burst=20 nodelay;

    # ... rest of proxy_pass config
}
```

This limits each IP to 10 requests per second with bursts up to 20.

---

## Security Best Practices

### ✅ Checklist

- [ ] Use strong database passwords
- [ ] Enable `DB_SSL=require` in production
- [ ] Keep system updated: `apt update && apt upgrade` monthly
- [ ] Use SSH keys instead of passwords
- [ ] Configure firewall (ufw) to only allow ports 22, 80, 443
- [ ] Regular database backups
- [ ] Monitor disk space usage
- [ ] Use fail2ban to prevent brute force SSH attempts

### Optional: Install Fail2Ban

```bash
# Install fail2ban
apt install fail2ban -y

# Enable and start
systemctl enable fail2ban
systemctl start fail2ban

# Check status
fail2ban-client status sshd
```

---

## Troubleshooting

### API Not Responding

```bash
# Check if containers are running
docker-compose ps

# Check API logs
docker-compose logs api

# Check if port 8080 is listening
netstat -tlnp | grep 8080

# Restart containers
docker-compose restart
```

### Database Connection Issues

```bash
# Check database logs
docker-compose logs db

# Verify database is running
docker-compose ps db

# Connect to database manually
docker exec -it bc-ferries-api-db-1 psql -U bcferries_prod bcferries

# Inside psql, check tables
\dt
```

### Nginx Issues

```bash
# Check Nginx status
systemctl status nginx

# Test configuration
nginx -t

# Check error logs
tail -50 /var/log/nginx/error.log

# Restart Nginx
systemctl restart nginx
```

### Disk Space Full

```bash
# Check disk usage
df -h

# Check Docker disk usage
docker system df

# Clean up old Docker images/containers
docker system prune -a

# Clean up old logs
journalctl --vacuum-time=7d
```

---

## Summary

**What This Setup Provides:**

- ✅ Free hosting forever (Oracle Cloud Always Free)
- ✅ HTTP API accessible at http://192.18.154.57
- ✅ Database with persistent storage
- ✅ Nginx reverse proxy
- ✅ Docker containerization
- ✅ ~99.9% uptime (Oracle Cloud dependent)

**Total Cost: $0/month**

**Maintenance: Manual deployment when code changes**

---

## Important Notes

### Why No HTTPS/SSL?

- No domain name (using IP address)
- Personal use only (not public-facing)
- Data is public ferry schedules (no sensitive data)
- Let's Encrypt doesn't issue certificates for IP addresses

### Oracle Cloud Always Free Limitations

**What's Free Forever:**

- 2 AMD VMs (1 OCPU + 1GB RAM each) OR
- 4 Arm-based Ampere A1 cores + 24GB RAM total
- 2 Block Volumes (200GB total)
- 10GB Object Storage
- 10TB outbound data transfer/month

**Current Usage:**

- 1 AMD VM (VM.Standard.E2.1.Micro)
- 1 OCPU + 1GB RAM
- Well within Always Free limits

### Performance with 1GB RAM

The 1GB RAM should handle:

- Light to moderate API traffic (hundreds of requests/day)
- Go application is lightweight
- PostgreSQL works but is memory-constrained
- Web scraping with chromedp runs but may be slow

If you need more performance, try getting an ARM instance (4 OCPU + 24GB) which is also Always Free but has capacity issues.

---

## LLM Context Summary

**For AI assistants reading this file:**

This is a BC Ferries API running on Oracle Cloud's Always Free tier. Key facts:

1. **Access:** `http://192.18.154.57` (HTTP only, no SSL)
2. **Server:** Ubuntu 24.04 on VM.Standard.E2.1.Micro (1 OCPU + 1GB RAM)
3. **Location:** Toronto (ca-toronto-1), Oracle Cloud
4. **Deployment:** Docker Compose with two containers (Go API + PostgreSQL 13)
5. **Reverse Proxy:** Nginx forwards port 80 → localhost:8080
6. **Cost:** $0/month (Always Free tier, no expiration)
7. **SSH:** `ssh bc-ferries-api` (or `ssh -i ~/.ssh/oracle-bc-ferries.key ubuntu@192.18.154.57`)
8. **Code Location:** `~/bc-ferries-api` on server
9. **Deployment:** `ssh bc-ferries-api`, then `cd ~/bc-ferries-api && git pull && docker compose down && docker compose up -d --build`
10. **SSH Key:** Stored in `~/.ssh/oracle-bc-ferries.key` (NOT in repo)

**Architecture Flow:**

```
Internet → 192.18.154.57:80 (Nginx) → localhost:8080 (Docker API) → localhost:5432 (Docker PostgreSQL)
```

The setup prioritizes zero cost over performance. Database password and server details are documented in this file for easy reference. SSH key is kept outside the repository for security.

## Commands

- Connect to server:
  ssh bc-ferries-api

- Deploy updates:
  ssh bc-ferries-api
  cd ~/bc-ferries-api
  git pull && docker compose down && docker compose up -d --build

- Check status:
  ssh bc-ferries-api
  cd ~/bc-ferries-api
  docker compose ps
