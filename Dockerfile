# syntax=docker/dockerfile:1
FROM golang:1.22.3 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build .

FROM scratch
WORKDIR /app
COPY --from=builder /app/ecs-events-exporter .
CMD ["/app/ecs-events-exporter"]
