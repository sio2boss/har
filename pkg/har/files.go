package har

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

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

func ExtractDownloadedFile(filename string, outputPath string, show bool) {
	extract_command, extract_args := getSystemCommandFromFilename(filename)
	if extract_command == "" {
		return
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		GetLogger().WithError(err).Info("Error creating output directory")
		return
	}

	var cmd *exec.Cmd
	if outputPath != "." {
		outputFlags := getOutputFlags(filename, outputPath)
		cmd = exec.Command(extract_command, extract_args, filename, outputFlags)
	} else {
		cmd = exec.Command(extract_command, extract_args, filename)
	}

	if show {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err := cmd.Run()
	if err != nil {
		return
	}
}

func RemoveDownloadedFile(filename string) {
	// TODO: We need to prevent full disk deletes here
	cmd := exec.Command("rm", "-rf", filename)
	cmd.Run()
}

func RemoveIfExists(destination string) error {
	if _, err := os.Stat(destination); os.IsExist(err) {
		return os.Remove(destination)
	}
	return nil
}

func getFilenameFromUrl(url string) string {
	tokens := strings.Split(url, "/")
	return tokens[len(tokens)-1]
}

func ConfirmExecution() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("About to run script that was just downloaded from the Internet, continue? [Y/n]: ")
	text, _ := reader.ReadString('\n')
	return text != "n" && text != "N"
}

func Verify(input io.ReadCloser, sha string) (bool, error) {
	hash := sha1.New()
	if _, err := io.Copy(hash, input); err != nil {
		return false, err
	}
	hashInBytes := hash.Sum(nil)
	hashsum := hex.EncodeToString(hashInBytes)
	return hashsum == sha, nil
}
