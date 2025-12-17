package har

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDownload(t *testing.T) {
	// Test creating a new download
	d, err := NewDownload("http://example.com/test.zip", "output.zip", true, nil, false)
	assert.NoError(t, err)
	assert.NotNil(t, d)
	assert.NotEmpty(t, d.TempDir)

	// Verify options are set correctly
	assert.Equal(t, "http://example.com/test.zip", d.Options.URL)
	assert.Equal(t, "output.zip", d.Options.OutputFile)
	assert.True(t, d.Options.ShowProgress)
	assert.Nil(t, d.Options.SHA1Sum)
	assert.False(t, d.Options.Force)
}

func TestGetDestinationPath(t *testing.T) {
	d, err := NewDownload("http://example.com/test.zip", "", true, nil, false)
	assert.NoError(t, err)

	// Test with temp directory
	tempPath := d.GetDestinationPath(true)
	assert.Contains(t, tempPath, d.TempDir)
	assert.Contains(t, tempPath, "test.zip")

	// Test with output file specified
	d.Options.OutputFile = "custom.zip"
	customPath := d.GetDestinationPath()
	assert.Equal(t, "custom.zip", customPath)

	// Test with no output file (should use CWD)
	d.Options.OutputFile = ""
	cwd, _ := os.Getwd()
	cwdPath := d.GetDestinationPath()
	assert.Equal(t, filepath.Join(cwd, "test.zip"), cwdPath)
}

func TestDownloadWithSHA1(t *testing.T) {
	// Create a test server with known content
	testContent := "test file content"
	correctSHA1 := "9032bbc224ed8b39183cb93b9a7447727ce67f9d" // SHA1 of "test file content"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// Test with correct SHA1
	d1, _ := NewDownload(server.URL, "", false, correctSHA1, false)
	n1 := d1.downloadFromUrl(true)
	assert.Equal(t, int64(len(testContent)), n1)

	// Test with incorrect SHA1
	d2, _ := NewDownload(server.URL, "", false, "incorrectsha1", false)
	n2 := d2.downloadFromUrl(true)
	assert.Equal(t, int64(0), n2) // Should fail and return 0
}
