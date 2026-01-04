FROM golang:1.25.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM ubuntu:latest AS dict-builder

RUN apt-get update && apt-get install -y wamerican && \
    cp /usr/share/dict/words /words && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/server .
COPY --from=dict-builder /words /usr/share/dict/words

EXPOSE 8080

CMD ["./server"]
