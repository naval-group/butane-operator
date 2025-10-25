# Build the manager binary
FROM cgr.dev/chainguard/go@sha256:2355a7b35abafd8e0b4eefcefe2b0e7eafbd2cbc6a7cdc82f48abf7b9e9cde60 AS builder

# Remove internal proxy configuration for open source version
# ENV GONOPROXY=gitlab.ftf.everest.mylittleforge.org/*
# ENV GOPROXY=https://nexus.main.dmz.mylittleforge.org/repository/go-proxy,direct
# ENV GOSUMDB=off

ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/controller/ internal/controller/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

# Refer to https://images.chainguard.dev/directory/image/static for more details
FROM cgr.dev/chainguard/static@sha256:cd71a91840a1a678638e9b9e6e1b8da9074c35b39b6c0be0e8b50a3b4b5b4ca2 AS runtime

WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
