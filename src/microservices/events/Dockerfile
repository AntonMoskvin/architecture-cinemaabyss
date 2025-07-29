FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go get github.com/segmentio/kafka-go && go build -o events .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/events .
CMD ["./events"]