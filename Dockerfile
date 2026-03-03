FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o review-proxy .

FROM docker:cli
COPY --from=builder /app/review-proxy /usr/local/bin/review-proxy
ENTRYPOINT ["review-proxy"]
