FROM golang:1.21-alpine AS builder

ARG VERSION=dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-X main.version=${VERSION}" -o ecr-exporter .

FROM alpine:latest

ARG VERSION=dev
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.title="ECR Prometheus Exporter"
LABEL org.opencontainers.image.description="Prometheus exporter for AWS ECR metrics"

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/ecr-exporter .

EXPOSE 8080

CMD ["./ecr-exporter"]