# MaaS-Router Admin Frontend Dockerfile
# UmiJS 4 + Ant Design Pro + React 18
# 多阶段构建: deps → build → nginx

# ---- Stage 1: Dependencies ----
FROM node:20-alpine AS deps

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm ci --prefer-offline --no-audit --no-fund

# ---- Stage 2: Build ----
FROM node:20-alpine AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

ENV UMI_ENV=production
ENV NODE_ENV=production

RUN npm run build

# ---- Stage 3: Production (Nginx) ----
FROM nginx:1.25-alpine AS production

# Copy custom nginx config
COPY <<'EOF' /etc/nginx/conf.d/default.conf
server {
    listen 8000;
    server_name _;

    root /usr/share/nginx/html;
    index index.html;

    # Gzip compression
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;
    gzip_min_length 256;

    # Static assets with long cache
    location /umi.js {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # SPA fallback
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Health check endpoint
    location /health {
        access_log off;
        return 200 'ok';
        add_header Content-Type text/plain;
    }
}
EOF

# Copy built assets
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8000/health || exit 1

CMD ["nginx", "-g", "daemon off;"]
