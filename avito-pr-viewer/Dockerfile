FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN go build -o pr-reviewer ./cmd/server

FROM alpine:3.20

RUN adduser -D appuser
WORKDIR /home/appuser

COPY --from=builder /app/pr-reviewer ./pr-reviewer

USER appuser

EXPOSE 8080

ENV APP_PORT=8080

CMD ["./pr-reviewer"]
