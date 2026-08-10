FROM golang:1.26 AS base
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

FROM base AS development
RUN go install github.com/air-verse/air@latest
COPY . .
CMD ["air", "-c", "scripts/worker.air.toml"]

FROM base AS builder
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /worker ./cmd/worker

FROM alpine:3.20 AS production
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /worker /worker
ENTRYPOINT ["/worker"]
