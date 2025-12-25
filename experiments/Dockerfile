FROM golang:1.24-alpine AS base

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . ./

RUN go test ./...

CMD ["go", "run", "."]
