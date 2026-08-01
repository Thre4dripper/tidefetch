# ── Stage 1: build the web UI ─────────────────────────────────────────
# Pinned to BUILDPLATFORM: the bundle is architecture-independent, and running
# node under QEMU for arm64 costs minutes for an identical result.
FROM --platform=$BUILDPLATFORM node:22-alpine AS webbuild
WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci --no-fund --no-audit
COPY web/ .
RUN npm run build

# ── Stage 2: cross-compile the Go binary (web UI embedded) ────────────
# Also native: Go cross-compiles, so the compiler never runs emulated.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS gobuild
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=webbuild /src/web/dist ./web/dist
ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/tidefetch ./cmd/tidefetch

# ── Stage 3: runtime with the aria2 engine ────────────────────────────
# aria2 is GPL-2.0-or-later and is installed here from the Alpine
# distribution package; see /usr/share/licenses in the image and
# https://github.com/aria2/aria2 for its source code.
FROM alpine:3.22
RUN apk add --no-cache aria2 ca-certificates tzdata \
    && addgroup -g 1000 tidefetch \
    && adduser -u 1000 -G tidefetch -D -h /config tidefetch \
    && mkdir -p /downloads \
    && chown tidefetch:tidefetch /downloads

COPY --from=gobuild /out/tidefetch /usr/local/bin/tidefetch

USER tidefetch
ENV HOME=/config
VOLUME ["/config", "/downloads"]
EXPOSE 8210

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
    CMD wget -q -O /dev/null http://127.0.0.1:8210/ || exit 1

ENTRYPOINT ["tidefetch"]
CMD ["serve", "-host", "0.0.0.0", "-port", "8210", "-dir", "/downloads"]
