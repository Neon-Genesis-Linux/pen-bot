FROM kgrv/golang AS builder

USER root

RUN apk add ca-certificates

WORKDIR /app
COPY . .

RUN go mod download

RUN go build -ldflags="-s -w -extldflags '-static'" -o pen-bot

FROM scratch

WORKDIR /app
COPY --from=builder --chown=65534:65534 /app/pen-bot .
COPY --from=builder /etc/ssl /etc/ssl

USER 65534:65534

ENV DISCORD_TOKEN=""

ENTRYPOINT ["./pen-bot"]
