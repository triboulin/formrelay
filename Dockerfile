# syntax=docker/dockerfile:1

# ---- Stage 1: Build ----
FROM golang:1.22-alpine AS build

WORKDIR /src

# Dépendances (mise en cache Docker)
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/formrelay ./cmd/server

# ---- Stage 2: Run ----
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S formrelay && adduser -S formrelay -G formrelay

WORKDIR /app

COPY --from=build /out/formrelay ./formrelay
COPY --from=build /src/templates ./templates

RUN mkdir -p /app/data && chown -R formrelay:formrelay /app

USER formrelay

EXPOSE 8080

ENTRYPOINT ["./formrelay"]
