FROM ghcr.io/nxtcoder17/nix AS builder
WORKDIR /app

RUN --mount=type=bind,source=flake.nix,target=flake.nix \
  --mount=type=bind,source=flake.lock,target=flake.lock \
  <<EOF
nix develop --verbose --command echo "nix setup complete"
EOF

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0
RUN --mount=type=bind,source=flake.nix,target=flake.nix \
  --mount=type=bind,source=flake.lock,target=flake.lock \
  --mount=type=bind,source=go.mod,target=go.mod \
  --mount=type=bind,source=go.sum,target=go.sum \
  --mount=type=bind,target=/app,rw \
  <<EOF
nix develop --command \
  run build output=/out/simple-dns-$TARGETOS-$TARGETARCH
EOF

FROM scratch AS executable
COPY --from=builder /out/simple-dns-* /

FROM gcr.io/distroless/static:nonroot
WORKDIR /home/nonroot
COPY --from=builder --chown=nonroot:nonroot /out/simple-dns-* ./simple-dns
ENTRYPOINT ["./simple-dns"]
