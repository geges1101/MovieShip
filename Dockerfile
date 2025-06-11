FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o main .

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ffmpeg
COPY --from=builder /app/main .
COPY --from=builder /app/internal ./internal
COPY --from=builder /app/web ./web
EXPOSE 8080
CMD ["./main"] 