# SPA build. Parameterised by WEB_PKG (pnpm filter).
# docker build -f build/web.Dockerfile --build-arg WEB_PKG=@base/webapp -t base/webapp-web:dev .
ARG NODE_VERSION=22-alpine3.23
FROM node:${NODE_VERSION} AS builder
ARG WEB_PKG
WORKDIR /src
RUN npm i -g pnpm@10.4.1
COPY pnpm-workspace.yaml package.json ./
COPY packages ./packages
COPY apps ./apps
COPY proto ./proto
COPY tsconfig.base.json ./
RUN pnpm fetch
RUN pnpm install --offline --filter ${WEB_PKG}...
RUN pnpm --filter ${WEB_PKG} build

FROM caddy:2-alpine
ARG WEB_PKG
COPY --from=builder /src/apps/*/web/dist /srv
COPY build/Caddyfile /etc/caddy/Caddyfile
EXPOSE 80
