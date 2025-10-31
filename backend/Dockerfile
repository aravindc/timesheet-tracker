# Stage 1: The "builder" stage
# This stage uses the full Go SDK to build the application
FROM golang:1.25-trixie  AS builder

RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to leverage Docker's build cache
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed.
RUN go mod download

# Copy the entire source code into the container
COPY . .

# Build the Go app.
# - CGO_ENABLED=0 creates a statically linked binary without any C dependencies.
# - -o /app/main specifies the output file name.
RUN CGO_ENABLED=1 GOOS=linux go build -a -o /app/main .

# ---

# Stage 2: The "final" stage
# This stage uses a minimal base image to create a small and secure final image
FROM debian:trixie-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the compiled binary from the "builder" stage
COPY --from=builder /app/main .

# Expose port 8080 to the outside world
# EXPOSE 8080

# Command to run the executable
CMD ["./main"]
