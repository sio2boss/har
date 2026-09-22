package main

import (
	"testing"

	"github.com/docopt/docopt-go"
	"github.com/sio2boss/har/pkg/har"
	"github.com/stretchr/testify/assert"
)

func TestArgumentParsing(t *testing.T) {

	args := []string{"g", "-s", "http://example.com/file.zip"}
	arguments, _ := docopt.ParseArgs(usage, args, "v1.4.0")

	url, _ := arguments["URL"].(string)
	silent := arguments["--silent"].(bool)

	assert.Equal(t, "http://example.com/file.zip", url)
	assert.Equal(t, true, silent)
}

func TestZimArgumentParsing(t *testing.T) {
	args := []string{"zim", "--max-depth=1", "--turbo", "http://example.com/", "-O", "google.zim"}
	arguments, err := docopt.ParseArgs(usage, args, "v0.0.0")
	assert.NoError(t, err)
	assert.Equal(t, true, arguments["zim"])
	url, _ := arguments["URL"].(string)
	assert.Equal(t, "http://example.com/", url)
	assert.Equal(t, "1", arguments["--max-depth"])
	assert.Equal(t, true, arguments["--turbo"])
	assert.Equal(t, "google.zim", arguments["-O"])

	defaults, err := docopt.ParseArgs(usage, []string{"zim", "http://example.com/"}, "v1.3.2")
	assert.NoError(t, err)
	assert.Equal(t, "1", defaults["--max-depth"])
	assert.Equal(t, "eng", defaults["--language"])
}

func TestLoggerConfiguration(t *testing.T) {
	logger := har.GetLogger()
	logger.SetSilent(true)
	assert.Equal(t, true, logger.IsSilent())

	logger.SetSilent(false)
	assert.Equal(t, false, logger.IsSilent())
}
