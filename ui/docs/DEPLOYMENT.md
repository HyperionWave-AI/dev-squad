# Hyperion Coordinator UI - Deployment Guide

## Table of Contents

1. [Build Process](#build-process)
2. [Development Deployment](#development-deployment)
3. [Production Deployment](#production-deployment)
4. [Docker Deployment](#docker-deployment)
5. [Environment Configuration](#environment-configuration)
6. [Nginx Configuration](#nginx-configuration)
7. [CI/CD Pipeline](#cicd-pipeline)

---

## Build Process

### Development Build

```bash
# Start Vite dev server with HMR
npm run dev

# Output:
# VITE v7.1.7  ready in 423 ms
# ➜  Local:   http://localhost:5173/ui/
```

**Features**:
- Hot Module Replacement (HMR)
- Fast refresh
- Source maps
- API proxy to MCP Bridge

### Production Build

```bash
# TypeScript check + Vite production build
npm run build

# Output directory: dist/
```

**Build Steps**:
1. TypeScript compilation (`tsc -b`)
2. Vite production build
3. Asset optimization
4. Code minification
5. Generate source maps

**Build Output**:
```
dist/
├── assets/
│   ├── index-[hash].js      # Main bundle
│   ├── vendor-[hash].js     # Third-party dependencies
│   └── index-[hash].css     # Compiled styles
├── index.html               # Entry HTML
└── ui/                      # Static assets
```

---

## Development Deployment

### Local Development

```bash
# Install dependencies
npm install

# Start dev server
npm run dev

# Open browser to http://localhost:5173/ui/
```

### Backend Connection

The UI connects to the MCP HTTP Bridge for API calls:

**Development Configuration** (`vite.config.ts`):
```typescript
export default defineConfig({
  server: {
    proxy: {
      '/api/v1': {
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:7095',
        changeOrigin: true
      }
    }
  }
})
```

**Start Backend**:
```bash
# From project root
make dev

# Or
make dev-hot  # With hot reload
```

### Full Stack Development

1. **Terminal 1 - Backend**:
```bash
cd /path/to/dev-squad
make dev
# Backend runs on http://localhost:7095
```

2. **Terminal 2 - Frontend**:
```bash
cd /path/to/dev-squad/ui
npm run dev
# Frontend runs on http://localhost:5173
```

3. **Browser**:
   - Open `http://localhost:5173/ui/`
   - API calls proxied to `http://localhost:7095`

---

## Production Deployment

### Build for Production

```bash
# Clean previous build
rm -rf dist/

# Build with TypeScript check
npm run build

# Verify build
ls -lh dist/
```

### Production Preview

```bash
# Build and preview locally
npm run build
npm run preview

# Open http://localhost:4173/ui/
```

### Deploy to Server

**Option 1: Manual Deployment**

```bash
# Build locally
npm run build

# Copy dist/ to server
scp -r dist/* user@server:/var/www/hyperion/ui/

# Restart nginx
ssh user@server "sudo systemctl restart nginx"
```

**Option 2: Docker Deployment** (Recommended)

See [Docker Deployment](#docker-deployment) section.

---

## Docker Deployment

### Dockerfile

The UI includes two Dockerfiles:

#### Production Dockerfile (`Dockerfile`)

```dockerfile
FROM node:18-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html/ui
COPY nginx.conf /etc/nginx/nginx.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
```

#### Development Dockerfile (`Dockerfile.dev`)

```dockerfile
FROM node:18-alpine

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .

EXPOSE 5173

CMD ["npm", "run", "dev", "--", "--host"]
```

### Build Docker Image

```bash
# Production image
docker build -t hyperion-ui:latest .

# Development image
docker build -f Dockerfile.dev -t hyperion-ui:dev .
```

### Run Docker Container

```bash
# Production
docker run -d \
  --name hyperion-ui \
  -p 80:80 \
  hyperion-ui:latest

# Development
docker run -d \
  --name hyperion-ui-dev \
  -p 5173:5173 \
  -v $(pwd):/app \
  -v /app/node_modules \
  hyperion-ui:dev
```

### Docker Compose

```yaml
version: '3.8'

services:
  ui:
    build:
      context: ./ui
      dockerfile: Dockerfile
    ports:
      - "80:80"
    environment:
      - NODE_ENV=production
    depends_on:
      - mcp-bridge

  mcp-bridge:
    build: ./mcp-http-bridge
    ports:
      - "7095:7095"
    environment:
      - PORT=7095
```

**Run with Docker Compose**:
```bash
docker-compose up -d
```

---

## Environment Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_MCP_BRIDGE_URL` | `http://localhost:7095` | MCP Bridge API URL |
| `NODE_ENV` | `development` | Node environment |

### Setting Environment Variables

**Development** (`.env.local`):
```bash
# Create .env.local file
cat > .env.local <<EOF
VITE_MCP_BRIDGE_URL=http://localhost:7095
EOF
```

**Production** (Docker):
```bash
docker run -d \
  -e VITE_MCP_BRIDGE_URL=http://mcp-bridge:7095 \
  hyperion-ui:latest
```

### Base Path Configuration

The UI is always served at `/ui/` path:

```typescript
// vite.config.ts
export default defineConfig({
  base: '/ui/',  // Always use /ui/ base path
})
```

**URLs**:
- **Root**: `http://localhost:7095/ui/`
- **Chat**: `http://localhost:7095/ui/chat`
- **Tasks**: `http://localhost:7095/ui/tasks`

---

## Nginx Configuration

### Production Nginx Config (`nginx.conf`)

```nginx
events {
    worker_connections 1024;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    sendfile        on;
    keepalive_timeout  65;

    server {
        listen 80;
        server_name localhost;

        # Serve UI at /ui/ path
        location /ui/ {
            alias /usr/share/nginx/html/ui/;
            try_files $uri $uri/ /ui/index.html;

            # Enable CORS for API calls
            add_header 'Access-Control-Allow-Origin' '*' always;
            add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS' always;
            add_header 'Access-Control-Allow-Headers' 'Content-Type, Authorization' always;
        }

        # Proxy API calls to MCP Bridge
        location /api/ {
            proxy_pass http://mcp-bridge:7095;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection 'upgrade';
            proxy_set_header Host $host;
            proxy_cache_bypass $http_upgrade;
        }

        # WebSocket support
        location /api/v1/chat/stream {
            proxy_pass http://mcp-bridge:7095;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_read_timeout 86400;
        }
    }
}
```

### Nginx Configuration Features

- **Static file serving** at `/ui/` path
- **API proxying** to MCP Bridge
- **WebSocket support** for chat streaming
- **SPA fallback** with `try_files`
- **CORS headers** for API access

---

## CI/CD Pipeline

### GitHub Actions Workflow

**Example `.github/workflows/ui-deploy.yml`**:

```yaml
name: Deploy UI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build-and-test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
          cache: 'npm'
          cache-dependency-path: ui/package-lock.json

      - name: Install dependencies
        run: |
          cd ui
          npm ci

      - name: Lint
        run: |
          cd ui
          npm run lint

      - name: TypeScript check
        run: |
          cd ui
          npx tsc --noEmit

      - name: Build
        run: |
          cd ui
          npm run build

      - name: Install Playwright
        run: |
          cd ui
          npx playwright install --with-deps chromium

      - name: Run Playwright tests
        run: |
          cd ui
          npm test

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: playwright-report
          path: ui/test-results/

  deploy:
    needs: build-and-test
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'

    steps:
      - uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'

      - name: Build
        run: |
          cd ui
          npm ci
          npm run build

      - name: Build Docker image
        run: |
          cd ui
          docker build -t hyperion-ui:${{ github.sha }} .
          docker tag hyperion-ui:${{ github.sha }} hyperion-ui:latest

      - name: Push to registry
        run: |
          echo "${{ secrets.DOCKER_PASSWORD }}" | docker login -u "${{ secrets.DOCKER_USERNAME }}" --password-stdin
          docker push hyperion-ui:${{ github.sha }}
          docker push hyperion-ui:latest

      - name: Deploy to server
        run: |
          # SSH and docker-compose up
```

### Pre-deployment Checklist

- [ ] ✅ All tests passing
- [ ] ✅ TypeScript compilation successful
- [ ] ✅ No ESLint errors
- [ ] ✅ Build completes without errors
- [ ] ✅ Environment variables configured
- [ ] ✅ Nginx configuration reviewed
- [ ] ✅ Backend connectivity verified

---

## Performance Optimization

### Build Optimization

**Code Splitting**:
```typescript
// Lazy load pages
const CodeChatPage = lazy(() => import('./pages/CodeChatPage'));
const KnowledgeBasePage = lazy(() => import('./pages/KnowledgeBasePage'));

<Suspense fallback={<CircularProgress />}>
  <Routes>
    <Route path="/chat" element={<CodeChatPage />} />
    <Route path="/knowledge" element={<KnowledgeBasePage />} />
  </Routes>
</Suspense>
```

**Bundle Analysis**:
```bash
# Install bundle analyzer
npm install --save-dev rollup-plugin-visualizer

# Add to vite.config.ts
import { visualizer } from 'rollup-plugin-visualizer';

export default defineConfig({
  plugins: [
    react(),
    visualizer({ open: true })
  ]
});

# Build with analysis
npm run build
# Opens stats.html in browser
```

### Nginx Caching

```nginx
location /ui/ {
    alias /usr/share/nginx/html/ui/;

    # Cache static assets
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # No cache for HTML
    location ~* \.html$ {
        expires -1;
        add_header Cache-Control "no-store, no-cache, must-revalidate";
    }
}
```

---

## Monitoring

### Health Check Endpoint

```nginx
location /ui/health {
    return 200 '{"status": "healthy"}';
    add_header Content-Type application/json;
}
```

### Logging

**Nginx Access Logs**:
```nginx
http {
    access_log /var/log/nginx/hyperion-ui-access.log;
    error_log /var/log/nginx/hyperion-ui-error.log;
}
```

**Application Logs**:
```typescript
// In production, send logs to monitoring service
if (import.meta.env.PROD) {
  console.log = (...args) => {
    // Send to logging service
    fetch('/api/logs', {
      method: 'POST',
      body: JSON.stringify({ level: 'log', message: args })
    });
  };
}
```

---

## Related Documentation

- [Architecture Overview](./ARCHITECTURE.md) - System architecture
- [Developer Guide](./DEVELOPER_GUIDE.md) - Development setup
- [Testing Guide](./TESTING.md) - Testing strategies
- [Troubleshooting](./TROUBLESHOOTING.md) - Common deployment issues

---

**Last Updated**: 2025-11-04
**Version**: ui2 (Hyperion Coordinator UI)
