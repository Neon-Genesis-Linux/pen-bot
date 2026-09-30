# Contributing

Contributions are welcome. This document covers the mechanics of preparing one:
what to read before editing, what to run before asking for review, and the
boundaries of a change a maintainer will accept.

These rules apply whether a change is written by hand or with assistance. Using
a tool is never a reason to hold a lower bar, and never a reason to skip a step.
If you used AI or LLM tooling, [AI_POLICY.md](AI_POLICY.md) also applies.

## Before editing

Read [DOCUMENTATION.md](DOCUMENTATION.md) first. It defines where each kind of
content belongs, and its rules for Go doc comments are the ones CI will not
check for you.

Read the files you are about to change rather than inferring their contents from
fragments.

## Verify

Run the checks the repository defines, and report the real output:

```bash
test -z "$(gofmt -l .)"
go mod tidy && git diff --exit-code go.mod go.sum
go vet ./...
go test ./...
./custom-gcl run --timeout=5m ./...
```

Do not report a check as passing unless it was run and passed. Do not summarise
test output into a claim it does not support. If a check could not be run, say
so plainly and name what was not verified. A maintainer reading an honest gap
can act on it; one reading a fabricated pass cannot.

If a claim in a document cannot be verified from the repository, check the
normative source before repeating it. Documentation that states a rule from
memory is worse than documentation that omits the rule.

## Boundaries

- Do not push, open a pull request, or comment on an issue unless asked.
- Do not weaken or disable a check in order to make a change pass.
- Do not commit secrets, tokens, or `.env` contents. See
  [`.env.example`](.env.example).
- Leave unrelated changes out of the diff. Unrequested reformatting makes a
  review harder.
- Submit only code you can account for the origin of.
- Stop and ask rather than guessing when the intent of a change is ambiguous.

## Commit messages

The repository uses [Conventional Commits](https://www.conventionalcommits.org/):
`feat:`, `fix:`, `chore:`, `docs:`, `readme:`, `refactor:`, `test:`, `ci:`, with
an optional scope, and `!` before the colon for a breaking change. Types are
lowercase. Match the scope style of recent commits in `git log`.

## References

- [Submitting patches: the essential guide to getting your code into the kernel](https://docs.kernel.org/process/submitting-patches.html),
  the reference for keeping one logical change per commit and for describing it
  so it still makes sense months later. This repository does not use its
  Developer Certificate of Origin or its email submission flow.
