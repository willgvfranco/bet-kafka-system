FROM golang:1.23-alpine AS builder

ARG SERVICE

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/service ./cmd/${SERVICE}/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/service /service
ENTRYPOINT ["/service"]
