FROM ubuntu:24.04@sha256:c4a8d5503dfb2a3eb8ab5f807da5bc69a85730fb49b5cfca2330194ebcc41c7b AS os
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates libatomic1 libstdc++6 \
    && rm -rf /var/lib/apt/lists/*

FROM os AS base
RUN apt-get update && apt-get install -y --no-install-recommends curl git \
    && rm -rf /var/lib/apt/lists/*
ENV MISE_DATA_DIR=/opt/mise \
    MISE_INSTALL_PATH=/usr/local/bin/mise
WORKDIR /app
COPY dev/install-mise ./dev/install-mise
RUN ./dev/install-mise
COPY .mise/config.toml .mise/mise.lock ./.mise/
RUN mise trust .mise/config.toml && mise install --locked --yes node github:pnpm/pnpm \
    && cp -a "$(mise where node)" /opt/node

# --- Builder ---
FROM base AS builder
ENV CI=true \
    pnpm_config_store_dir=/pnpm/store
COPY web/ .
RUN --mount=type=cache,id=unkey-pnpm,target=/pnpm/store \
    mise exec -- pnpm install --frozen-lockfile \
    && mise exec -- pnpm --filter @unkey/portal build

# --- Runner ---
FROM os AS runner
WORKDIR /app

COPY --from=base /opt/node /opt/node
ENV NODE_ENV=production \
    PATH=/opt/node/bin:$PATH

RUN groupadd --system --gid 1001 nodejs \
    && useradd --system --uid 1001 --gid nodejs portal

COPY --from=builder --chown=portal:nodejs /app/apps/portal/.output ./.output

USER portal

EXPOSE 3000
ENV PORT=3000
ENV HOST="0.0.0.0"

CMD ["node", ".output/server/index.mjs"]
