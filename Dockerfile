FROM golang:1.25-alpine as builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/tys

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 1337

CMD ["./main"]
