# ---- Stage 1: Build (Bun) ----
FROM oven/bun:1-alpine AS builder
RUN apk update && apk add --no-cache libc6-compat

WORKDIR /app

# Backend API URL — set at build time (Vite bakes it into the bundle)
ARG VITE_API_URL=/api
ENV VITE_API_URL=${VITE_API_URL}

# Install dependencies (separate from source for layer caching)
COPY package.json ./
RUN bun install --frozen-lockfile

# Copy source and build
COPY . .
RUN bun run build

# ---- Stage 2: Runner (Nginx) ----
FROM nginx:alpine AS runner

# Configure Nginx for SPA routing (TanStack Router)
RUN echo 'server { \
    listen 80; \
    location / { \
        root /usr/share/nginx/html; \
        index index.html index.htm; \
        try_files $uri $uri/ /index.html; \
    } \
}' > /etc/nginx/conf.d/default.conf

# Copy static assets from builder
COPY --from=builder /app/dist /usr/share/nginx/html

# Expose port 80 (container port, host port mapped via CLIENT_PORT in docker-compose)
EXPOSE 80

# Start Nginx
CMD ["nginx", "-g", "daemon off;"]
