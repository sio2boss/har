package har

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func HandleGet(d *Download) error {
	logger := GetLogger()
	if d.Options.Force {
		RemoveIfExists(d.Options.OutputFile)
	}

	if d.downloadFromUrl() < 1 {
		logger.Error("download failed")
		return nil
	}
	return nil
}

func HandleExtract(d *Download, outputPath string) error {
	logger := GetLogger()
	if d.downloadFromUrl(true) < 1 {
		logger.Error("download failed")
		return nil
	}

	if outputPath == "" {
		outputPath = "."
	}

	ExtractDownloadedFile(d.GetDestinationPath(true), outputPath, d.Options.ShowProgress)
	return nil
}

func HandleBinary(d *Download) error {
	logger := GetLogger()
	if d.downloadFromUrl(true) < 1 {
		logger.Error("download failed")
		return nil
	}

	destPath := d.GetDestinationPath(true)
	if err := os.Chmod(destPath, 0776); err != nil {
		logger.WithError(err).Error("failed to set file permissions")
		return nil
	}

	destination := d.Options.OutputFile
	if destination == "" {
		destination = "~/.local/bin/" + getFilenameFromUrl(d.Options.URL)
	}

	if d.Options.Force {
		RemoveIfExists(destination)
	}

	if err := os.Rename(destPath, destination); err != nil {
		logger.WithError(err).Error("failed to move file to destination")
		return nil
	}
	return nil
}

func isArchive(filename string) bool {
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return false
	}

	lastExt := parts[len(parts)-1]
	secondLastExt := ""
	if len(parts) >= 2 {
		secondLastExt = parts[len(parts)-2]
	}

	switch lastExt {
	case "zip", "tgz", "tar", "7z", "rar":
		return true
	case "gz":
		return secondLastExt == "tar"
	case "bz2":
		return secondLastExt == "tar"
	case "xz":
		return secondLastExt == "tar"
	case "lzma":
		return secondLastExt == "tar"
	case "zst":
		return secondLastExt == "tar"
	default:
		return false
	}
}

func HandleInstall(d *Download, shell string, assumeYes bool) error {
	logger := GetLogger()
	url := d.Options.URL

	// Check if URL is an archive
	filename := getFilenameFromUrl(url)
	isArchiveFile := isArchive(filename)

	var destPath string
	if len(url) > 4 && url[0:4] == "http" {
		// For archives, download to current directory instead of temp
		if isArchiveFile {
			if d.downloadFromUrl(false) < 1 {
				logger.Error("download failed")
				return nil
			}
			destPath = d.GetDestinationPath(false)
		} else {
			if d.downloadFromUrl(true) < 1 {
				logger.Error("download failed")
				return nil
			}
			destPath = d.GetDestinationPath(true)
		}
	} else {
		destPath = url
		// Check if local file is an archive
		isArchiveFile = isArchive(destPath)
	}

	// If it's an archive, extract and look for install.sh/setup.sh
	if isArchiveFile {
		outputPath := "."

		// Extract the archive
		ExtractDownloadedFile(destPath, outputPath, d.Options.ShowProgress)

		// Find first directory that doesn't start with "."
		var installDir string
		entries, err := os.ReadDir(outputPath)
		if err != nil {
			logger.WithError(err).Error("failed to read extraction directory")
			return nil
		}

		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				installDir = filepath.Join(outputPath, entry.Name())
				break
			}
		}

		if installDir == "" {
			logger.Error("no suitable directory found (directories starting with '.' are ignored)")
			return nil
		}

		// Check for install.sh or setup.sh
		var installScript string
		installShPath := filepath.Join(installDir, "install.sh")
		setupShPath := filepath.Join(installDir, "setup.sh")

		if info, err := os.Stat(installShPath); err == nil && !info.IsDir() {
			installScript = installShPath
		} else if info, err := os.Stat(setupShPath); err == nil && !info.IsDir() {
			installScript = setupShPath
		} else {
			logger.Error("no install.sh or setup.sh found in " + installDir)
			return nil
		}

		// Make script executable
		if err := os.Chmod(installScript, 0755); err != nil {
			logger.WithError(err).Error("failed to make script executable")
			return nil
		}

		if !assumeYes {
			if !ConfirmExecution() {
				return nil
			}
		}

		// Get just the script filename since we'll execute from the install directory
		scriptName := filepath.Base(installScript)

		if !d.Options.ShowProgress {
			logger.Info("Running: './" + scriptName + "' in " + installDir)
		}

		// Execute the script from the install directory - the OS will handle the shebang
		cmd := exec.Command("./" + scriptName)
		cmd.Dir = installDir
		if !d.Options.ShowProgress {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}

		err = cmd.Run()
		if err != nil {
			logger.WithError(err).Error("failed to run install script")
			return nil
		}

		logger.Debug("install script ran successfully")
		return nil
	}

	// Not an archive, run as script
	if !assumeYes {
		if !ConfirmExecution() {
			return nil
		}
	}

	if !d.Options.ShowProgress {
		logger.Info("Running: '" + shell + " " + destPath + "'")
	}

	cmd := exec.Command(shell, destPath)
	if !d.Options.ShowProgress {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err := cmd.Run()
	if err != nil {
		logger.WithError(err).Error("failed to run command")
		return nil
	}

	logger.Debug("command ran successfully")
	return nil
}

