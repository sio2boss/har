package har

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type DownloadOptions struct {
	URL          string
	OutputFile   string
	ShowProgress bool
	SHA1Sum      interface{}
	Force        bool
}

type Download struct {
	Options DownloadOptions
	TempDir string
}

func NewDownload(url string, outputFile string, showProgress bool, sha interface{}, force bool) (*Download, error) {
	dir, err := os.MkdirTemp("", "har")
	if err != nil {
		return nil, err
	}

	return &Download{
		Options: DownloadOptions{
			URL:          url,
			OutputFile:   outputFile,
			ShowProgress: showProgress,
			SHA1Sum:      sha,
			Force:        force,
		},
		TempDir: dir,
	}, nil
}

func (d *Download) Cleanup() {
	RemoveDownloadedFile(d.TempDir)
}

func (d *Download) GetDestinationPath(useTemp ...bool) string {

	// Default to false if no value provided
	isTemp := false
	if len(useTemp) > 0 {
		isTemp = useTemp[0]
	}

	// If useTemp is true, use the temp directory
	if isTemp {
		// If output file is not specified, use the default temp directory
		return filepath.Join(d.TempDir, getFilenameFromUrl(d.Options.URL))
	}

	// If output file is specified, use it
	if d.Options.OutputFile != "" {
		return d.Options.OutputFile
	}

	// If no output file is specified, download to current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return getFilenameFromUrl(d.Options.URL)
	}
	return filepath.Join(cwd, getFilenameFromUrl(d.Options.URL))
}

func (d *Download) downloadFromUrl(useTemp ...bool) int64 {
	logger := GetLogger()

	// Check file existence first
	fileName := d.GetDestinationPath(useTemp...)
	logger.Info("Downloading to: ", fileName)
	if _, err := os.Stat(fileName); os.IsExist(err) {
		logger.Info("File you requested to download already exists")
		return 0
	}

	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
		logger.WithError(err).Info("Error creating directory structure")
		return 0
	}

	output, err := os.Create(fileName)
	if err != nil {
		logger.Info("Error while creating", fileName, "-", err)
		return 0
	}

	response, err := http.Get(d.Options.URL)
	if err != nil || response == nil {
		if response != nil && response.StatusCode > 400 {
			logger.Fatal("Error while downloading, response code: ", response.StatusCode)
		} else {
			logger.Fatal("Error while downloading, invalid response")
		}
	}
	defer response.Body.Close()

	// Display Progress Bar
	var reader io.ReadCloser
	if d.Options.ShowProgress {
		logger.StartProgress(response.ContentLength)
		reader = io.NopCloser(logger.GetProgressReader(response.Body))
	} else {
		reader = response.Body
	}

	// Copy
	n, err := io.Copy(output, reader)
	if err != nil {
		logger.WithError(err).Info("Error while downloading", d.Options.URL)
	}

	// Close
	if d.Options.ShowProgress {
		logger.StopProgress()
	}
	output.Close()

	// Verify sha1sum
	if _, ok := d.Options.SHA1Sum.(string); ok && n > 0 {
		// Read
		written, err := os.Open(fileName)
		if err != nil {
			logger.WithError(err).Info("Error in reading file downloaded", d.Options.URL)
			written.Close()
			os.Remove(fileName)
			return 0
		}
		v, err := Verify(written, d.Options.SHA1Sum.(string))
		if !v || err != nil {
			logger.WithError(err).Info("Error in sha1sum validation", d.Options.URL)
			written.Close()
			os.Remove(fileName)
			return 0
		}
	}

	if d.Options.ShowProgress && d.Options.SHA1Sum != nil {
		logger.Info("Downloaded file matches: ", d.Options.SHA1Sum)
	}
	return n
}
