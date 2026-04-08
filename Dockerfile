# Stage 1: build
FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /waspctl .

# Stage 2: run
FROM alpine:3.21

COPY --from=builder /waspctl /usr/local/bin/waspctl

ENTRYPOINT ["waspctl"]
