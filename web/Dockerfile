# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN --mount=type=cache,target=/go/pkg/mod/,sharing=locked \
    --mount=source=go.sum,target=go.sum \
    --mount=source=go.mod,target=go.mod \
    go mod download -x

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=target=. \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/server ./server/cmd/

FROM alpine:3.21

RUN apk add --no-cache curl && \
    curl -sfS https://dotenvx.sh | sh

COPY --from=builder /bin/server /server

EXPOSE 8080

ENTRYPOINT ["dotenvx", "run", "--"]
CMD ["/server"]
