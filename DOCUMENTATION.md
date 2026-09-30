# Documentation

Documentation is written in two systems, and they are not interchangeable. **Go
doc comments** document the API and live in the source. **Markdown** documents
everything else (setup, architecture, operations) and lives in `docs/`. A
runbook in a doc comment and an API reference in `docs/` are both wrong.

> **Adding a Go identifier?** Write a doc comment above it that starts with the
> identifier's name, in complete sentences, saying what a caller must know. That
> is all. The rules below and `gofmt` cover the rest.

## Where it lives

| Subject | Format | Location |
|---|---|---|
| Exported Go identifiers | doc comment | the `.go` file, above the declaration |
| Package purpose | package comment | in exactly one file of the package |
| Invariants, algorithms, gotchas | plain comment | inside the function body |
| Setup, configuration, deployment | Markdown | `docs/` |
| Orientation, badges, quick links | Markdown | `README.md` |
| Contributor rules | Markdown | root, alongside this file |

## Go doc comments

### What to write

A doc comment is the comment immediately preceding a top-level declaration, with
no blank line between the comment and the declaration:

```go
// RegisterCommands appends application command definitions to be synced with Discord.
func RegisterCommands(cmds ...discord.ApplicationCommandCreate) {
```

Every exported name should have one. Unexported names get one only when the
code would not be clear without it. A comment that restates the name is noise.

1. **Start with the name of the thing.** `RegisterCommands appends…`, not `This
   function appends…`. For a package, start with `Package community`; for a
   command, with the program's name.
2. **Write complete sentences.** The first becomes the summary line in `go doc`,
   so it has to stand alone. Name parameters and results directly: `Quote
   returns a literal representing s`. Say "reports whether" for booleans,
   never "whether or not".
3. **Explain why, not how.** If a reader could derive it from the signature,
   leave it out. Never describe the implementation: when the algorithm changes,
   the comment should not have to.
4. **Document what a caller must know:** the zero value, concurrency guarantees
   stronger than the default (functions are assumed concurrent-safe, types
   single-goroutine), and special cases.
5. **Use one receiver name across a type's methods** so `go doc` lists them
   consistently.
6. **One sentence per line.** `gofmt` preserves your line breaks rather than
   rewrapping, so diffs stay limited to the lines that actually changed.
   Rewrapping happens when `go doc` and pkg.go.dev print, not in the source.

### Package comments

One file in each package carries a comment starting `Package <name>`, placed
directly above the `package` clause rather than above a function. Keep it to one
file: comments in a second file are concatenated, not ignored.

A package with no exported API still needs one when importing it *does* something.
Such a package is imported for its side effect, with a blank import
(`import _ "path/to/pkg"`). Without that note the import looks removable, and the
obvious cleanup (deleting the `init`) breaks logging everywhere with no compiler
complaint:

```go
// Package logger installs the process-wide slog handler on import, writing
// logfmt records to stdout at INFO when ENV is "production" and DEBUG otherwise.
// Each record carries the file and line it came from. Import it for its side
// effect.
package logger
```

### Syntax

`gofmt` has canonicalised doc comments since Go 1.19, so use its syntax rather
than hand-formatting.

| Feature | Form | Notes |
|---|---|---|
| Paragraph | Unindented, non-blank lines | Separate paragraphs with a blank line |
| Heading | `# Heading` | Unindented, one line, blank line either side |
| Link | `[Text]`, defined as `[Text]: URL` at the end | Keeps URLs out of the prose |
| Doc link | `[Name]`, `[pkg.Name]`, `[*pkg.Type]` | Must be delimited by punctuation or whitespace |
| List | `- item` or `1. item` | Optional blank line before; continuation indented four spaces; no nesting |
| Code block | Indent one tab | Blank line before; not nested lists or headings |
| Grouped `const`/`var` | One comment on the group | Short end-of-line comments per entry |
| Note | `TODO(name): …`, `BUG(name): …` | Two or more capitals, then an identifier |
| Deprecation | Paragraph starting `Deprecated: ` | Say what to use instead |

Directive comments such as `//go:generate` are excluded from rendered
documentation, and `gofmt` moves them to the end of the comment. A directive is
lower case throughout up to the colon, so a Note such as `TODO(name):` is
ordinary text and is kept.

### Traps

These fail silently. The comment still renders, just wrongly.

- **Indenting a wrapped prose line.** An indented span is a code block, so one
  stray indent turns a sentence into preformatted text.
- **Not indenting a wrapped list item**, for the same reason.
- **Leaving a list or code block unindented**, which reads as a paragraph.
- **An undelimited doc link.** `map[ast.Expr]TypeAndValue` contains no link; a
  bracket needs punctuation or whitespace on both sides.

### Example

Good, and neither the constraint nor its reason is visible from the signature:

```go
// RegisterCommands appends application command definitions to the set Start
// syncs. A definition with no handler routed for it reaches Discord and does
// nothing.
func RegisterCommands(cmds ...discord.ApplicationCommandCreate) {
```

Bad. It restates the name, starts in lower case, and adds nothing. The content
is already in the signature:

```go
// returns the user. gets the user from the database
func GetUser(ctx context.Context, db *DB, id string) (*User, error) {
```

## Documentation tree

`docs/` holds narrative documentation: how to run the bot, how it is put
together, how to operate it. `docs.yml` builds the tree on every push and pull
request, so a page that does not build does not merge.

The tree is MyST markdown, so a page is a `.md` file that Sphinx parses rather
than plain CommonMark. The rest of the repository is already markdown, and one
format means a contributor does not learn a second one to edit a page. The
built site is identical either way; this is about who can write the source.

### Building

