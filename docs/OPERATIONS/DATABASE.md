# Database

The bot stores its data in a SQLite database, in a single file on disk. There is
no database server to run, no credentials to manage and no separate container
in the deployment.

## Where the file lives

`DB_PATH` names the file. The directory holding it is created if it does not
exist.

```bash
DB_PATH=/data/pen-bot.db
```

When `DB_PATH` is unset, the file name comes from `DB_BOT_INSTANCE_ID`, which
exists so that several instances can share one directory without seeing each
other's data:

```bash
DB_BOT_INSTANCE_ID=pen-bot-1   # writes data/pen-bot-pen-bot-1.db
```

Characters that are not letters, digits, `-` or `_` are dropped from the
instance id, so the name cannot escape the directory it is in.

## The container

The image sets `DB_PATH=/data/pen-bot.db`, and the compose file mounts a
volume there:

```yaml
services:
  app:
    volumes:
      - bot_data:/data
```

The image creates `/data` owned by the unprivileged user the container runs as.
Docker seeds a fresh named volume with the ownership of the directory it
covers, so the volume is writable without any manual permission fixing.

## The schema

Three tables hold the bot's data, plus one that records which migrations have
run.

`guilds`
: One row per guild, holding the guild's settings as a JSON document.

`users`
: One row per user per guild, holding the user's profile as a JSON document.

`moderation_logs`
: One row per moderation action, with the moderator, the target, the action
  and the reason.

`schema_migrations`
: One row per applied migration, recording the file name and when it ran.

Timestamps are stored as Unix milliseconds in an `INTEGER` column, declared
`DATETIME`. Read them by selecting them into a `time.Time`; the driver converts
them.

The driver converts a timestamp only when the column it came from is declared
`DATE`, `DATETIME` or `TIMESTAMP`. An aggregate over such a column, such as
`MAX(created_at)`, has no declared type and comes back as an integer, so
scanning it into a `time.Time` fails. Select the column itself instead.

## Migrations

Migrations are files in `internal/db/migrations/`, named `NNN_name.sql`. The
running bot applies them at startup, before it serves any command, so a
deployment needs no separate migration step. Each file runs in one transaction,
and its name is recorded in `schema_migrations` so it is not applied twice.

A migration must be plain DDL. The file is handed to the database as one
statement, so a semicolon inside a string literal is safe.

The name of the file is what records it as applied. Editing a file that a
database has already run changes nothing for that database: it keeps the
schema it has. To change the schema of a database that is already in use, add a
new migration file rather than editing an old one.

## Backing up

The database is one file, but while the bot is running it also has a write-ahead
log beside it. Copying only the `.db` file would miss the most recent writes.

Take a consistent copy with `VACUUM INTO`, which writes a complete snapshot
without stopping the bot:

```bash
sqlite3 /data/pen-bot.db "VACUUM INTO '/data/backup-$(date +%F).db'"
```

Restore by stopping the bot and putting the snapshot back in place:

```bash
docker compose stop app
cp backup-2026-11-10.db /data/pen-bot.db
rm -f /data/pen-bot.db-wal /data/pen-bot.db-shm
docker compose start app
```

The bot applies any pending migrations on the next start.

## Inspecting the data

Any SQLite client can open the file, and the `sqlite3` command line tool is
enough for most questions:

```bash
sqlite3 /data/pen-bot.db ".tables"
sqlite3 /data/pen-bot.db "SELECT * FROM moderation_logs ORDER BY created_at DESC LIMIT 10;"
```

Stop the bot first if you intend to write, so that the two do not contend for
the same lock.

## Concurrency

SQLite allows one writer at a time. The bot runs with the write-ahead log
enabled, a busy timeout, and a small connection pool, which is enough for the
traffic one guild produces. Writes are serialised no matter how large the pool
is, so raising the pool size will not make them faster.
