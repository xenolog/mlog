/*
Package mlog provides advanced handlers for slog structured logging,
in which log records include a message,
a severity level, and various other attributes expressed as key-value pairs.

It defines a types [HumanReadableHandler], [MultipleHandler]
which provides additional functionality to the stdlib [log/slog].

# MultipleHandler

[MultipleHandler] allows to write one log event to multiple destinations.
See `examples/multiple_destinations.go` to usage.

# HumanReadableHandler

[HumanReadableHandler] is a alternative structured log representation where
timestamp, level and message wrote as plain text line,
but additional attributes show as JSON block.

A log record consists of a time, a level, a message, and a set of key-value
pairs, where the keys are strings and the values may be of any type.
As an example,

	slog.Info("hello", "count", 3)

creates a record containing the time of the call,
a level of Info, the message "hello", and a single
pair with key "count" and value 3.

The default handler formats the log record's message, time, level, and attributes
as a string and passes it to the [log] package.

	2022/11/08 15:28:26 INFO hello count=3

For more control over the output format, create a logger with a different handler.
This statement uses [slog.New] to create a new logger with a HumanReadableHandler
that writes structured records in text form to standard error:

	logger := slog.New(mlog.HumanReadableHandler(os.Stderr, nil))

[HumanReadableHandler] output is a sequence of timestamp, level and message as plain text
to human readability and additional key=value pairs as JSON, easily and unambiguously
parsed by machine. This statement:

	logger.Info("hello", "count", 3)

produces this output:

	2023-11-23T15:30:09.224406Z I --  hello  ATTRS={"count":3}

Setting a logger as the default with

	slog.SetDefault(logger)

will cause the top-level functions like [slog.Info] to use it.
[slog.SetDefault] also updates the default logger used by the [log] package,
so that existing applications that use [log.Printf] and related functions
will send log records to the logger's handler without needing to be rewritten.
*/
package mlog
