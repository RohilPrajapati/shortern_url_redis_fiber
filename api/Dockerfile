# --- Stage 1: Build the Go Binary ---
FROM golang:alpine AS builder

# Install build dependencies if needed (e.g., git/certs)
RUN apk add --no-cache git

WORKDIR /build

# Copy dependency manifests first to leverage Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build a statically linked binary (Optimized for Alpine scratch runtimes)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .

# --- Stage 2: Minimal Runtime Environment ---
FROM alpine:latest

RUN apk --no-cache add ca-certificates

# Fix: Create the system group first explicitly, then create the user inside it
RUN addgroup -S appuser && adduser -S -G appuser -h /app -D -H appuser

WORKDIR /app

# Senior Optimization: Set the owner to appuser right when copying! No separate RUN chown needed.
COPY --from=builder --chown=appuser:appuser /build/main /app/

# Switch context to the non-root execution layer
USER appuser

EXPOSE 3000

CMD ["./main"]