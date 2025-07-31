FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o proxy-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/proxy-service .
CMD ["./proxy-service"]