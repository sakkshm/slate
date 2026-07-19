FROM node:20-alpine AS base
RUN apk add --no-cache git
WORKDIR /app
