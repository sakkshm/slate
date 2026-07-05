FROM golang:1.26 AS base
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

FROM base AS development
RUN go install github.com/air-verse/air@latest
COPY . .
CMD ["air", "--build.cmd", "go build -o ./tmp/main ./cmd/api/main.go", "--build.bin", "./tmp/main"]

FROM base AS builder
ARG ENTRYPOINT_PATH=./cmd/api
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /server ${ENTRYPOINT_PATH}

FROM scratch AS production
COPY --from=builder /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]