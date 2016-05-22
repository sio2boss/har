package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/docopt/docopt-go"
	"gopkg.in/cheggaaa/pb.v1"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

var log = logrus.New()

func init() {
	log.Formatter = new(logrus.TextFormatter)
	log.Level = logrus.DebugLevel
}

func downloadFromUrl(output_path string, url string) string {

	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]

	log.Info("Downloading", url)

	// Check file existence first
	if _, err := os.Stat(fileName); os.IsExist(err) {
		log.Info("File you requested to download already exists")
		return ""
	}

	output, err := os.Create(fileName)
	if err != nil {
		log.Info("Error while creating", fileName, "-", err)
		return ""
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		log.Info("Error while downloading", url, "-", err)
		return ""
	}
	defer response.Body.Close()

	bar := pb.New(int(response.ContentLength))
	bar.SetUnits(pb.U_BYTES)
	bar.ShowSpeed = true
	bar.SetMaxWidth(100)
	bar.Start()
	reader := bar.NewProxyReader(response.Body)
	n, err := io.Copy(output, reader)
	if err != nil {
		bar.Finish()
		log.Info("Error while downloading", url, "-", err)
		return ""
	}
	bar.Finish()

	log.Info(n, "bytes downloaded.")

	return fileName
}

func getSystemCommandFromFilename(filename string) (string, string) {

	parts := strings.Split(filename, ".")

	switch parts[len(parts)-1] {
	case "zip":
		return "unzip", ""
	case "tgz":
		return "tar", "xvfz"
	case "gz":
		if parts[len(parts)-2] != "tar" {
			return "gunzip", ""
		}
		return "tar", "xvfz"
	case "bz2":
		return "tar", "xvfj"
	case "tar":
		return "tar", "xvf"
	default:
		return "", ""
	}
}

func extractDownloadedFile(filename string) {

	extract_command, extract_args := getSystemCommandFromFilename(filename)

	if extract_command == "" {
		return
	}

	cmd := exec.Command(extract_command, extract_args, filename)
	err := cmd.Run()
	if err != nil {
		log.Info("Unable to extract file due to error: %s\n", err)
		return
	}
}

func removeDownloadedFile(filename string) {

	cmd := exec.Command("rm", "-f", filename)
	err := cmd.Run()
	if err != nil {
		log.Info("Unable to remove file due to error: %s\n", err)
	}
}

func main() {

	usage := `Har, from the Swedish verb 'to have'.  Download the url and
unpack it if necessary.

Usage:
  har [--output=<dir>] <url>
  har -h | --help
  har --version

Options:
  -h --help       Show this screen.
  --version       Show version.
  --output=<dir>  Output directory. [default: . ].

`
	arguments, _ := docopt.Parse(usage, nil, true, "Har 1.0", false)

	url, _ := arguments["<url>"].(string)
	output_path, _ := arguments["--output"].(string)

	filename := downloadFromUrl(output_path, url)

	if filename != "" {
		extractDownloadedFile(filename)
		removeDownloadedFile(filename)
	}

}
