# Build stage
FROM node:20-bookworm-slim AS builder

ARG BUILD_TIME=unknown

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install all dependencies (including dev for build)
RUN npm ci --no-audit --no-fund

# Copy source code
COPY . .

# Install Playwright Chromium with system dependencies
# This automatically installs both the browser AND required system libraries
ENV PLAYWRIGHT_BROWSERS_PATH=/app/.local-browsers
RUN npx playwright install --with-deps chromium

# Build TypeScript
RUN npm run build


# Runtime stage
FROM node:20-bookworm-slim

# Install minimal runtime dependencies for Chromium
# These are the runtime-only libraries (not build tools)
RUN apt-get update && apt-get install -y --no-install-recommends \
    libnss3 \
    libnspr4 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libpango-1.0-0 \
    libcairo2 \
    libasound2 \
    libstdc++6 \
    libdbus-1-3 \
    && rm -rf /var/lib/apt/lists/*

# Add non-root user
RUN groupadd -r appgroup && useradd -r -g appgroup -G audio,video appuser

WORKDIR /app

# Copy package files first
COPY --chown=appuser:appgroup package*.json ./

# Install production dependencies only (before copying built code/browsers)
RUN npm ci --omit=dev --no-audit --no-fund

# Copy built application code
COPY --from=builder --chown=appuser:appgroup /app/dist ./dist

# Copy the exact Playwright browsers downloaded in the builder stage
COPY --from=builder --chown=appuser:appgroup /app/.local-browsers /app/.local-browsers

# Point Playwright to the correct, absolute path of the copied browsers
ENV PLAYWRIGHT_BROWSERS_PATH=/app/.local-browsers
ENV NODE_ENV=production

EXPOSE 3000

USER appuser

CMD ["node", "dist/index.js"]
