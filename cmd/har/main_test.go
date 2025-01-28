package main

import (
	"testing"

	"github.com/docopt/docopt-go"
	"github.com/sio2boss/har/pkg/har"
	"github.com/stretchr/testify/assert"
)

func TestArgumentParsing(t *testing.T) {

	args := []string{"g", "-s", "http://example.com/file.zip"}
	arguments, _ := docopt.ParseArgs(usage, args, "v1.2.2")

	url, _ := arguments["URL"].(string)
	silent := arguments["--silent"].(bool)

	assert.Equal(t, "http://example.com/file.zip", url)
	assert.Equal(t, true, silent)
}

func TestLoggerConfiguration(t *testing.T) {
	logger := har.GetLogger()
	logger.SetSilent(true)
	assert.Equal(t, true, logger.IsSilent())

	logger.SetSilent(false)
	assert.Equal(t, false, logger.IsSilent())
}
