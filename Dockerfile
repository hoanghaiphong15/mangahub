# Build stage
FROM golang:1.21-alpine AS builder

# Install gcc and libc-dev required for SQLite (CGO)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application (CGO_ENABLED=1 is required for SQLite)
RUN CGO_ENABLED=1 GOOS=linux go build -o mangahub-server ./cmd/api-server/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Install sqlite for runtime
RUN apk add --no-cache sqlite

# Copy the binary from the builder stage
COPY --from=builder /app/mangahub-server .

# Create a data directory for the SQLite database
RUN mkdir -p /app/data

# Expose all 5 protocol ports
EXPOSE 8080 9090 9091 9092

# Run the binary
CMD ["./mangahub-server"]