```bash
uv sync --frozen
uv run sphinx-build -W -b html docs public
```

`pyproject.toml` and `uv.lock` sit at the repository root, not in `docs/`. That
placement is load-bearing: uv puts the virtualenv next to the manifest, so a
manifest in `docs/` would put `.venv` inside the Sphinx source tree, and Sphinx
would then treat every installed package's README and licence file as project
content. The `-W` build fails on a clean checkout with hundreds of errors.

`uv.lock` is committed, the same way `go.sum` is. `--frozen` means the build
uses it rather than re-resolving, and `uv lock --check` in `docs.yml` fails if
the two have drifted. Run `uv lock` deliberately after changing
`pyproject.toml`.

The output lands in `public/`, which is not tracked.

### Files

Target layout, as pages land. `INDEX.md` is the only page so far:

```
pyproject.toml           Sphinx and MyST, the only Python dependencies
uv.lock                  committed, so a build resolves the same versions
docs/
├── conf.py             Sphinx build config
├── INDEX.md            entry point; holds the toctree
├── SETUP.md
├── ARCHITECTURE.md
└── OPERATIONS/         one directory per operational area
    └── DEPLOYMENT.md
```

- One topic per file. Split a file before it grows past a few screens.
- **Upper case file names, `.md`.** The whole base name is upper case with
  hyphens between words: `DEPLOYMENT.md`, not `Deployment.md`. Directory names
  follow the same rule, and the extension stays lower case, matching
  `DOCUMENTATION.md` and `README.md`.
- Names are the document names, and document names are case sensitive: a toctree
  entry or `{doc}` role must match the file's case exactly. Sphinx defaults
  `root_doc` to `index`, so a build config must set `root_doc = "INDEX"` to
  match. `master_doc` is the alias of the same setting; `root_doc` is the name
  to write.
- Anything in `docs/` belongs in the toctree in `INDEX.md`, or it will be
  missed.

### Markup

Section levels come from the number of leading hashes, the same as anywhere else
in the repository:

````markdown
    # Project Title

    ## Setup

    ### Configuration
````

Roles and directives use MyST's curly-bracket form. **Prefer the colon fence**
for a directive, `:::{name}` opened and `:::` closed:

````markdown
:::{code-block} go
func main() {
    community.Register()
}
:::

:::{note}
The bot retries its database connection in the background, so it starts and
stays up even when Postgres is not yet reachable.
:::
````

The backtick fence works too, and is what MyST documents by default:

````markdown
```{note}
Anything at all.
```
````

Use a colon fence when the directive's body contains a fenced block. A nested
backtick fence uses the same delimiter as its parent, which is ambiguous where a
colon fence is not, so this needs no escaping:

````markdown
:::{note}
Here is the call site:

```go
community.Register()
```
:::
````

Either form may be shown as a literal inside a `markdown` fence, because MyST
reads `:::` as an ordinary literal there rather than as a directive. That is
what makes this section quotable.

| Meaning | Syntax |
|---|---|
| Literal, and a command or symbol | `` `go mod tidy` `` |
| Strong, emphasis | `**bold**`, `*italic*` |
| Admonition | `:::{note}`, `:::{warning}`, `:::{danger}` |
| Directive with a fenced body | `:::{code-block} go` ... `:::` |
| Section in this tree | {ref}`SETUP` |
| Another document | {doc}`/ARCHITECTURE` |
| Heading anchor | links to `#a-heading` work in the built site |

Prefer a doc reference to a raw URL: a link written once and named survives the
page it points at moving.

Three things are MyST extensions rather than CommonMark, so they need saying:

- **Definition lists are opt-in.** `INDEX.md` uses them, which requires `deflist`
  in `myst_enable_extensions` in `conf.py`. Without it the list renders as the
  literal text `SETUP: description`, the build still succeeds, and `-W` does not
  catch it.
- **Colon fences are opt-in.** Directives are written as `::: {name}`, which
  requires `colon_fence` in the same setting. `conf.py` explains why the repo
  prefers the colon form.
- **A directive fence has to be closed**, even when the body is markdown. An
  unclosed fence swallows the rest of the file.

### Writing

- Wrap prose at a sensible width and keep it. Indentation is significant, so
  never mix tabs and spaces.
- Say what the reader should do next.
- Use commands that can be pasted as-is, with output taken from a real run.
- Put the failure mode next to the instruction it belongs to.

## Before requesting review

- [ ] The checks in [`.github/workflows/ci.yml`](.github/workflows/ci.yml) pass.
      The hooks in [`.pre-commit-config.yaml`](.pre-commit-config.yaml) cover
      everything `ci.yml` runs, plus more, so running them locally catches that
      failure before the push does. There is no hook for the docs build below.
- [ ] `uv run sphinx-build -W -b html docs public` is clean, which is what
      [`.github/workflows/docs.yml`](.github/workflows/docs.yml) runs.
- [ ] New pages are in the `docs/INDEX.md` toctree, with matching case.
- [ ] Commands and file paths are real and copy-pasteable.

## References

- [Go Doc Comments](https://go.dev/doc/comment), the normative specification for
  the doc comment rules above.
- [Effective Go: Comment](https://go.dev/doc/effective_go#comment).
- [MyST syntax](https://myst-parser.readthedocs.io/en/latest/syntax/syntax.html),
    the reference for the roles and directives in the tree.
- [Sphinx Markdown support](https://www.sphinx-doc.org/en/master/usage/markdown.html),
    how MyST is wired into the build.
- [GitHub Pages from a GitHub Actions workflow](https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site#publishing-with-a-custom-github-actions-workflow),
    the route a future `docs.yml` deploy job would take. Not used yet.
