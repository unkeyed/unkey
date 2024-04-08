# use the official Bun image
# see all versions at https://hub.docker.com/r/oven/bun/tags
FROM oven/bun:1
WORKDIR /usr/src/app

COPY . .
RUN bun install


# run the app
USER bun
EXPOSE 8000/tcp
ENTRYPOINT [ "bun", "run", "index.ts" ]