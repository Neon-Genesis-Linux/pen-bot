FROM golang:1.26.10-alpine3.24 AS build

# VERSION and COMMIT are baked into the binary at build time via the -ldflags
# below. Defaults mirror flake.nix and CI. The .git directory is excluded from
# the build context, so the real commit has to be supplied by the host, e.g.:
#   docker build --build-arg COMMIT=$(git rev-parse HEAD) .
ARG VERSION=0.0.1-alpha
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

FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /app

COPY --from=build /output/pen-bot /app/pen-bot

CMD ["/app/pen-bot"]