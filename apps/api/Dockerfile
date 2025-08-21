FROM  node:22.18.0-slim AS compile

WORKDIR /app
RUN npm install -g workerd@latest pnpm

COPY pnpm-lock.yaml ./
RUN pnpm fetch


COPY . .
RUN rm .dev.vars || true


RUN pnpm install --offline -r

WORKDIR /app/apps/api
RUN pnpm wrangler deploy --outdir=dist --dry-run

RUN pnpm workerd compile ./worker.capnp > unkey


FROM ubuntu:24.04
COPY --from=compile /app/apps/api/unkey /usr/bin/unkey

EXPOSE 8787
CMD ["unkey"]


# FROM --platform=linux/amd64 node:lts AS compile

# WORKDIR /app
# RUN npm install -g workerd pnpm

# COPY pnpm-lock.yaml ./
# RUN pnpm fetch


# COPY . .
# RUN rm .dev.vars || true



# RUN pnpm install --offline -r

# WORKDIR /app/apps/api
# RUN pnpm wrangler deploy --dry-run --outdir .wrangler/dist

# FROM jacoblincool/workerd

# COPY --from=compile /app/apps/api/worker.capnp /app/worker.capnp
# COPY --from=compile /app/apps/api/.wrangler/dist /app/.wrangler/dist

# WORKDIR /app

# EXPOSE 8787