var decompress = `
#!/bin/bash
echo ""
echo "Self Extracting..."

export TMPDIR=$(mktemp -d /tmp/selfextract.XXXXXX)

ARCHIVE=$(awk '/^__ARCHIVE_BELOW__/ {print NR + 1; exit 0; }' $0)

tail -n+$ARCHIVE $0 | tar xzv --strip-components=1 -C $TMPDIR

CDIR=$(pwd)
cd $TMPDIR
echo "Installing..."

if [ -f "./installer" ] && [ -x "./installer" ]; then
    ./installer
else
    # Find first directory that doesn't start with "."
    INSTALL_DIR=""
    for dir in */; do
        if [ -d "$dir" ] && [ "${dir#.}" = "$dir" ]; then
            INSTALL_DIR="$dir"
            break
        fi
    done
    
    if [ -n "$INSTALL_DIR" ]; then
        cd "$INSTALL_DIR"
        if [ -f "./install.sh" ] && [ -x "./install.sh" ]; then
            ./install.sh
        elif [ -f "./setup.sh" ] && [ -x "./setup.sh" ]; then
            ./setup.sh
        else
            echo "No installer, install.sh, or setup.sh found"
            exit 1
        fi
    else
        echo "No installer found and no suitable directory found"
        exit 1
    fi
fi

cd $CDIR
rm -rf $TMPDIR

exit 0

__ARCHIVE_BELOW__
`

func HandleCreateArchive(directory string, outputFile string) error {
	logger := GetLogger()

	// Create temporary file for the archive
	tempfile := os.TempDir() + "payload.tgz"
	defer os.Remove(tempfile)

	// Split directory into base path and target directory
	var basePath, targetDir string

	// Trim trailing slashes before processing
	directory = strings.TrimRight(directory, "/")
	lastSlash := strings.LastIndex(directory, "/")
	if lastSlash != -1 {
		basePath = directory[:lastSlash]
		targetDir = directory[lastSlash+1:]
	} else {
		basePath = "."
		targetDir = directory
	}

	// Store current directory
	currentDir, err := os.Getwd()
	if err != nil {
		logger.WithError(err).Error("failed to get current directory")
		return nil
	}
	defer os.Chdir(currentDir) // Ensure we return to original directory

	// Change to base directory
	if err := os.Chdir(basePath); err != nil {
		logger.WithError(err).Error("failed to change to base directory")
		return nil
	}

	// Compress the directory from the new working directory
	cmd := exec.Command("tar", "cvfz", tempfile, targetDir)
	if err := cmd.Run(); err != nil {
		logger.WithError(err).Error("failed to compress directory")
		return nil
	}

	// Read the compressed tar
	content, err := os.ReadFile(tempfile)
	if err != nil {
		logger.WithError(err).Error("failed to read compressed archive")
		return nil
	}

	// Set output filename
	filename := targetDir + ".har"
	if outputFile != "" {
		filename = outputFile
	}

	// Create the self-extracting archive
	var outbytes []byte
	outbytes = append(outbytes, []byte(decompress)...)
	outbytes = append(outbytes, content...)

	// Write the file with execute permissions
	if err := os.WriteFile(filename, outbytes, 0744); err != nil {
		logger.WithError(err).Error("failed to write self-extracting archive")
		return nil
	}

	logger.Info("Successfully created self-extracting archive: ", filename)
	return nil
}
