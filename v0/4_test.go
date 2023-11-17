package mlog_test

import (
	"fmt"
	"testing"

	"github.com/itchyny/gojq"
)

// -----------------------------------------------------------------------------

func Jq(t *testing.T, data any, query string) bool {
	v, err := JqGet(data, query)
	if err != nil {
		t.Fatal(err)
	}

	res, ok := v.(bool)
	if !ok {
		t.Fatal(fmt.Errorf("JQ query result is not boolean"))
	}
	return res
}

func JqGet(data any, query string) (any, error) {
	jq, err := gojq.Parse(query)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	iter := jq.Run(data)
	v, ok := iter.Next()
	if !ok {
		return nil, fmt.Errorf("JQ query path not found")
	}
	if err, ok := v.(error); ok {
		return nil, err
	}
	return v, nil
}

func JqGetString(data any, query string) string { // note, no error returned to simplify test writing
	v, err := JqGet(data, query)
	if err != nil {
		return ""
	}
	str, ok := v.(string)
	if !ok {
		return ""
	}
	return str
}
