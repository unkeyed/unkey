FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -o main .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .

ENV PORT 8080

CMD ["./main"]
