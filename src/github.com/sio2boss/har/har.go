package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"github.com/docopt/docopt-go"
	"gopkg.in/cheggaaa/pb.v1"
)

func downloadFromUrl(output_path string, url string) string {

	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]

	fmt.Println("Downloading", url)

	// Check file existence first
	if _, err := os.Stat(fileName); os.IsExist(err) {
		fmt.Println("File you requested to download already exists")
		return ""
	}

	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return ""
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
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
		fmt.Println("Error while downloading", url, "-", err)
		return ""
	}
	bar.Finish()

	fmt.Println(n, "bytes downloaded.")

	return fileName
}

func getSystemCommandFromFilename(filename string) (string, string) {

	parts := strings.Split(filename, ".")

	switch parts[len(parts)-1] {
	case "zip":
		return "unzip", "";
	case "tgz":
		return "tar", "xvfz";
	case "gz":
		if parts[len(parts)-2] != "tar" {
			return "gunzip", ""
		}
		return "tar", "xvfz";
	case "bz2":
		return "tar", "xvfj";
	case "tar":
		return "tar", "xvf";
	default:
		return "", "";
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
		fmt.Println("Unable to extract file due to error: %s\n", err)
		return
	}
}


func removeDownloadedFile(filename string) {

	cmd := exec.Command("rm", "-f", filename)
	err := cmd.Run()
	if err != nil {
		fmt.Println("Unable to remove file due to error: %s\n", err)
	}
}

func main() {

	usage := `Har.

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

	// countries := []string{"GB", "FR", "ES", "DE", "CN", "CA", "ID", "US"}

	//for i := 0; i < len(countries); i++ {
	//	url := "http://download.geonames.org/export/dump/" + countries[i] + ".zip"
	//	downloadFromUrl(url)
	//}
}
