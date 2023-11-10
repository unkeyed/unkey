FROM oven/bun:1
WORKDIR /app
COPY . .
RUN bun install

ARG PORT
EXPOSE ${PORT:-8000}

CMD ["bun", "server.ts"]
