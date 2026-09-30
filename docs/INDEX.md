# pen-bot

Documentation for the pen-bot Discord bot.

This is the entry point for the documentation tree. The rules governing both
this tree and the Go doc comments in the source live in `DOCUMENTATION.md` at the
repository root.

Documentation of the Go API is not here. Every exported identifier should be
documented by the comment above its declaration, and `go doc` renders those.

## Contents

Pages are added to the toctree below as they are written. It stays commented out
until those pages exist: an enabled toctree naming a missing document warns
rather than failing, so the build would still go green and ship a broken
navigation tree. Uncomment it once the pages land.

Sphinx treats that warning as an error only under `-W`, which
`.github/workflows/docs.yml` passes on every push and pull request. Run it the
same way locally before requesting review:

```bash
uv sync --frozen
uv run sphinx-build -W -b html docs public
```

The output directory is `public/` and is not tracked.

<!--
```{toctree}
:maxdepth: 2
:caption: Contents

SETUP
ARCHITECTURE
```
-->

## Planned documents

SETUP
: Installing dependencies and running the bot locally, including the
  configuration the bot reads from the environment.

ARCHITECTURE
: How the repository is arranged, how a slash command is registered and handled
  end to end, and the dependencies allowed between packages.

OPERATIONS
: Directory with documentation regarding day-to-day running: deployments,
  the database schema and its migrations, and operational notes for scheduled
  background work.

## Indices

* {ref}`genindex`
* {ref}`search`
