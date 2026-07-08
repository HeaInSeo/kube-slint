# Builds the slint-gate CLI binary image.
# Usage: docker run --rm -v $(pwd):/workspace ghcr.io/heainseo/slint-gate \
#          --summary /workspace/artifacts/sli-summary.json \
#          --policy /workspace/.slint/policy.yaml \
#          --output /workspace/slint-gate-summary.json
#
# Image pinning: version tags, not digests — see docs/DECISIONS.md D-019.
# Digest pinning needs an automated update process (e.g. Renovate) that
# doesn't exist here yet; a manually-pinned digest would just rot.
FROM golang:1.25 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -a -trimpath -o slint-gate ./cmd/slint-gate

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/slint-gate .
USER 65532:65532
ENTRYPOINT ["/slint-gate"]
