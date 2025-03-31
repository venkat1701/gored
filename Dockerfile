# Use Go 1.24.1 instead of 1.21
FROM golang:1.24.1-alpine AS builder

WORKDIR /app

# Copy go.mod and download dependencies
COPY go.mod ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o gored .

# Create final runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy compiled binary from builder stage
COPY --from=builder /app/gored .

EXPOSE 7171

CMD ["./gored"]
