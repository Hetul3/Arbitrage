FROM golang:1.24-alpine AS base

WORKDIR /app

RUN apk add --no-cache poppler-utils

COPY go.mod ./
RUN go mod download

COPY . .

CMD ["go", "run", "./cmd/polymarket_collector"]
