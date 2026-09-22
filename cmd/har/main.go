package main

import (
	"fmt"
	"os"
	"strconv"

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
  har (z|zim) [--max-depth=<n>] [--any-max=<mb>] [--not-media-max=<mb>] [--picture-max=<mb>] [--document-max=<mb>] [--music-max=<mb>] [--video-max=<mb>] [--include-zip] [--include-exe] [--include-any] [--no-overreach-media] [--overreach-media] [--overreach-any] [--turbo] [--skip-download] [--working-dir=<path>] [--timestamp] [--creator=<str>] [--publisher=<str>] [--description=<str>] [--long-description=<str>] [--language=<code>] [-s] URL [-O FILE]
  har -h | --help
  har --version

Arguments:
  URL             Web address of archive, script, git repo (.git), or site to pack as ZIM
  DIR             Directory to turn into self-extracting installer

Options:
  -h --help                 Show this screen
  --version                 Show version
  [shared]
  -O FILE                   Output filename
  -s, --silent              Do not show download progress
  -y                        Assume yes, non-interactive mode
  --sha1=<sum>              Verify sha1sum of downloaded content
  [install]
  --ruby                    Run script with ruby
  --python                  Run script with python
  --python3                 Run script with python3
  --zsh                     Run script with zsh
  --bash                    Run script with bash (default)
  [extract]
  -C DIR                    Directory to extract contents of archive
  [zim]
  --turbo                   Disable crawl delay
  --max-depth=<n>           HTML link depth; 0 is this page only [default: 1]
  --any-max=<mb>            Delete any file larger than this (MB) [default: 128]
  --not-media-max=<mb>      Max size for non-media files (MB) [default: 2]
  --picture-max=<mb>        Max size for pictures (MB)
  --document-max=<mb>       Max size for documents (MB)
  --music-max=<mb>          Max size for music (MB)
  --video-max=<mb>          Max size for video (MB)
  --include-zip             Include archive files
  --include-exe             Include program files
  --include-any             Download any file type
  --no-overreach-media      Skip off-site media except images on the page (default)
  --overreach-media         Also fetch off-site video and other media
  --overreach-any           Fetch any off-site href/src
  --skip-download           Pack existing --working-dir without crawling
  --working-dir=<path>      Keep crawled files in this directory
  --timestamp               Append timestamp to zim filename
  --creator=<str>           ZIM creator metadata
  --publisher=<str>         ZIM publisher metadata
  --description=<str>       ZIM description metadata
  --long-description=<str>  ZIM long description metadata
  --language=<code>         ISO 639-3 language code [default: eng]
`

func main() {

	// Parse arguments
	arguments, err := docopt.ParseArgs(usage, nil, "v1.4.0")
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
		arguments["c"].(bool) || arguments["create"].(bool) ||
		arguments["z"].(bool) || arguments["zim"].(bool)

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

	case arguments["z"] == true || arguments["zim"] == true:
		opts, parseErr := zimOptionsFromArgs(arguments, url, outputFile)
		if parseErr != nil {
			err = parseErr
			break
		}
		err = har.HandleZim(download, opts)
	}

	// Check if there was an error
	if err != nil {
		har.Fatal(err)
		os.Exit(1)
	}
}

func zimOptionsFromArgs(arguments map[string]interface{}, url, outputFile string) (har.ZimOptions, error) {
	maxDepth, err := optInt(arguments, "--max-depth", 1)
	if err != nil {
		return har.ZimOptions{}, err
	}
	anyMax, err := optInt(arguments, "--any-max", 128)
	if err != nil {
		return har.ZimOptions{}, err
	}
	notMediaMax, err := optInt(arguments, "--not-media-max", 2)
	if err != nil {
		return har.ZimOptions{}, err
	}
	pictureMax, err := optInt(arguments, "--picture-max", 0)
	if err != nil {
		return har.ZimOptions{}, err
	}
	documentMax, err := optInt(arguments, "--document-max", 0)
	if err != nil {
		return har.ZimOptions{}, err
	}
	musicMax, err := optInt(arguments, "--music-max", 0)
	if err != nil {
		return har.ZimOptions{}, err
	}
	videoMax, err := optInt(arguments, "--video-max", 0)
	if err != nil {
		return har.ZimOptions{}, err
	}

	return har.ZimOptions{
		URL:              url,
		OutputFile:       outputFile,
		MaxDepth:         maxDepth,
		AnyMaxMB:         anyMax,
		NotMediaMaxMB:    notMediaMax,
		PictureMaxMB:     pictureMax,
		DocumentMaxMB:    documentMax,
		MusicMaxMB:       musicMax,
		VideoMaxMB:       videoMax,
		IncludeZip:       arguments["--include-zip"] == true,
		IncludeExe:       arguments["--include-exe"] == true,
		IncludeAny:       arguments["--include-any"] == true,
		NoOverreachMedia: arguments["--no-overreach-media"] == true,
		OverreachMedia:   arguments["--overreach-media"] == true,
		OverreachAny:     arguments["--overreach-any"] == true,
		Turbo:            arguments["--turbo"] == true,
		SkipDownload:     arguments["--skip-download"] == true,
		WorkingDir:       optStr(arguments, "--working-dir"),
		Timestamp:        arguments["--timestamp"] == true,
		Creator:          optStr(arguments, "--creator"),
		Publisher:        optStr(arguments, "--publisher"),
		Description:      optStr(arguments, "--description"),
		LongDescription:  optStr(arguments, "--long-description"),
		Language:         optStr(arguments, "--language"),
	}, nil
}

func optStr(arguments map[string]interface{}, key string) string {
	v, _ := arguments[key].(string)
	return v
}

func optInt(arguments map[string]interface{}, key string, def int) (int, error) {
	v, _ := arguments[key].(string)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s %q is not an integer", key, v)
	}
	return n, nil
}
