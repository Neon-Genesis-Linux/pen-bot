dev:
    docker compose up --build --watch
test:
    gotestsum --format testname
lint:
    golangci-lint run
    hadolint Dockerfile
format:
    gofumpt -w -l .
    yamlfmt .
    dockerfmt -c Dockerfile
check:
    just test
    just lint
    just format
