FROM node:lts AS base

RUN npm i -g pnpm turbo bun
RUN corepack enable


FROM base AS builder

# Set working directory
WORKDIR /unkey

COPY . .
RUN turbo prune chproxy --docker

# Add lockfile and package.json's of isolated subworkspace
FROM base AS installer
WORKDIR /unkey

# First install dependencies (as they change less often)
COPY .gitignore .gitignore
COPY --from=builder /unkey/out/json/ .
COPY --from=builder /unkey/out/pnpm-lock.yaml ./pnpm-lock.yaml
RUN pnpm install

# Build the project and its dependencies
COPY --from=builder /unkey/out/full/ .
COPY turbo.json turbo.json

RUN pnpm turbo build --filter=chproxy...

FROM base AS runner
WORKDIR /unkey

COPY --from=installer /unkey .

WORKDIR /unkey/apps/chproxy
CMD bun run ./src/main.ts
