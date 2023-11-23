# mLog -- set of advanced handlers to improve the native GOlang slog functionality

[![Go Reference](https://pkg.go.dev/badge/github.com/xenolog/mlog/.svg)](https://pkg.go.dev/github.com/xenolog/mlog/)

mLog provides a following handlers:

* **MultipleHandler** -- allows to write one log event to multiple destinations.
* **HumanReadableHandler** -- Alternative structured log representation where timestamp, level and message show as plain text line, but additional attributes show in the JSON block

---

**MultipleHandler**  example:

this code log events to the different destinations:

* debug level to a file
* info and above in human readable format to the stdout

```go
func main() {
  debugLogFile := "/tmp/debug.log"

  // create log file to write debug log stream
  debugWriter, err := os.OpenFile(debugLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
  if err != nil {
    log.Fatal(err)
  }

  // create handlers to write events info
  debugHandler := slog.NewJSONHandler(debugWriter, &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug})
  stdHandler := mlog.NewHumanReadableHandler(os.Stdout, &mlog.HumanReadableHandlerOptions{Level: slog.LevelInfo})

  // combine log handlers and initialize logger
  logHandler := mlog.NewMultipleHandler([]slog.Handler{debugHandler, stdHandler}, nil)
  logger := slog.New(logHandler)
  slog.SetDefault(logger) // redirect all goland logs streams (ancient log and slog) to a freshly created logger

  logger.Debug("first message", "now", time.Now())
  logger.Error("second message", "now", time.Now())
  log.Print("third message, using log.Print(...)")
  slog.Debug("fourth message", "now", time.Now())
}
```

After run this code we will see a following results:

`stdout:`

```log
2023-11-23T15:30:09.224406Z E --  second message  ATTRS={"now":"2023-11-23T18:30:09.224402+03:00"}
2023-11-23T15:30:09.224551Z I --  third message, using log.Print(...)
```

`cat /tmp/debug.log`

```log
{"time":"2023-11-23T18:30:09.223927+03:00","level":"DEBUG","source":{"function":"main.main","file":"/src/mlog/examples/multiple_destinations.go","line":34},"msg":"first message","now":"2023-11-23T18:30:09.223908+03:00"}
{"time":"2023-11-23T18:30:09.224406+03:00","level":"ERROR","source":{"function":"main.main","file":"/src/mlog/examples/multiple_destinations.go","line":35},"msg":"second message","now":"2023-11-23T18:30:09.224402+03:00"}
{"time":"2023-11-23T18:30:09.224551+03:00","level":"INFO","msg":"third message, using log.Print(...)"}
{"time":"2023-11-23T18:30:09.224589+03:00","level":"DEBUG","source":{"function":"main.main","file":"/src/mlog/examples/multiple_destinations.go","line":37},"msg":"fourth message","now":"2023-11-23T18:30:09.224586+03:00"}
```
