# syntax=docker/dockerfile:1

FROM node:22-alpine AS builder

WORKDIR /app

RUN --mount=type=cache,target=/root/.npm \
    --mount=source=package.json,target=package.json \
    --mount=source=package-lock.json,target=package-lock.json \
    npm ci

COPY . .

RUN npm run build -- --outDir /dist

FROM nginx:1.27-alpine

COPY --from=builder /dist /usr/share/nginx/html

EXPOSE 80
