FROM node:22-alpine AS base

# Pruner stage - generates a subset of the monorepo
FROM base AS pruner
RUN apk update && apk add --no-cache libc6-compat
WORKDIR /app
RUN npm install -g turbo pnpm
COPY . .
RUN turbo prune @unkey/dashboard --docker

# Installer stage - installs dependencies and builds the project
FROM base AS installer
RUN apk update && apk add --no-cache libc6-compat
WORKDIR /app

# Install pnpm
RUN npm install -g pnpm

# First install dependencies (as they change less often)
COPY --from=pruner /app/out/json/ .
COPY --from=pruner /app/out/pnpm-lock.yaml ./pnpm-lock.yaml
RUN pnpm install --frozen-lockfile

# Build the project and all its dependencies
COPY --from=pruner /app/out/full/ .
COPY turbo.json turbo.json

# Build arguments for environment variables
ARG NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY
ARG NEXT_PUBLIC_CLERK_SIGN_IN_URL
ARG NEXT_PUBLIC_CLERK_SIGN_UP_URL
ARG NEXT_PUBLIC_UNKEY_API_URL
ARG NEXT_PUBLIC_UNKEY_APP_URL
ARG NEXT_PUBLIC_UNKEY_API_ID
ARG NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY
ARG NEXT_PUBLIC_WORKOS_REDIRECT_URI

RUN pnpm turbo run build --filter=@unkey/dashboard...

# Runner stage - production runtime
FROM base AS runner
WORKDIR /app

# Don't run production as root
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# Install pnpm
RUN npm install -g pnpm

# Copy package files
COPY --from=installer /app .

USER nextjs

# Set runtime environment variables
ENV NODE_ENV=production
ENV PORT=3000
# Bind to all interfaces for Docker access but Next.js will use host header
ENV HOSTNAME="0.0.0.0"

EXPOSE 3000

# Use Next.js built-in server
CMD ["pnpm", "--filter", "@unkey/dashboard", "start"]