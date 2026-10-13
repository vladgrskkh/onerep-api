FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.27.3

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /server /server
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY --from=builder /app/migrations /migrations

USER nobody

EXPOSE 8081

ENTRYPOINT ["/server"]
