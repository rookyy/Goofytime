FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o stundenerfassung .

FROM alpine:3.21

RUN adduser -D -u 1000 appuser

COPY --from=builder /app/stundenerfassung /app/stundenerfassung
COPY --from=builder /app/templates /app/templates

RUN mkdir /app/data && chown appuser /app/data

USER appuser
WORKDIR /app

ENV DB_PATH=/app/data/stunden.db
ENV PORT=8080

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/login || exit 1

CMD ["./stundenerfassung"]
