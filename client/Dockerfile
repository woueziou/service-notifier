# ---- Stage 1: Build (Bun) ----
FROM oven/bun:alpine AS builder

# Backend API URL — set at build time
ARG VITE_API_URL=/api
ENV VITE_API_URL=${VITE_API_URL}

WORKDIR /app
COPY package.json bun.lockb ./
RUN bun install --frozen-lockfile
COPY . .
RUN bun run build

# ---- Stage 2: Serve (Node 24) ----
FROM node:24-alpine

RUN npm install -g serve
COPY --from=builder /app/dist /app/dist

WORKDIR /app
EXPOSE 80
CMD ["serve", "-s", "dist", "-l", "80"]
