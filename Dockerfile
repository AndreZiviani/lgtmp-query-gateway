FROM golang:1.24-alpine AS builder
ARG CGO_ENABLED=0
WORKDIR /app

COPY go.mod go.sum /app/
RUN go mod download

COPY . /app
RUN go build -o /app/app cmd/gateway/main.go

FROM alpine:3
COPY --from=builder /app/app /app/app
CMD ["/app/app"]