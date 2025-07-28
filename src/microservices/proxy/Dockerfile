FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod init proxy && go build -o proxy .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/proxy .
CMD ["./proxy"]