# ---- Base ----
FROM node:22-alpine AS base
RUN npm install -g pnpm

# ---- Dependencies ----
FROM base AS deps
WORKDIR /unkey

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml turbo.json ./

# Apps
COPY ./apps/dashboard/package.json ./apps/dashboard/package.json

# Internal packages
COPY ./internal/billing/package.json ./internal/billing/package.json
COPY ./internal/clickhouse/package.json ./internal/clickhouse/package.json
COPY ./internal/db/package.json ./internal/db/package.json
COPY ./internal/encoding/package.json ./internal/encoding/package.json
COPY ./internal/encryption/package.json ./internal/encryption/package.json
COPY ./internal/events/package.json ./internal/events/package.json
COPY ./internal/hash/package.json ./internal/hash/package.json
COPY ./internal/icons/package.json ./internal/icons/package.json
COPY ./internal/id/package.json ./internal/id/package.json
COPY ./internal/keys/package.json ./internal/keys/package.json
COPY ./internal/resend/package.json ./internal/resend/package.json
COPY ./internal/schema/package.json ./internal/schema/package.json
COPY ./internal/tsconfig/package.json ./internal/tsconfig/package.json
COPY ./internal/ui/package.json ./internal/ui/package.json
COPY ./internal/validation/package.json ./internal/validation/package.json
COPY ./internal/vault/package.json ./internal/vault/package.json
COPY ./internal/vercel/package.json ./internal/vercel/package.json
COPY ./internal/error/package.json ./internal/error/package.json
COPY ./internal/rbac/package.json ./internal/rbac/package.json

RUN --mount=type=cache,id=pnpm,target=/root/.local/share/pnpm/store pnpm install --frozen-lockfile

# ---- Builder ----
FROM base AS builder
WORKDIR /unkey

COPY --from=deps /unkey/node_modules ./node_modules
COPY --from=deps /unkey/apps/dashboard/node_modules ./apps/dashboard/node_modules
COPY --from=deps /unkey/internal ./internal

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml turbo.json ./
COPY ./apps/dashboard ./apps/dashboard
COPY ./internal ./internal

RUN --mount=type=cache,id=turbo,target=/unkey/.turbo pnpm turbo run build --filter=@unkey/dashboard

# ---- Runner ----
FROM node:22-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV HOSTNAME="0.0.0.0"
ENV PORT=3000

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# Copy public assets into the app directory, matching standalone server layout
COPY --from=builder --chown=nextjs:nodejs /unkey/apps/dashboard/public ./apps/dashboard/public

# Set correct permissions for prerender cache
RUN mkdir -p apps/dashboard/.next
RUN chown nextjs:nodejs apps/dashboard/.next

# Copy standalone build output
COPY --from=builder --chown=nextjs:nodejs /unkey/apps/dashboard/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /unkey/apps/dashboard/.next/static ./apps/dashboard/.next/static

USER nextjs

EXPOSE 3000

CMD ["node", "apps/dashboard/server.js"]
