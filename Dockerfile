# syntax=docker/dockerfile:1

# ─────────────────────────────────────────────────────────────────────────────
# Build stage: use Nix to build the Go binary (with the embedded Svelte webapp)
# as a fully-static executable.
# ─────────────────────────────────────────────────────────────────────────────
FROM nixos/nix:latest AS build

WORKDIR /src

# Copy the full repository. Keeping .git means Nix's flake source filtering
# (gitignore + tracked-file semantics) matches a normal `nix build .`.
COPY . .

# Build the default flake package (static Go binary + embedded webapp) and the
# CA certificate bundle. Both resolve the nixpkgs input pinned in flake.lock.
RUN nix --extra-experimental-features "nix-command flakes" \
      build .#default --out-link /out \
    && nix --extra-experimental-features "nix-command flakes" \
      build nixpkgs#cacert --out-link /cacert \
    && mkdir -p /dist/bin /dist/etc/ssl/certs \
    && cp /out/bin/infra-pipeline-ui /dist/bin/infra-pipeline-ui \
    && cp /cacert/etc/ssl/certs/ca-bundle.crt /dist/etc/ssl/certs/ca-certificates.crt

# ─────────────────────────────────────────────────────────────────────────────
# Runtime stage: only the binary (+ CA bundle for TLS). No shell, no glibc.
# ─────────────────────────────────────────────────────────────────────────────
FROM scratch

COPY --from=build /dist/bin/infra-pipeline-ui /infra-pipeline-ui

# CA bundle so the Go binary can verify https://api.bitbucket.org TLS certs.
COPY --from=build /dist/etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt \
    BBC_CONFIG=/config.yml

EXPOSE 8080

ENTRYPOINT ["/infra-pipeline-ui"]
