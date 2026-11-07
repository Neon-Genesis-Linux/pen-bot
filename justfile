dev:
    docker compose up --build --watch
test:
    gotestsum --format testname
lint:
    ./custom-gcl run
    hadolint Dockerfile
format:
    gofumpt -w -l .
    yamlfmt .
    dockerfmt -c Dockerfile
check:
    just test
    just lint
    just format
