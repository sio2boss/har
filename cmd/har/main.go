package main

import (
	"os"

	"github.com/docopt/docopt-go"
	"github.com/sio2boss/har/pkg/har"
)

var usage = `Har, from the Swedish verb 'to have', downloads the URL and handles repetitive tasks for you.

Usage:
  har URL [-O FILE]
  har (g|get)     [-y] [-s] [--sha1=<sum>] URL [-O FILE]
  har (i|install) [--ruby|--python|--python3|--zsh|--bash] [-y] [-s] [--sha1=<sum>] URL
  har (b|binary)  [-y] [-s] [--sha1=<sum>] URL [-O FILE]
  har (x|extract) [-s] [--sha1=<sum>] URL [-C DIR]
  har (c|create)  DIR [-O FILE]
  har -h | --help
  har --version

Arguments:
  URL             Web address of archive or script you want to have
  DIR             Directory to turn into self-extracting installer

Options:
  -h --help       Show this screen
  --version       Show version
  -C DIR          Directory to extract contents of archive
  -O FILE         Output filename
  --ruby          Run script with ruby
  --python        Run script with python
  --python3       Run script with python3
  -y              Assume yes, use for non-interactive mode
  -s, --silent    Do not show download progress
  --sha1=<sum>    Verify sha1sum of downloaded content before proceeding
`

func main() {

	// Parse arguments
	arguments, err := docopt.ParseArgs(usage, nil, "v1.2.2")
	if err != nil {
		har.Fatal(err)
		os.Exit(1)
	}
	url, _ := arguments["URL"].(string)
	silent := arguments["--silent"].(bool)
	sha := arguments["--sha1"]
	force := arguments["-y"] == true
	outputFile, _ := arguments["-O"].(string)

	// Switch logger to silent mode
	har.GetLogger().SetSilent(silent)

	// Create download object
	download, err := har.NewDownload(url, outputFile, silent, sha, force)
	if err != nil {
		har.Fatal(err)
	}
	defer download.Cleanup()

	// Check if any mode is specified
	modeSpecified := arguments["i"].(bool) || arguments["install"].(bool) ||
		arguments["b"].(bool) || arguments["binary"].(bool) ||
		arguments["g"].(bool) || arguments["get"].(bool) ||
		arguments["x"].(bool) || arguments["extract"].(bool) ||
		arguments["c"].(bool) || arguments["create"].(bool)

	// Process the command
	switch {
	case !modeSpecified || arguments["g"] == true || arguments["get"] == true:
		err = har.HandleGet(download)

	case arguments["x"] == true || arguments["extract"] == true:
		extractionDir, _ := arguments["-C"].(string)
		err = har.HandleExtract(download, extractionDir)

	case arguments["b"] == true || arguments["binary"] == true:
		err = har.HandleBinary(download)

	case arguments["i"] == true || arguments["install"] == true:
		shell := ""
		switch {
		case arguments["--ruby"] == true:
			shell = "ruby"
		case arguments["--python"] == true:
			shell = "python"
		case arguments["--python3"] == true:
			shell = "python3"
		case arguments["--zsh"] == true:
			shell = "zsh"
		default:
			shell = "bash"
		}
		err = har.HandleInstall(download, shell, force)

	case arguments["c"] == true || arguments["create"] == true:
		directory, _ := arguments["DIR"].(string)
		err = har.HandleCreateArchive(directory, outputFile)
	}

	// Check if there was an error
	if err != nil {
		har.Fatal(err)
		os.Exit(1)
	}
}
