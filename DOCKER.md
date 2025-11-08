# Docker Setup Guide

This guide explains how to run the OCPP Emulator using Docker and Docker Compose.

## Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- At least 2GB free disk space
- Ports 3000, 8080, and 27017 available

## Quick Start

### 1. Build and Start All Services

```bash
docker-compose up --build
```

This will start:
- **MongoDB** on port 27017
- **Backend** on port 8080
- **Frontend** on port 3000

### 2. Access the Application

Open your browser and navigate to:
- **Frontend UI**: http://localhost:3000
- **Backend API**: http://localhost:8080/api/health

### 3. Stop All Services

```bash
docker-compose down
```

To also remove volumes (delete all data):
```bash
docker-compose down -v
```

## Configuration

### Environment Variables

Copy the example environment file:
```bash
cp .env.example .env
```

Edit `.env` to customize:
- MongoDB connection
- Backend port
- Frontend URLs
- CSMS connection URL

### Custom Configuration

The backend uses `configs/config.docker.yaml` when running in Docker. To use a custom configuration:

1. Create your config file (e.g., `configs/config.custom.yaml`)
2. Mount it in `docker-compose.yml`:
   ```yaml
   volumes:
     - ./configs/config.custom.yaml:/app/configs/config.yaml:ro
   ```

## Development Workflow

### Rebuild After Code Changes

Backend:
```bash
docker-compose build backend
docker-compose up backend
```

Frontend:
```bash
docker-compose build frontend
docker-compose up frontend
```

### View Logs

All services:
```bash
docker-compose logs -f
```

Specific service:
```bash
docker-compose logs -f backend
docker-compose logs -f mongodb
```

### Run Tests in Container

```bash
# Backend tests
docker-compose exec backend go test ./...

# Frontend tests
docker-compose exec frontend npm test
```

## Container Architecture

```
┌─────────────────────────────────────────────────────┐
│  Frontend Container (Nginx)                         │
│  Port: 3000                                          │
│  - Serves React app                                 │
│  - Proxies /api/* to backend                        │
│  - WebSocket proxy for /api/ws/*                    │
└──────────────────┬──────────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────────┐
│  Backend Container (Go)                             │
│  Port: 8080                                         │
│  - REST API endpoints                               │
│  - WebSocket message streaming                      │
│  - OCPP protocol implementation                     │
└──────────────────┬──────────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────────┐
│  MongoDB Container                                   │
│  Port: 27017                                        │
│  - Persistent storage                               │
│  - Automatic initialization                         │
└─────────────────────────────────────────────────────┘
```

## Networking

All containers run on the `ocpp-network` bridge network:
- Containers can communicate using service names (e.g., `mongodb`, `backend`)
- External access via published ports
- Frontend proxies API requests to avoid CORS issues

## Volumes

### MongoDB Data
```yaml
mongodb_data:
  - Persists database between restarts
  - Location: Docker volume (managed by Docker)
```

### Config Files
```yaml
./configs:/app/configs:ro
  - Read-only config mount
  - Allows hot config changes without rebuild
```

## Troubleshooting

### MongoDB Connection Fails

**Issue**: Backend can't connect to MongoDB

**Solution**:
```bash
# Check MongoDB health
docker-compose ps mongodb

# View MongoDB logs
docker-compose logs mongodb

# Ensure MongoDB is healthy before backend starts
docker-compose up mongodb
docker-compose up backend
```

### Frontend API Calls Fail

**Issue**: 404 errors on `/api/*` requests

**Solution**:
1. Check nginx proxy configuration in `web/docker/nginx.conf`
2. Verify backend is running: `docker-compose logs backend`
3. Test backend directly: `curl http://localhost:8080/api/health`

### WebSocket Connection Issues

**Issue**: "WebSocket connection failed" in browser console

**Solutions**:
1. Check nginx WebSocket proxy configuration
2. Verify backend WebSocket endpoint: `http://localhost:8080/api/ws/messages`
3. Check browser console for detailed error messages
4. Ensure connection upgrade headers are set correctly

### Port Already in Use

**Issue**: `bind: address already in use`

**Solution**:
```bash
# Find process using port (e.g., 3000)
lsof -ti:3000

# Kill the process
kill -9 $(lsof -ti:3000)

# Or change port in docker-compose.yml
```

