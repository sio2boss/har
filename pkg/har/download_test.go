package har

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
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

func TestIsGitRepoUrl(t *testing.T) {
	assert.True(t, isGitRepoUrl("https://github.com/foo/bar.git"))
	assert.True(t, isGitRepoUrl("https://github.com/foo/bar.git/"))
	assert.True(t, isGitRepoUrl("git@github.com:foo/bar.git"))
	assert.False(t, isGitRepoUrl("https://github.com/foo/bar"))
	assert.False(t, isGitRepoUrl("https://example.com/file.zip"))
	assert.False(t, isGitRepoUrl("https://example.com/file.git.bak"))
}

func TestGetRepoNameFromUrl(t *testing.T) {
	assert.Equal(t, "bar", getRepoNameFromUrl("https://github.com/foo/bar.git"))
	assert.Equal(t, "bar", getRepoNameFromUrl("https://github.com/foo/bar.git/"))
	assert.Equal(t, "av-shell", getRepoNameFromUrl("https://github.com/sio2boss/av-shell.git"))
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

	// Git URLs should strip .git for the destination directory name
	gd, err := NewDownload("https://github.com/foo/bar.git", "", true, nil, false)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(cwd, "bar"), gd.GetDestinationPath())
	assert.Equal(t, filepath.Join(gd.TempDir, "bar"), gd.GetDestinationPath(true))
}

func TestCloneFromUrl(t *testing.T) {
	// Create a local git repo to clone from
	srcDir := t.TempDir()
	srcRepo := filepath.Join(srcDir, "sample")
	assert.NoError(t, os.MkdirAll(srcRepo, 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(srcRepo, "README"), []byte("hi"), 0644))

	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	run(srcRepo, "git", "init")
	run(srcRepo, "git", "config", "user.email", "test@example.com")
	run(srcRepo, "git", "config", "user.name", "test")
	run(srcRepo, "git", "add", ".")
	run(srcRepo, "git", "commit", "-m", "init")

	destDir := t.TempDir()
	d, err := NewDownload(srcRepo, filepath.Join(destDir, "cloned"), true, nil, false)
	assert.NoError(t, err)

	// Force .git URL detection by using a path that ends in .git
	bare := filepath.Join(srcDir, "sample.git")
	run(srcDir, "git", "clone", "--bare", srcRepo, bare)

	d.Options.URL = bare
	n := d.cloneFromUrl(false)
	assert.Equal(t, int64(1), n)

	_, err = os.Stat(filepath.Join(destDir, "cloned", "README"))
	assert.NoError(t, err)
}

func TestCleanupRemovesOnlyTempDir(t *testing.T) {
	d, err := NewDownload("http://example.com/a.zip", "", false, nil, false)
	assert.NoError(t, err)

	temp := d.TempDir
	assert.NoError(t, os.WriteFile(filepath.Join(temp, "file.txt"), []byte("x"), 0644))

	sibling, err := os.MkdirTemp(filepath.Dir(temp), "sibling")
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(sibling) })

	d.Cleanup()

	_, err = os.Stat(temp)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(sibling)
	assert.NoError(t, err)
}

func TestTrashPathRefusesSensitiveLocations(t *testing.T) {
	orig := moveToTrash
	moveToTrash = func(paths ...string) error {
		t.Fatalf("trash called for %v", paths)
		return nil
	}
	t.Cleanup(func() { moveToTrash = orig })

	assert.Error(t, trashPath(""))
	assert.Error(t, trashPath("."))
	assert.Error(t, trashPath(string(filepath.Separator)))

	home, err := os.UserHomeDir()
	assert.NoError(t, err)
	assert.Error(t, trashPath(home))

	cwd, err := os.Getwd()
	assert.NoError(t, err)
	assert.Error(t, trashPath(cwd))
	assert.Error(t, trashPath(firstPathElement(t, t.TempDir())))
	assert.Error(t, trashPath(relativePathToRoot(t)))
}

func TestTrashPathRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	assert.NoError(t, os.WriteFile(target, []byte("x"), 0644))
	link := filepath.Join(dir, "link")
	assert.NoError(t, os.Symlink(target, link))

	orig := moveToTrash
	moveToTrash = func(paths ...string) error {
		t.Fatalf("trash called for %v", paths)
		return nil
	}
	t.Cleanup(func() { moveToTrash = orig })

	assert.Error(t, trashPath(link))
	_, err := os.Stat(target)
	assert.NoError(t, err)
}

func TestCloneForceTrashesExactDestination(t *testing.T) {
	parent := t.TempDir()
	dest := filepath.Join(parent, "cloned")
	assert.NoError(t, os.MkdirAll(dest, 0755))
	marker := filepath.Join(dest, "keep.txt")
	assert.NoError(t, os.WriteFile(marker, []byte("x"), 0644))
	sibling := filepath.Join(parent, "sibling.txt")
	assert.NoError(t, os.WriteFile(sibling, []byte("y"), 0644))

	var got []string
	orig := moveToTrash
	moveToTrash = func(paths ...string) error {
		got = append(got, paths...)
		return nil
	}
	t.Cleanup(func() { moveToTrash = orig })

	d, err := NewDownload(filepath.Join(parent, "missing.git"), dest, true, nil, true)
	assert.NoError(t, err)
	t.Cleanup(d.Cleanup)

	n := d.cloneFromUrl(false)
	assert.Equal(t, int64(0), n)

	want, err := filepath.Abs(dest)
	assert.NoError(t, err)
	assert.Equal(t, []string{filepath.Clean(want)}, got)
	_, err = os.Stat(marker)
	assert.NoError(t, err)
	_, err = os.Stat(sibling)
	assert.NoError(t, err)
}

func firstPathElement(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	abs = filepath.Clean(abs)
	for {
		parent := filepath.Dir(abs)
		if parent == abs || isFilesystemRoot(parent) {
			return abs
		}
		abs = parent
	}
}

func relativePathToRoot(t *testing.T) string {
	t.Helper()
	rel := "."
	for i := 0; i < 64; i++ {
		abs, err := filepath.Abs(rel)
		if err != nil {
			t.Fatal(err)
		}
		if isFilesystemRoot(filepath.Clean(abs)) {
			return rel
		}
		rel = filepath.Join(rel, "..")
	}
	t.Fatal("did not reach filesystem root")
	return ""
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
