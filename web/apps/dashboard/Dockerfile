# ---- Base ----
FROM node:22-alpine AS base
RUN npm install -g pnpm

# ---- Dependencies ----
FROM base AS deps
WORKDIR /unkey/web

COPY ./web/package.json ./web/pnpm-lock.yaml ./web/pnpm-workspace.yaml ./web/turbo.json ./

# Apps
COPY ./web/apps/dashboard/package.json ./apps/dashboard/package.json

# Internal packages
COPY ./web/internal/billing/package.json ./internal/billing/package.json
COPY ./web/internal/clickhouse/package.json ./internal/clickhouse/package.json
COPY ./web/internal/db/package.json ./internal/db/package.json
COPY ./web/internal/encoding/package.json ./internal/encoding/package.json
COPY ./web/internal/encryption/package.json ./internal/encryption/package.json
COPY ./web/internal/events/package.json ./internal/events/package.json
COPY ./web/internal/hash/package.json ./internal/hash/package.json
COPY ./web/internal/icons/package.json ./internal/icons/package.json
COPY ./web/internal/id/package.json ./internal/id/package.json
COPY ./web/internal/keys/package.json ./internal/keys/package.json
COPY ./web/internal/resend/package.json ./internal/resend/package.json
COPY ./web/internal/schema/package.json ./internal/schema/package.json
COPY ./web/internal/tsconfig/package.json ./internal/tsconfig/package.json
COPY ./web/internal/ui/package.json ./internal/ui/package.json
COPY ./web/internal/validation/package.json ./internal/validation/package.json
COPY ./web/internal/vault/package.json ./internal/vault/package.json
COPY ./web/internal/vercel/package.json ./internal/vercel/package.json
COPY ./web/internal/error/package.json ./internal/error/package.json
COPY ./web/internal/rbac/package.json ./internal/rbac/package.json

RUN --mount=type=cache,id=pnpm,target=/root/.local/share/pnpm/store pnpm install --frozen-lockfile

# ---- Builder ----
FROM base AS builder
WORKDIR /unkey/web

COPY --from=deps /unkey/web/node_modules ./node_modules
COPY --from=deps /unkey/web/apps/dashboard/node_modules ./apps/dashboard/node_modules
COPY --from=deps /unkey/web/internal ./internal

COPY ./web/package.json ./web/pnpm-lock.yaml ./web/pnpm-workspace.yaml ./web/turbo.json ./
COPY ./web/apps/dashboard ./apps/dashboard
COPY ./web/internal ./internal

RUN --mount=type=cache,id=turbo,target=/unkey/.turbo pnpm turbo run build --filter=@unkey/dashboard

# ---- Runner ----
FROM node:22-alpine AS runner
WORKDIR /unkey/web

ENV NODE_ENV=production
ENV HOSTNAME="0.0.0.0"
ENV PORT=3000

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# Copy public assets into the app directory, matching standalone server layout
COPY --from=builder --chown=nextjs:nodejs /unkey/web/apps/dashboard/public ./apps/dashboard/public

# Set correct permissions for prerender cache
RUN mkdir -p apps/dashboard/.next
RUN chown nextjs:nodejs apps/dashboard/.next

# Copy standalone build output
COPY --from=builder --chown=nextjs:nodejs /unkey/web/apps/dashboard/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /unkey/web/apps/dashboard/.next/static ./apps/dashboard/.next/static

USER nextjs

EXPOSE 3000

CMD ["node", "apps/dashboard/server.js"]
