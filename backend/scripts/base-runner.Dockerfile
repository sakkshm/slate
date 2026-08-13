FROM node:20-alpine AS base
RUN apk add --no-cache git python3 make g++ linux-headers
WORKDIR /app
