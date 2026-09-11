FROM golang:1.26-alpine3.24 AS build

# VERSION and COMMIT are baked into the binary at build time via the -ldflags
# below. Defaults mirror flake.nix and CI. The .git directory is excluded from
# the build context, so the real commit has to be supplied by the host, e.g.:
#   docker build --build-arg COMMIT=$(git rev-parse HEAD) .
ARG VERSION=dev
ARG COMMIT=dev

WORKDIR /app

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -trimpath \
    -ldflags="-s -w -X github.com/Neon-Genesis-Linux/pen-bot/internal/core.Version=${VERSION} -X github.com/Neon-Genesis-Linux/pen-bot/internal/core.Commit=${COMMIT}" \
    -o /output/pen-bot ./cmd/pen-fun

# The database lives on a volume mounted here. The directory is created in
# this stage so that it can be given to the unprivileged user the runtime
# image runs as: Docker seeds a fresh named volume with the ownership of the
# directory it covers, so a volume created from this image is writable by that
# user without the operator having to fix its permissions by hand.
RUN mkdir -p /data && chown 65532:65532 /data

FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /app

COPY --from=build /output/pen-bot /app/pen-bot
COPY --from=build --chown=65532:65532 /data /data

ENV DB_PATH=/data/pen-bot.db

CMD ["/app/pen-bot"]
