FROM golang:1.23 AS builder
WORKDIR /app
RUN apt-get update && apt-get install -y gcc libc6-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/bot

FROM debian:latest
RUN apt-get update && apt-get install -y ca-certificates tzdata sqlite3
ENV TZ=Europe/Moscow
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8443
CMD ["/root/main"]