# use the official Bun image
# see all versions at https://hub.docker.com/r/oven/bun/tags
FROM node:latest
WORKDIR /usr/src/app

RUN npm i -g pnpm

COPY . .
RUN pnpm install
RUN pnpm build

# run the app


ARG PORT
EXPOSE ${PORT:-3000}

ENTRYPOINT [ "node", "./dist/index.js" ]