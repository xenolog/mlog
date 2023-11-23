// in this example we log events to the different destinations:
// debug level to a file
// info and above -- in human readable format to the stdout

package main

import (
	"log"
	"log/slog"
	"os"
	"time"

	mlog "github.com/xenolog/mlog/v0"
)

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
