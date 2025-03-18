FROM golang:1.20 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod tidy
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
RUN chmod +x /root/main
EXPOSE 8080
CMD ["/root/main"]
