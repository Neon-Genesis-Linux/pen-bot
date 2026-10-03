# AGENTS.md

Normative sources, read before contributing. This file only adds what they
cannot tell you:

- [CONTRIBUTING.md](CONTRIBUTING.md) — checks to run and report, diff boundaries,
  Conventional Commits format.
- [AI_POLICY.md](AI_POLICY.md) — disclosure and the `Assisted-by:` trailer for
  tool-assisted work. Applies to agent-written code.
- [DOCUMENTATION.md](DOCUMENTATION.md) — **not on `main` yet.** Both this file and
  `CONTRIBUTING.md` link it, but it currently exists only on the local `docs`
  branch (`git show docs:DOCUMENTATION.md`). It holds the Go doc comment rules CI
  does not check. Read it before adding exported identifiers; expect the link to
  404 until `docs` merges.

## Verify

CI runs these in order (`.github/workflows/ci.yml`):

```bash
test -z "$(gofmt -l .)"
go mod tidy && git diff --exit-code go.mod go.sum
go vet ./...
go test ./...
./custom-gcl run --timeout=5m ./...
```

- **Use `./custom-gcl`, never `golangci-lint`.** Stock `golangci-lint run` aborts
  with `plugin(nilaway): plugin "nilaway" not found`, because `nilaway` is a
  module plugin. `custom-gcl` is a gitignored build of golangci-lint v2.13.2 +
  NilAway from `.custom-gcl.yml`; `nix develop` builds it on shell entry. Re-run
  `golangci-lint custom` after editing `.custom-gcl.yml` or the pinned version.
- **Focused runs:** `go test ./internal/db/...`, `go test -run TestExpiry
  ./internal/db/`. Tests are pure unit tests — no Postgres, Discord, or network
  needed. Only `internal/db` has tests; everything else reports "no test files".
  Unexported-helper tests live in `package db`, public-API tests in
  `package db_test`.
- `pre-commit run --all-files` is a superset (adds staticcheck, gosec,
  govulncheck). `go-vulncheck-repo-mod` is **not** in CI and fails whenever the
  local toolchain trails the patched Go release. That is a toolchain-version
  signal, not a code defect — bump Go rather than deleting the hook.

## Layout

One root module, `github.com/Neon-Genesis-Linux/pen-bot`, one binary:
`cmd/pen-fun`. `internal/core` owns process-wide state and startup;
`community`, `moderation`, `db`, `logger` are libraries it does not import back.

- **Registration order is a contract.** `community.Register()` and any
  `core.RegisterIntents(...)` must run *before* `core.Start()` (see
  `cmd/pen-fun/main.go`). Both append to package vars in
  `internal/core/handler.go` that `Start` reads when syncing commands and opening
  the gateway, so registering afterwards silently syncs nothing.
- **A command takes two calls:** `core.RegisterCommands(...)` for the Discord
  definition, `core.Mux().SlashCommand("/name", h)` for the handler. Subcommands
  nest via `core.Mux().Route("/name", ...)` with `"/sub"` paths — see
  `internal/community/xkcd.go`. Slash commands only; prefix commands were
  removed in a breaking change.
- `internal/logger` works solely through a blank import; its `init` is the whole
  package. Drop `_ ".../internal/logger"` and the build still compiles while the
  bot silently falls back to slog's default handler.
- `internal/moderation` is an empty stub that `main.go` does not register.

## Lint rules that reject correct-looking code

`go vet` and `go test` pass on code `./custom-gcl` rejects.

- `exhaustruct_v5` (explicit mode) covers `.../pen-bot/.*`: every field of your
  own structs must be set at every literal. Adding one field breaks all its
  literal sites. Third-party structs are exempt except the three listed in
  `.golangci.yml` `ignore-patterns`.
- `goconst` fires at 3 occurrences of a 3+ character string — extract a `const`.
  That is why `xkcdCommand` and `tldrCommand` are constants.
- `intrange`: `for i := range n`. `errorlint`: `errors.Is`, not `==`.
- `goimports` `local-prefixes` is the module path, so three groups: stdlib,
  third-party, `github.com/Neon-Genesis-Linux/pen-bot/...`.
- `revive` runs a curated rule list that deliberately omits `exported` and
  `package-comments`, so absent doc comments slip through the linter.
- `_test.go` files are exempt from `errcheck`, `gosec`, `unparam`,
  `exhaustruct_v5`.

## Database

- Migrations are `//go:embed`ed from `internal/db/migrations/` and applied by the
  running bot at startup (`db.ApplyMigrations`). There is no migration CLI. Add
  `NNN_name.sql`; the filename is recorded in `schema_migrations` and the file
  runs in one transaction.
- The runner splits each file on `;`, so no semicolon may appear inside a string
  literal, a dollar-quoted body, or a `DO $$` block. Keep migrations to plain
  DDL.
- `db.GlobalDB()` is filled asynchronously, after `Start` has already returned to
  its signal wait (`internal/core/start.go`), and stays nil forever if Postgres is
  unreachable — the bot runs happily without a database. Callers must tolerate
  nil rather than assume a live handle.
- The cache in `internal/db/cache.go` is a process-local map with TTL, not Redis.

## Configuration

- Every setting is an env var read at call time. **Nothing loads `.env`** — there
  is no dotenv library and `.envrc` is only `use flake`. `compose.yaml` injects it
  via `env_file`; for a local run, export the variables yourself. Full list in
  `.env.example`.
- `ENV=production` selects INFO, anything else DEBUG (`internal/logger`).
- `GUILD_ID` is optional: set it to sync commands to one guild (instant) instead
  of globally (slow to propagate).
- For the database, `DATABASE_URL` wins outright; otherwise `DB_HOST`, `DB_PORT`,
  `DB_USER`, `DB_PASSWORD`, `DB_NAME` are composed, and `DB_BOT_INSTANCE_ID`
  renames the database to `pen_bot_<id>`. `POSTGRES_PASSWORD` is compose-only.
- The compose file is `compose.yaml` (Compose v2); README's `docker-compose up`
  is stale. Use `docker compose up`, or `docker compose watch` to honour the
  `develop.watch` rebuild. Keep `.dockerignore` — it is what keeps `.env` and the
  52MB `custom-gcl` out of the build context.

## Workflow

- Conventional Commits, lowercase type, optional scope; match the scopes used in
  recent `git log`.
- Do not push, open a pull request, or comment on an issue unless asked.
- Never add `Signed-off-by`, `Co-developed-by`, or any other certification on a
  contributor's behalf (see AI_POLICY.md). An `Assisted-by:` trailer is the
  sanctioned form.
- `.env` is gitignored and secret; keep it out of diffs.
