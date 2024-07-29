FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.22 as builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=main

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod /workspace/go.mod
COPY go.sum /workspace/go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . ./

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -ldflags "-X main.developmentMode=false -X main.gitVersion=${VERSION}" -a -o axonops-dev-operator cmd/main.go

# Use distroless as minimal base image to package the axonops-dev-operator binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM --platform=${BUILDPLATFORM:-linux/amd64} gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/axonops-dev-operator /axonops-dev-operator
USER 65532:65532

ENTRYPOINT ["/axonops-dev-operator"]
