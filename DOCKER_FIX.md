# Docker Frontend Loading Issue - Fixed

## Issue Description

After building and running Docker containers, the frontend would sometimes show "Loading" indefinitely and nginx logs showed HTTP 499 errors:

```
ocpp-emu-frontend  | GET /api/health HTTP/1.1" 499 0
ocpp-emu-frontend  | GET /api/stations HTTP/1.1" 499 0
```

HTTP 499 = "Client Closed Request" - the browser gave up waiting for a response.

## Root Cause

The frontend was configured to make API calls directly to `http://localhost:8080` (the backend container), which is not accessible from the browser when running in Docker. The requests would timeout because:

1. Browser → tries to connect to `localhost:8080`
2. `localhost:8080` from browser = host machine, not the Docker container
3. Backend container is only accessible via nginx proxy at `localhost:3000/api/*`
4. Request times out after 8 seconds
5. Nginx logs HTTP 499 (client closed request)

## Solution

Changed the frontend to use **relative URLs** that go through the nginx reverse proxy:

### 1. Updated API Client (`web/src/services/api.js`)

**Before:**
```javascript
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'
```

**After:**
```javascript
// Use relative URL in Docker (goes through nginx proxy)
// Use full URL in development (direct to backend)
const API_BASE_URL = import.meta.env.VITE_API_URL || ''
```

Also increased timeout from 8 seconds to 30 seconds for better reliability.

### 2. Updated WebSocket URL (`web/src/pages/Messages.jsx`)

**Before:**
```javascript
const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080'
```

**After:**
```javascript
// Use relative WebSocket URL for Docker/production
const WS_URL = import.meta.env.VITE_WS_URL ||
  (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host
```

### 3. Updated Docker Compose

**Before:**
```yaml
args:
  VITE_API_URL: http://localhost:3000
  VITE_WS_URL: ws://localhost:3000
```

**After:**
```yaml
args:
  VITE_API_URL: ""
  VITE_WS_URL: ""
```

Empty strings mean the frontend will use relative URLs.

### 4. Updated `.env.example`

Added guidance for development vs Docker:

```bash
# For local development (frontend dev server)
VITE_API_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080

# For Docker: Leave empty to use relative URLs
# VITE_API_URL=
# VITE_WS_URL=
```

## How It Works Now

### Docker/Production Flow
```
Browser → http://localhost:3000/api/health
    ↓ (relative URL)
Nginx at localhost:3000
    ↓ (proxy_pass)
Backend at backend:8080
    ↓ (response)
Browser ✅
```

### Development Flow
When running `npm run dev` locally with `VITE_API_URL=http://localhost:8080`:

```
Browser → http://localhost:8080/api/health
    ↓ (direct connection)
Backend at localhost:8080
    ↓ (response)
Browser ✅
```

## Verification

All endpoints now working correctly:

```bash
$ curl http://localhost:3000/api/health
{"status":"healthy",...}

$ curl http://localhost:3000/api/stations
{"count":1,"stations":[...]}

$ curl http://localhost:3000
<!doctype html>...  # Frontend HTML
```

HTTP status codes:
- Frontend: 200 ✅
- Health API: 200 ✅
- Stations API: 200 ✅
- Messages API: 200 ✅

## Testing Steps

1. Rebuild frontend:
   ```bash
   docker-compose build frontend
   ```

2. Restart containers:
   ```bash
   docker-compose up -d
   ```

3. Test in browser:
   - Open http://localhost:3000
   - Navigate to Dashboard → should load immediately
   - Navigate to Stations → should load immediately
   - Navigate to Messages → should load immediately
   - No more "Loading..." stuck screens

4. Check nginx logs (should see 200, not 499):
   ```bash
   docker-compose logs frontend | tail -20
   ```

## Key Takeaways

### For Docker Deployment
- ✅ Use **relative URLs** (`""` or `"/"`) for API calls
- ✅ Let nginx handle proxying to backend
- ✅ Use `window.location.host` for WebSocket URLs
- ✅ Set environment variables to empty string in docker-compose

### For Local Development
- ✅ Set `VITE_API_URL=http://localhost:8080` in `.env`
- ✅ Set `VITE_WS_URL=ws://localhost:8080` in `.env`
- ✅ Frontend dev server can directly access backend

### Architecture
```
┌──────────────────────────────────────────┐
│  Browser (localhost:3000)                │
└──────────────┬───────────────────────────┘
               │ Relative URLs
               ↓
┌──────────────────────────────────────────┐
│  Nginx Proxy (Container: frontend:80)   │
│  - Serves static files                   │
│  - Proxies /api/* → backend:8080         │
│  - Proxies WebSocket upgrades            │
└──────────────┬───────────────────────────┘
               │ Container network
               ↓
┌──────────────────────────────────────────┐
│  Backend API (Container: backend:8080)   │
│  - REST endpoints                        │
│  - WebSocket message streaming           │
└──────────────┬───────────────────────────┘
               │
               ↓
┌──────────────────────────────────────────┐
│  MongoDB (Container: mongodb:27017)      │
└──────────────────────────────────────────┘
```

## Files Modified

1. `web/src/services/api.js` - Use relative URL, increase timeout
2. `web/src/pages/Messages.jsx` - Use dynamic WebSocket URL
3. `docker-compose.yml` - Set environment variables to empty
4. `.env.example` - Add development vs Docker guidance

## Additional Benefits

1. **Works with any hostname** - not hardcoded to localhost
2. **HTTPS ready** - automatically uses wss:// if served over https://
3. **Port agnostic** - works on any port nginx is configured to
4. **Better timeout** - 30s instead of 8s for slower connections
5. **Development friendly** - clear separation of dev vs prod config

## Common Pitfalls to Avoid

❌ **Don't** hardcode `http://localhost:8080` in production code
❌ **Don't** skip rebuilding frontend after config changes
❌ **Don't** use different ports between services and nginx

✅ **Do** use relative URLs for Docker/production
✅ **Do** use environment variables for configuration
✅ **Do** rebuild frontend after changing build args
✅ **Do** test both development and Docker modes

---

**Issue**: Frontend stuck on "Loading" with HTTP 499 errors
**Root Cause**: Hardcoded backend URL not accessible from browser in Docker
**Solution**: Use relative URLs that go through nginx proxy
**Status**: ✅ **FIXED**

**Date**: 2025-11-08
**Time to Fix**: ~15 minutes
**Impact**: Critical (app unusable) → None (fully working)
