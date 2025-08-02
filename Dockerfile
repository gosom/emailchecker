FROM golang:1.24.4-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o checker cmd/email-checker/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata sqlite
WORKDIR /app

COPY --from=builder /app/checker .

RUN mkdir -p /app/data

EXPOSE 8080
CMD ["./checker", "server", "--port", ":8080"]
