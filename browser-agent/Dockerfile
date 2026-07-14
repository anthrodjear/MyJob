# Build stage
FROM node:20-alpine AS builder

ARG BUILD_TIME=unknown

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install all dependencies (including dev for build)
RUN npm cache clean --force && npm ci --no-audit --no-fund

# Copy source code
COPY . .

# Set the path where Playwright installs browsers *during build*
ENV PLAYWRIGHT_BROWSERS_PATH=/app/.local-browsers
RUN npx playwright install chromium

# Build TypeScript
RUN npm run build


# Runtime stage
FROM node:20-alpine

# Install Playwright system dependencies (chromium provided by Playwright copy, not apk)
RUN apk add --no-cache \
    nss \
    freetype \
    harfbuzz \
    ttf-freefont

# Add non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy package files first
COPY --chown=appuser:appgroup package*.json ./

# Install production dependencies only (before copying built code/browsers)
RUN npm cache clean --force && npm ci --omit=dev --no-audit --no-fund && npm cache clean --force

# Copy built application code
COPY --from=builder --chown=appuser:appgroup /app/dist ./dist

# Copy the exact Playwright browsers downloaded in the builder stage
COPY --from=builder --chown=appuser:appgroup /app/.local-browsers /app/.local-browsers

# Point Playwright to the correct, absolute path of the copied browsers
ENV PLAYWRIGHT_BROWSERS_PATH=/app/.local-browsers
ENV NODE_ENV=production

USER appuser

EXPOSE 3000

CMD ["npm", "start"]
