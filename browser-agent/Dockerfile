# Build stage
FROM node:20-alpine AS builder

ARG BUILD_TIME=unknown

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install all dependencies (including dev for build)
RUN npm ci

# Copy source code
COPY . .

# Build TypeScript
RUN npm run build

# Runtime stage
FROM node:20-alpine

# Install Playwright system dependencies only (browser binary comes from npm)
RUN apk add --no-cache \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ttf-freefont

# Add non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy package files and built code
COPY --from=builder --chown=appuser:appgroup /app/package*.json ./
COPY --from=builder --chown=appuser:appgroup /app/dist ./dist

# Install production dependencies only
RUN npm ci --omit=dev && npm cache clean --force

# Playwright browsers are installed as a production dependency
# Ensure Chromium binary is available for Playwright
ENV PLAYWRIGHT_BROWSERS_PATH=0

USER appuser

EXPOSE 3000
ENV NODE_ENV=production

CMD ["npm", "start"]
