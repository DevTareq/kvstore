# Stage 1: Build the Go binary
FROM golang:1.21 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy Go module files (no go.sum required if no dependencies)
COPY go.mod ./
RUN touch go.sum  # Ensure go.sum exists even if empty

# Copy the entire project source code
COPY . .

# Build the Go binary statically
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o moniepoint ./cmd/server/main.go

# Stage 2: Create a minimal final image (Alpine for smaller footprint)
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/moniepoint .

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./moniepoint"]
