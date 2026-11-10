# AGENTS.md

Normative sources, read before contributing. This file only adds what they
cannot tell you:

- [CONTRIBUTING.md](CONTRIBUTING.md) — checks to run and report, diff boundaries,
  Conventional Commits format.
- [AI_POLICY.md](AI_POLICY.md) — disclosure for tool-assisted work. Applies to
  agent-written code.
- [DOCUMENTATION.md](DOCUMENTATION.md) — how documentation is written, which
  content belongs where, and the Go doc comment rules CI does not check. Read it
  before adding exported identifiers.

## Verify

CI runs these in order (`.github/workflows/ci.yml`):

```bash
test -z "$(gofmt -l .)"
go mod tidy && git diff --exit-code go.mod go.sum
go vet ./...
go test ./...
./custom-gcl run --timeout=5m ./...
```

`docs.yml` builds the documentation tree and fails on any warning, and both
workflows must pass:

```bash
uv sync --frozen
uv run sphinx-build -W -b html docs public
```

- **Use `./custom-gcl`, never `golangci-lint`.** Stock `golangci-lint run` aborts
  with `plugin(nilaway): plugin "nilaway" not found`, because `nilaway` is a
  module plugin. `custom-gcl` is a gitignored build of golangci-lint v2.13.2 +
  NilAway from `.custom-gcl.yml`; `nix develop` builds it on shell entry. Re-run
  `golangci-lint custom` after editing `.custom-gcl.yml` or the pinned version.
- **Focused runs:** `go test ./internal/db/...`, `go test -run TestExpiry
  ./internal/db/`. Tests are pure unit tests — no Discord or network needed, and
  the database tests need no server either: they open a SQLite file in a
  temporary directory. Only `internal/db` has tests; everything else reports "no
  test files". Unexported-helper tests live in `package db`, public-API tests in
  `package db_test`.
- `pre-commit run --all-files` is a superset of `ci.yml` (adds staticcheck,
  gosec, govulncheck). There is no hook for the docs build, so run that by hand.
  `go-vulncheck-repo-mod` is **not** in CI and fails whenever the local
  toolchain trails the patched Go release. That is a toolchain-version signal,
  not a code defect — bump Go rather than deleting the hook.

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
- `internal/moderation` is registered by `main.go`; it holds `/self-timeout`.

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

- The database is SQLite, through `modernc.org/sqlite`. The driver is pure Go, so
  the build stays `CGO_ENABLED=0` and the distroless image is unchanged. It
  depends on `modernc.org/libc`, and upstream asks that the two stay in step:
  Go's module resolution will not select a `libc` older than the one
  `modernc.org/sqlite` requires, but a newer one can be selected, and an
  incompatible one fails to compile rather than misbehaving.
- Timestamps are **Unix milliseconds in an `INTEGER` column**, declared
  `DATETIME`. Do not store them as text: the driver writes text while a column
  default writes an integer, and SQLite then orders the column by kind rather
  than by instant, so rows come back in the wrong order. The DSN parameters
  `_time_integer_format=unix_milli` and `_inttotime=1` are what make the
  driver read and write that representation.
- The driver converts a timestamp on the way out **only when the column's
  declared type is `DATE`, `DATETIME` or `TIMESTAMP`**. An aggregate or an
  arithmetic expression over such a column has no declared type, so it comes
  back as an integer and scanning it into a `time.Time` fails. Select the
  column itself, or convert the integer in SQL.
- The driver reads an integer back as milliseconds only when it is larger than
  1e12, which is September 2001. A timestamp before that is silently read as
  seconds. Every timestamp here is written by the running bot, so this cannot
  happen; a column holding a historic timestamp would need care.
- Migrations are `//go:embed`ed from `internal/db/migrations/` and applied by the
  running bot at startup (`db.ApplyMigrations`). There is no migration CLI. Add
  `NNN_name.sql`; the filename is recorded in `schema_migrations` and the file
  runs in one transaction.
- A migration file is handed to the driver as one statement, not split on
  semicolons, so a semicolon inside a string literal is safe. Keep migrations
  to plain DDL anyway.
- Primary keys are `INTEGER PRIMARY KEY AUTOINCREMENT`. The `AUTOINCREMENT` is
  required: without it SQLite reuses an id after a delete, which breaks the
  keyset pagination that walks a log by `(created_at, id)`.
- An index that serves a query has to carry the order the query asks for. The
  moderation log index ends with `id` because the pagination cursor compares
  the pair `(created_at, id)`: without it SQLite sorts every page again with a
  temporary B-tree over the whole table. `internal/db/migrations_test.go` asserts
  the query plan, so a change that reintroduces the sort fails the build.
- `db.GlobalDB()` is filled asynchronously, after `Start` has already returned to
  its signal wait (`internal/core/start.go`), and stays nil forever if the
  database cannot be opened — the bot runs happily without a database. Callers
  must tolerate nil rather than assume a live handle.
- SQLite has one writer. The DSN sets `journal_mode(WAL)`, a `busy_timeout` and
  `_txlock=immediate`, and the pool is small. Do not raise the pool size to
  raise write throughput: writes are serialised regardless.
- `_txlock=immediate` is load-bearing, not a default. A deferred transaction
  that reads and then writes cannot always take the write lock afterwards, and
  fails with `SQLITE_BUSY` in a way the busy timeout does not retry.
- The cache in `internal/db/cache.go` is a process-local map with TTL, not Redis.

## Configuration

- Every setting is an env var read at call time. **Nothing loads `.env`** — there
  is no dotenv library and `.envrc` is only `use flake`. `compose.yaml` injects it
  via `env_file`; for a local run, export the variables yourself. Full list in
  `.env.example`.
- `ENV=production` selects INFO, anything else DEBUG (`internal/logger`).
- `GUILD_ID` is optional: set it to sync commands to one guild (instant) instead
  of globally (slow to propagate).
- `DB_PATH` names the database file. `DB_BOT_INSTANCE_ID` names the file when
  `DB_PATH` is unset, so that instances sharing a directory each get their own.
  The container image sets `DB_PATH=/data/pen-bot.db` and mounts a volume there.
- The compose file is `compose.yaml` (Compose v2); use `docker compose up`, or
  `docker compose watch` to honour the `develop.watch` rebuild. Keep
  `.dockerignore` — it is what keeps `.env` and the 52MB `custom-gcl` out of the
  build context.

## Workflow

- Conventional Commits, lowercase type, optional scope; match the scopes used in
  recent `git log`.
- Do not push, open a pull request, or comment on an issue unless asked.
- Never add `Signed-off-by`, `Co-developed-by`, or any other certification on a
  contributor's behalf (see AI_POLICY.md).
- `.env` is gitignored and secret; keep it out of diffs.
