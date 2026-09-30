# Configuration file for the Sphinx documentation builder.
#
# Build with: uv run sphinx-build -W -b html docs public
# (uv sync --frozen first; the manifest is at the repository root so the venv
# stays out of this source directory)
#
# The -W is not optional. Without it a page can reference a document that does
# not exist and still build green, which is the failure mode docs/INDEX.md
# describes. .github/workflows/docs.yml passes it on every push and pull
# request. See DOCUMENTATION.md for why the project and root document are named
# in upper case.

project = "pen-bot"
author = "Neon Genesis Linux"

# The document holding the root toctree, and so the root of the tree. Sphinx
# defaults to "index"; the entry point is upper case to match DOCUMENTATION.md
# and README.md. Omitting this fails the build outright, because Sphinx looks
# for a lower case index that does not exist.
#
# root_doc is the current name for this. master_doc is its alias: the two set
# each other, and the Sphinx reference documents master_doc as the alias added
# in 4.0. Neither is deprecated, so either works; root_doc is the one to write.
root_doc = "INDEX"

# The tree is MyST markdown rather than reStructuredText. See DOCUMENTATION.md
# for why, and for the syntax this buys.
extensions = ["myst_parser"]

# "deflist" is opt-in, and dropping it fails silently: a "term:" block would
# render as the literal text "term: definition" with a clean build and no
# warning, which -W does not catch. Removing this line costs the definition
# lists in INDEX.md, so leave it alone.
#
# "colon_fence" adds :::{directive} alongside ```{directive}. Per MyST-Parser,
# colons render correctly in any standard Markdown editor, and the form is
# recommended for admonition-type directives.
myst_enable_extensions = ["deflist", "colon_fence"]

# Gives every heading a generated anchor, so a deep link into a page works and
# the rendered headings link to themselves. Off by default in MyST.
myst_heading_anchors = 3

# Discord's markup, GitHub's tree renders, and the bot's own embed code all use
# backticks. The default highlight language would highlight prose samples.
highlight_language = "none"

# Turns an unresolved cross reference into a warning, which the -W build then
# fails on. INDEX.md resolves {ref}`genindex` and {ref}`search`, and a reference
# to a document that does not exist yet is exactly the error worth catching
# before it merges rather than after.
nitpicky = True

# source_encoding is deliberately unset. Sphinx deprecated it in 9.0 and will
# only support UTF-8 in 11, so every file in docs/ is plain UTF-8 by default.
