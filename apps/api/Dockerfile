FROM --platform=linux/amd64 node:lts AS compile

WORKDIR /app
RUN npm install -g workerd pnpm

COPY pnpm-lock.yaml ./
RUN pnpm fetch


COPY . .




RUN pnpm install --offline -r

WORKDIR /app/apps/api
RUN pnpm wrangler deploy --outdir=dist --dry-run

RUN pnpm workerd compile ./workerd.capnp > unkey


FROM --platform=linux/amd64 ubuntu:latest
COPY --from=compile /app/apps/api/unkey /usr/bin/unkey

EXPOSE 8787
CMD ["unkey"]
