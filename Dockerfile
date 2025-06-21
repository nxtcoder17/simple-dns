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
  --mount=type=bind,target=/app \
  <<EOF
nix develop --command \
  run build out=./bin/simple-dns-$TARGETOS-$TARGETARCH
EOF

FROM gcr.io/distroless/static:nonroot
WORKDIR /home/nonroot
ARG TARGETOS TARGETARCH
COPY --from=builder --chown=nonroot:nonroot /app/bin/simple-dns-$TARGETOS-$TARGETARCH ./simple-dns
ENTRYPOINT ["./simple-dns"]
