FROM oven/bun:alpine

WORKDIR /app

RUN apk add github-cli
RUN apk add make
RUN bun install -g mintlify@latest

COPY . .

EXPOSE 3000

CMD ["make", "dev"]
