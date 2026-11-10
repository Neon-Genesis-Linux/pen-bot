FROM golang:1.26-alpine3.24 AS build

WORKDIR /app

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -trimpath -ldflags="-s -w" -o /output/pen-bot ./cmd/pen-fun

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
