FROM node:22.18.0

WORKDIR /unkey

# Install pnpm
RUN npm install -g pnpm

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml turbo.json ./

# Packages
COPY ./packages/cache/package.json ./packages/cache/package.json
COPY ./packages/error/package.json ./packages/error/package.json
COPY ./packages/rbac/package.json ./packages/rbac/package.json

# Apps
COPY ./apps/dashboard/package.json ./apps/dashboard/package.json

# Internal
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

# Install dependencies
RUN pnpm install -r

# Copy code
COPY ./apps/dashboard ./apps/dashboard
COPY ./internal ./internal
COPY ./packages ./packages

# Build
RUN pnpm turbo run build --filter=@unkey/dashboard

# Set hostname to 0.0.0.0 to allow external connections
ENV HOSTNAME="0.0.0.0"
ENV PORT=3000
EXPOSE 3000

# Run in production mode
CMD ["pnpm", "--filter", "@unkey/dashboard", "start"]