### Out of Disk Space

**Issue**: Build fails with "no space left on device"

**Solution**:
```bash
# Remove unused Docker resources
docker system prune -a --volumes

# Check disk usage
docker system df
```

## Connecting to External CSMS

To connect to a CSMS running on your host machine:

1. Use `host.docker.internal` instead of `localhost`:
   ```yaml
   csms:
     default_url: "ws://host.docker.internal:9000"
   ```

2. Or use your host's IP address:
   ```bash
   # Find your IP
   ifconfig | grep "inet " | grep -v 127.0.0.1
   ```

## Production Deployment

### Security Hardening

1. **Enable TLS/HTTPS**:
   ```yaml
   server:
     tls:
       enabled: true
       cert_file: "/app/certs/server.crt"
       key_file: "/app/certs/server.key"
   ```

2. **Use MongoDB Authentication**:
   ```yaml
   mongodb:
     uri: "mongodb://user:password@mongodb:27017"
   ```

3. **Restrict CORS Origins**:
   Edit nginx.conf to allow only specific origins

4. **Use Docker Secrets** for sensitive data:
   ```yaml
   secrets:
     mongodb_password:
       file: ./secrets/mongodb_password.txt
   ```

### Performance Tuning

1. **Increase MongoDB pool size**:
   ```yaml
   mongodb:
     max_pool_size: 200
   ```

2. **Adjust message buffer**:
   ```yaml
   application:
     message_buffer_size: 5000
     batch_insert_interval: 2s
   ```

3. **Set container resource limits**:
   ```yaml
   services:
     backend:
       deploy:
         resources:
           limits:
             cpus: '2'
             memory: 2G
   ```

## Health Checks

All services have health checks:

```bash
# Check all services health
docker-compose ps

# Expected output:
# NAME                  STATUS
# ocpp-emu-backend      Up (healthy)
# ocpp-emu-frontend     Up
# ocpp-emu-mongodb      Up (healthy)
```

Backend health endpoint:
```bash
curl http://localhost:8080/api/health
```

Expected response:
```json
{
  "status": "healthy",
  "version": "0.1.0",
  "database": "connected",
  "stations": {...}
}
```

## Backup and Restore

### Backup MongoDB Data

```bash
# Create backup
docker-compose exec mongodb mongodump \
  --db=ocpp_emu \
  --out=/tmp/backup

# Copy to host
docker cp ocpp-emu-mongodb:/tmp/backup ./backup
```

### Restore MongoDB Data

```bash
# Copy backup to container
docker cp ./backup ocpp-emu-mongodb:/tmp/backup

# Restore
docker-compose exec mongodb mongorestore \
  --db=ocpp_emu \
  /tmp/backup/ocpp_emu
```

## Advanced Usage

### Running Individual Services

```bash
# Only MongoDB
docker-compose up mongodb

# Backend without frontend
docker-compose up mongodb backend

# All services in detached mode
docker-compose up -d
```

### Scaling Services

```bash
# Run multiple backend instances (requires load balancer)
docker-compose up --scale backend=3
```

### Custom Build Arguments

```bash
# Build with specific Go version
docker-compose build \
  --build-arg GO_VERSION=1.23 \
  backend
```

## Monitoring

### Container Stats

```bash
# Real-time stats
docker stats ocpp-emu-backend ocpp-emu-frontend ocpp-emu-mongodb

# One-time stats
docker-compose top
```

### Log Aggregation

For production, consider:
- ELK Stack (Elasticsearch, Logstash, Kibana)
- Grafana + Loki
- Datadog
- CloudWatch (AWS)

### Example with Prometheus

```yaml
# docker-compose.yml additions
services:
  prometheus:
    image: prom/prometheus
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
```

## Further Reading

- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Docker Best Practices](https://docs.docker.com/develop/dev-best-practices/)
- [MongoDB Docker](https://hub.docker.com/_/mongo)
- [Nginx Docker](https://hub.docker.com/_/nginx)

## Support

For issues with Docker setup:
1. Check this documentation
2. Review logs: `docker-compose logs`
3. Verify configuration files
4. Open an issue on GitHub with:
   - Docker version: `docker --version`
   - Docker Compose version: `docker-compose --version`
   - Complete error logs
   - Steps to reproduce
