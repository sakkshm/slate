FROM golang:1.26 AS base
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

FROM base AS development
RUN go install github.com/air-verse/air@latest
COPY . .
CMD ["air", "-c", "scripts/api.air.toml"]

FROM base AS builder
ARG ENTRYPOINT_PATH=./cmd/api
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /server ${ENTRYPOINT_PATH}

FROM alpine:3.20 AS production
WORKDIR /app
RUN apk add --no-cache ca-certificates wget \
    && addgroup -S slate && adduser -S slate -G slate
COPY --from=builder /server /server
USER slate
EXPOSE 8080
ENTRYPOINT ["/server"]

