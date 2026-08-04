FROM golang:1.26 AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN go build -ldflags="-s -w" -o tusic ./cmd/tusic/main.go

FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates mpv libmpv-dev alsa-utils sqlite3 && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /tusic

# Create a non-root user and add to audio group
RUN useradd -m -G audio user && \
    chown -R user:user /tusic

# Copy only the compiled binary from the builder stage
COPY --from=builder /build/tusic .

# Switch to the non-root user
USER user

# Run the binary
CMD [ "./tusic" ]
