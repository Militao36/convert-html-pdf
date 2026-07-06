# Etapa 1: Build
FROM golang:1.26-bookworm AS builder

WORKDIR /app
COPY . .

RUN go mod tidy
RUN go build -o server .

# Etapa 2: Runtime com Chromium
FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
ENV CHROME_PATH=/usr/bin/chromium

RUN apt-get update && apt-get install -y \
    chromium \
    ca-certificates \
    fonts-liberation \
    fonts-dejavu \
    fonts-noto-color-emoji \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
