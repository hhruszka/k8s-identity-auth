# Stage 1: Build the Go application
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to leverage Docker cache
# This step ensures that dependencies are re-downloaded only if go.mod or go.sum changes
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY main.go ./

# Build the Go application
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o testapp main.go

# Stage 2: Create the final, minimal runtime image
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Create a non-root user and group
# This is a security best practice for Kubernetes deployments.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Copy the compiled executable from the builder stage
COPY --from=builder /app/testapp .

# Command to run the application
# Use the absolute path to the executable.
CMD ["/app/testapp"]