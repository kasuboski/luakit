# Build stage - builds the luakit gateway binary
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binary for target platform
ARG TARGETPLATFORM
# Parse TARGETPLATFORM to get GOOS and GOARCH (format: os/arch[/variant])
RUN GOOS=$(echo $TARGETPLATFORM | cut -d'/' -f1) && \
    GOARCH=$(echo $TARGETPLATFORM | cut -d'/' -f2) && \
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
    -ldflags="-w -s -extldflags '-static'" \
    -o /usr/local/bin/luakit \
    ./cmd/gateway

# Final stage - minimal scratch image
FROM scratch

# Copy the static binary
COPY --from=builder /usr/local/bin/luakit /luakit

# Set entrypoint for BuildKit gateway frontend
ENTRYPOINT ["/luakit"]
