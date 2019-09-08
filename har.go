package main

import (
	"bufio"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/docopt/docopt-go"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

var usage = `Har, from the Swedish verb 'to have', downloads the URL and handles repetitive tasks for you.

Usage:
  har i [--sudo] [--ruby|--python|--python3] [-y] URL
  har b [-y] URL
  har g URL [-O FILE]
  har x URL [-C DIR]
  har -h | --help
  har --version

Arguments:
  URL             Web address of archive or script you want to have

Options:
  -h --help       Show this screen
  --version       Show version
  -C DIR          Directory to extract contents of archive
  -O FILE         Output filename
  --ruby          Run script with ruby
  --python        Run script with python
  --python3       Run script with python3
  --sudo          Run with sudo
  -y              Use for interactive mode
`

var log = logrus.New()

func init() {

	log.Out = os.Stdout

}

func getFilenameFromUrl(url string) string {
	tokens := strings.Split(url, "/")
	return tokens[len(tokens)-1]
}

func downloadFromUrl(fileName string, url string) int64 {

	// Check file existence first
	if _, err := os.Stat(fileName); os.IsExist(err) {
		log.Info("File you requested to download already exists")
		return 0
	}

	output, err := os.Create(fileName)
	if err != nil {
		log.Info("Error while creating", fileName, "-", err)
		return 0
	}

	response, err := http.Get(url)
	if err != nil || response == nil {
		if response != nil && response.StatusCode > 400 {
			log.Fatal("Error while downloading, response code: ", response.StatusCode)
		} else {
			log.Fatal("Error while downloading, invalid response")
		}
	}
	defer response.Body.Close()

	bar := pb.New64(response.ContentLength)
	bar.SetWidth(100)
	bar.Start()
	reader := bar.NewProxyReader(response.Body)
	n, err := io.Copy(output, reader)
	if err != nil {
		log.WithError(err).Info("Error while downloading", url)
	}
	bar.Finish()
	output.Close()

	return n
}

func getSystemCommandFromFilename(filename string) (string, string) {

	parts := strings.Split(filename, ".")

	switch parts[len(parts)-1] {
	case "zip":
		return "unzip", ""
	case "tgz":
		return "tar", "-xzf"
	case "gz":
		if parts[len(parts)-2] != "tar" {
			return "gunzip", ""
		}
		return "tar", "xvfz"
	case "bz2":
		return "tar", "-xjf"
	case "tar":
		return "tar", "-xf"
	default:
		return "", ""
	}
}

func getOutputFlags(filename string, outputPath string) string {

	parts := strings.Split(filename, ".")

	out := outputPath

	switch parts[len(parts)-1] {
	case "zip":
		return "-d" + out
	case "tgz":
		return "-C" + out
	case "gz":
		if parts[len(parts)-2] != "tar" {
			return ""
		}
		return "-C" + out
	case "bz2":
		return "-C" + out
	case "tar":
		return "-C" + out
	default:
		return ""
	}
}

func extractDownloadedFile(filename string, outputPath string) {

	// Get extract commands
	extract_command, extract_args := getSystemCommandFromFilename(filename)
	if extract_command == "" {
		return
	}

	var cmd *exec.Cmd
	if outputPath != "." {
		outputFlags := getOutputFlags(filename, outputPath)
		cmd = exec.Command(extract_command, extract_args, filename, outputFlags)
	} else {
		cmd = exec.Command(extract_command, extract_args, filename)
	}

	// Extract
	// TODO add verbose flag
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.WithError(err).Error("Unable to extract file")
		return
	}
}

func removeDownloadedFile(filename string) {

	cmd := exec.Command("rm", "-rf", filename)
	err := cmd.Run()
	if err != nil {
		log.WithError(err).Error("Unable to remove file")
	}
}

func main() {

	arguments, _ := docopt.ParseDoc(usage)

	url, _ := arguments["URL"].(string)

	// Process modes of operation
	if arguments["g"] == true {

		// Download
		filename := getFilenameFromUrl(url)
		if arguments["-O"] != nil {
			filename, _ = arguments["-O"].(string)
		}

		downloadFromUrl(filename, url)

	} else if arguments["x"] == true {

		// Download and extract

		// Create temp directory
		dir, err := ioutil.TempDir("", "har")
		if err != nil {
			log.Fatal(err)
		}

		// Download
		filename := dir + string(os.PathSeparator) + getFilenameFromUrl(url)
		if downloadFromUrl(filename, url) < 1 {
			log.Fatal("Unable to download")
		}

		// Figure out output path
		output_path := "."
		if arguments["-C"] != nil {
			output_path, _ = arguments["-C"].(string)
		}

		// Extract
		if filename != "" {
			extractDownloadedFile(filename, output_path)
		}

		removeDownloadedFile(dir)

	} else if arguments["b"] == true {

		// Download, chmod, and move

		// Create temp directory
		dir, err := ioutil.TempDir("", "har")
		if err != nil {
			log.Fatal(err)
		}

		// Download
		filename := dir + string(os.PathSeparator) + getFilenameFromUrl(url)
		if downloadFromUrl(filename, url) < 1 {
			return
		}

		// Chmod
		err = os.Chmod(filename, 0776)
		if err != nil {
			log.Fatal(err)
		}

		// Force Move?
		destination := "/usr/local/bin/" + getFilenameFromUrl(url)
		if arguments["-y"] == true {
			err = os.Remove(destination)
			if err != nil {
				log.Fatal(err)
			}
		}
		err = os.Rename(filename, destination)
		if err != nil {
			log.Fatal(err)
		}

		removeDownloadedFile(dir)

	} else if arguments["i"] == true {

		// Download, chmod, and move

		// Create temp directory
		dir, err := ioutil.TempDir("", "har")
		if err != nil {
			log.Fatal(err)
		}

		// Download
		filename := dir + string(os.PathSeparator) + getFilenameFromUrl(url)
		if downloadFromUrl(filename, url) < 1 {
			return
		}

		shell := "bash"
		if arguments["--ruby"] == true {
			shell = "ruby"
		} else if arguments["--python"] == true {
			shell = "python"
		} else if arguments["--python3"] == true {
			shell = "python3"
		}

		// Check before running
		reader := bufio.NewReader(os.Stdin)
		if arguments["-y"] == false {
			fmt.Print("About to run script that was just downloaded from the Internet, continue? [Y/n]: ")
			text, _ := reader.ReadString('\n')
			if text == "n" || text == "N" {
				return
			}
		}
		fmt.Println()

		// Run
		if arguments["--sudo"] == true {
			fmt.Println("Running: '"+"sudo", shell, filename+"':")
			cmd := exec.Command("sudo", shell, filename)
			err = cmd.Run()
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err != nil {
				log.Fatal("Unable to run script", err)
			}
		} else {
			fmt.Println("Running: '", shell, filename, "':")
			cmd := exec.Command(shell, filename)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				log.Fatal("Unable to run script", err)
			}
		}

		removeDownloadedFile(dir)
	}

}
