# Build
FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/pr-channels ./cmd/pr-channels

# Run
FROM gcr.io/distroless/static-debian12
COPY --from=build /bin/pr-channels /bin/pr-channels
EXPOSE 8080
ENTRYPOINT ["/bin/pr-channels"]
