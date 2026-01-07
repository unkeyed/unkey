FROM node:22.18.0

RUN npm i -g pnpm turbo

WORKDIR /unkey


COPY . .
RUN rm -rf ./**/node_modules


WORKDIR /unkey/web


RUN pnpm install --frozen-lockfile

RUN pnpm turbo build --filter=api...


WORKDIR /unkey/web/apps/api
ENV WRANGLER_SEND_METRICS=false
RUN rm .dev.vars || true
CMD pnpm wrangler dev
