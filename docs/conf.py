# Configuration file for the Sphinx documentation builder.
#
# Build with: uv run sphinx-build -W -b html docs public
# (uv sync --frozen first; the manifest is at the repository root so the venv
# stays out of this source directory)
#
# The -W is not optional. Without it a page can reference a document that does
# not exist and still build green, which is the failure mode docs/INDEX.md
# describes. .github/workflows/docs.yml passes it on every push and pull
# request. See DOCUMENTATION.md for why the project and master document are
# named in upper case.

project = "pen-bot"
author = "Neon Genesis Linux"

# Sphinx defaults to "index"; the entry point is upper case to match
# DOCUMENTATION.md and README.md. Omitting this fails the build outright.
master_doc = "INDEX"

# The tree is MyST markdown rather than reStructuredText. See DOCUMENTATION.md
# for why, and for the syntax this buys.
extensions = ["myst_parser"]

# "deflist" is opt-in, and dropping it fails silently: a "term:" block would
# render as the literal text "term: definition" with a clean build and no
# warning, which -W does not catch. Removing this line costs the definition
# lists in INDEX.md, so leave it alone.
myst_enable_extensions = ["deflist"]

# Gives every heading a generated anchor, so a deep link into a page works and
# the rendered headings link to themselves. Off by default in MyST.
myst_heading_anchors = 3

# Discord's markup, GitHub's tree renders, and the bot's own embed code all use
# backticks. The default highlight language would highlight prose samples.
highlight_language = "none"

# Turns an unresolved cross reference into a warning. The current tree has no
# references to resolve, so this costs nothing yet and catches the first one
# that would otherwise fail silently once the planned pages land.
nitpicky = True
