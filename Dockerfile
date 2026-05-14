FROM golang:1.26.2-alpine3.23 AS build

WORKDIR /app

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -trimpath -ldflags="-s -w" -o /output/pen-bot ./cmd/pen-fun

FROM gcr.io/distroless/static-debian13

WORKDIR /app
USER nonroot

COPY --from=build /output/pen-bot /app/pen-bot

CMD ["/app/pen-bot"]
