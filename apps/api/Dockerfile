# ─── Stage 1: Builder ────────────────────────────────────────────────────────
# Uses the official Go image to compile the binary.
# CGO_ENABLED=0 produces a fully static binary that can run in a scratch image.
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Download dependencies first so Docker layer cache is reused when only source changes.
COPY go.mod go.sum ./
RUN go mod download

# Copy the remaining source.
COPY . .

# Build a statically linked binary.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /app/server ./cmd/server

# ─── Stage 2: Runtime ────────────────────────────────────────────────────────
# distroless/static contains only the necessary runtime files with no shell,
# package manager, or OS utilities — minimises the attack surface.
FROM gcr.io/distroless/static:nonroot

# Copy the compiled binary from the builder stage.
COPY --from=builder /app/server /server

# Run as the nonroot user provided by the distroless image.
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/server"]
