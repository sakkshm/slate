FROM node:20-bookworm AS base
RUN apt-get update \
    && apt-get install -y --no-install-recommends git python3 make g++ \
    && rm -rf /var/lib/apt/lists/* \
    && npm install -g npm@11
WORKDIR /app
