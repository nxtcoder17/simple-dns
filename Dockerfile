FROM ghcr.io/nxtcoder17/nix AS builder
WORKDIR /app

RUN --mount=type=bind,source=flake.nix,target=flake.nix \
  --mount=type=bind,source=flake.lock,target=flake.lock \
  <<EOF
nix develop --verbose --command echo "nix setup complete"
EOF

ENV CACHE_DIR=/cache

ENV GOMODCACHE=$CACHE_DIR/gomodcache
ENV GOCACHE=$CACHE_DIR/gocache

ENV CGO_ENABLED=0
RUN --mount=type=bind,source=flake.nix,target=flake.nix \
  --mount=type=bind,source=flake.lock,target=flake.lock \
  --mount=type=bind,source=go.mod,target=go.mod \
  --mount=type=bind,source=go.sum,target=go.sum \
  --mount=type=cache,target=$GOMODCACHE \
  --mount=type=cache,target=$GOCACHE \
  <<EOF
time nix develop --command go mod download -x -json
echo "DOWNLOADED go modules"
EOF

SHELL ["bash", "-c"]
RUN --mount=type=bind,source=.,target=/app \
  --mount=type=cache,target=$GOMODCACHE \
  --mount=type=cache,target=$GOCACHE \
  <<EOF
nix develop --command run build output=/out/simple-dns
EOF

FROM gcr.io/distroless/static:nonroot
WORKDIR /home/nonroot
COPY --from=builder --chown=nonroot:nonroot /out/simple-dns ./simple-dns
ENTRYPOINT ["./simple-dns"]
