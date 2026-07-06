# Etapa 1: Build
FROM golang:1.26-bookworm AS builder

WORKDIR /app
COPY . .

RUN go mod tidy
RUN go build -o server .

# Etapa 2: Runtime com Chromium headless pronto para container
FROM chromedp/headless-shell:latest

ENV CHROME_PATH=/headless-shell/headless-shell

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8080

ENTRYPOINT ["/app/server"]
