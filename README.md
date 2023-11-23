# mLog -- set of advanced handlers to improve the native GOlang slog functionality

[![Go Reference](https://pkg.go.dev/badge/github.com/xenolog/mlog/.svg)](https://pkg.go.dev/github.com/xenolog/mlog/)

mLog provides a following handlers:

* **MultipleHandler** -- allows to write one log event to multiple destinations.
* **HumanReadableHandler** -- Alternative structured log representation where timestamp, level and message show as plain text line, but additional attributes show in the JSON block
