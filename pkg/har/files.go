package har

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Bios-Marcel/wastebasket/v2"
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

// moveToTrash is wastebasket.Trash, replaced in tests so they never touch the real Trash.
var moveToTrash = wastebasket.Trash

// removeOwnedTempDir deletes dir only when it is still a har temp directory
// created under os.TempDir.
func removeOwnedTempDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("refusing to remove an empty temp directory")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)

	tempRoot, err := filepath.Abs(os.TempDir())
	if err != nil {
		return err
	}
	tempRoot = filepath.Clean(tempRoot)
	rel, err := filepath.Rel(tempRoot, abs)
	if err != nil {
		return err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("refusing to remove temp directory outside %s: %s", tempRoot, abs)
	}
	if !strings.HasPrefix(filepath.Base(abs), "har") {
		return fmt.Errorf("refusing to remove temp directory %s", abs)
	}
	return os.RemoveAll(abs)
}

// trashPath moves path to the OS trash when it is exactly one non-sensitive location.
func trashPath(path string) error {
	if path == "" || path == "." {
		return fmt.Errorf("refusing to trash %q", path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)
	if err := refuseSensitivePath(abs); err != nil {
		return err
	}

	info, err := os.Lstat(abs)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to trash symlink %s", abs)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return err
	}
	if err := refuseSensitivePath(filepath.Clean(resolved)); err != nil {
		return err
	}
	return moveToTrash(abs)
}

func refuseSensitivePath(path string) error {
	cleaned := filepath.Clean(path)
	if cleaned == "" || cleaned == "." || isFilesystemRoot(cleaned) {
		return fmt.Errorf("refusing to trash filesystem root %s", cleaned)
	}
	if isFilesystemRoot(filepath.Dir(cleaned)) {
		return fmt.Errorf("refusing to trash top-level path %s", cleaned)
	}
	if home, err := os.UserHomeDir(); err == nil && cleaned == filepath.Clean(home) {
		return fmt.Errorf("refusing to trash home directory %s", cleaned)
	}
	if cwd, err := os.Getwd(); err == nil && cleaned == filepath.Clean(cwd) {
		return fmt.Errorf("refusing to trash current directory %s", cleaned)
	}
	return nil
}

func isFilesystemRoot(path string) bool {
	cleaned := filepath.Clean(path)
	if cleaned == string(filepath.Separator) {
		return true
	}
	volume := filepath.VolumeName(cleaned)
	return volume != "" && (cleaned == volume || cleaned == volume+string(filepath.Separator))
}

func RemoveIfExists(destination string) error {
	if _, err := os.Stat(destination); os.IsExist(err) {
		return os.Remove(destination)
	}
	return nil
}

func getFilenameFromUrl(url string) string {
	url = strings.TrimRight(url, "/")
	tokens := strings.Split(url, "/")
	return tokens[len(tokens)-1]
}

func isGitRepoUrl(url string) bool {
	return strings.HasSuffix(strings.TrimRight(url, "/"), ".git")
}

func getRepoNameFromUrl(url string) string {
	return strings.TrimSuffix(getFilenameFromUrl(url), ".git")
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
