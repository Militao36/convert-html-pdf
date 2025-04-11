# Etapa 1: Build com Go instalado manualmente
FROM ubuntu:22.04 AS builder

ENV DEBIAN_FRONTEND=noninteractive

# Instalar dependências
RUN apt-get update && apt-get install -y \
    wget \
    ca-certificates \
    git \
    build-essential

# Instalar Go manualmente
ENV GOLANG_VERSION=1.21.9
RUN wget https://go.dev/dl/go$GOLANG_VERSION.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go$GOLANG_VERSION.linux-amd64.tar.gz && \
    ln -s /usr/local/go/bin/go /usr/bin/go

ENV PATH=$PATH:/usr/local/go/bin

WORKDIR /app
COPY . .

RUN go mod tidy
RUN go build -o server .

# Etapa 2: Runtime com wkhtmltopdf e Ubuntu
FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    wkhtmltopdf \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
