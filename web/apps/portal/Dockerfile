FROM node:22-alpine AS base
RUN corepack enable && corepack prepare pnpm@8.6.9 --activate

# --- Dependencies ---
FROM base AS deps
WORKDIR /app
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
COPY web/apps/portal/package.json ./apps/portal/package.json
# Copy internal packages that portal depends on
COPY web/internal/ui/package.json ./internal/ui/package.json
COPY web/internal/icons/package.json ./internal/icons/package.json
COPY web/internal/id/package.json ./internal/id/package.json
RUN pnpm install --frozen-lockfile

# --- Builder ---
FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/apps/portal/node_modules ./apps/portal/node_modules
COPY --from=deps /app/internal ./internal
COPY web/ .
RUN pnpm --filter @unkey/portal build

# --- Runner ---
FROM node:22-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 portal

COPY --from=builder --chown=portal:nodejs /app/apps/portal/dist ./dist

USER portal

EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "dist/server/server.js"]
