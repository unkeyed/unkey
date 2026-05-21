FROM oven/bun:alpine@sha256:5acc90a93e91ff07bf72aa90a7c9f0fa189765aec90b47bdbf2152d2196383c0

WORKDIR /app

RUN apk add github-cli
RUN apk add make
RUN bun install -g mintlify@4.2.573

COPY . .

EXPOSE 3000

CMD ["make", "dev"]
