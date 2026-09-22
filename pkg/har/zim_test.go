package har

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cookiengineer/gozim/archive/zim"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoragePath(t *testing.T) {
	u, err := url.Parse("https://example.com/dir/page.html?q=1")
	require.NoError(t, err)
	assert.Equal(t, "dir/page.html%3Fq=1", storagePath(u, true))

	amp, err := url.Parse("https://example.com/logo.png?w=10&q=75")
	require.NoError(t, err)
	assert.Equal(t, "logo.png%3Fw=10%26q=75", storagePath(amp, true))
	assert.Equal(t, "logo.png?w=10&q=75", zimEntryPath(storagePath(amp, true)))
	assert.Equal(t, "/logo.png%3Fw=10%26q=75", zimHref(zimEntryPath(storagePath(amp, true))))

	css, err := url.Parse("https://example.com/style.css?v=2")
	require.NoError(t, err)
	assert.Equal(t, "style.css", storagePath(css, true))

	root, err := url.Parse("https://example.com/")
	require.NoError(t, err)
	assert.Equal(t, "index.html", storagePath(root, true))
}

func TestRewriteHTMLLinksAndOverlay(t *testing.T) {
	origin, _ := url.Parse("https://example.com/index.html")
	src := []byte(`<!doctype html><html><head><meta http-equiv="refresh" content="0;url=x"><title>Hi</title></head><body><a href="https://example.com/about">About</a><img src="https://cdn.test/pic.png"></body></html>`)
	out, err := rewriteHTML(src, origin, origin, nil)
	require.NoError(t, err)
	htmlOut := string(out)
	assert.Contains(t, htmlOut, `href="/about/index.html"`)
	assert.Contains(t, htmlOut, `src="/cdn.test/pic.png"`)
	assert.NotContains(t, strings.ToLower(htmlOut), `http-equiv="refresh"`)
	assert.Contains(t, htmlOut, `display: none !important`)
	assert.Contains(t, htmlOut, "cookie")
}

func TestRewriteStripsNextRuntime(t *testing.T) {
	origin, _ := url.Parse("https://example.com/lot/index.html")
	src := []byte(`<html><head><script src="/_next/static/chunks/main.js"></script><link rel="preload" as="script" href="/_next/static/chunks/webpack.js"><script>self.__next_f.push([1,"x"])</script></head><body><p>Lot 1</p><script>var theme="light"</script></body></html>`)
	out, err := rewriteHTML(src, origin, origin, nil)
	require.NoError(t, err)
	htmlOut := string(out)
	assert.NotContains(t, htmlOut, "__next_f")
	assert.NotContains(t, htmlOut, "/_next/static/chunks/main.js")
	assert.NotContains(t, htmlOut, "webpack.js")
	assert.Contains(t, htmlOut, "Lot 1")
	assert.Contains(t, htmlOut, `theme=`)
	assert.Contains(t, htmlOut, "light")
}

func TestRewriteQueryAndRedirectTarget(t *testing.T) {
	origin, _ := url.Parse("https://example.com/")
	src := []byte(`<html><body><img src="/logo.png?w=10&amp;q=75"><img src="https://cdn.test/a.jpg?tr=w-1&amp;h=2"></body></html>`)
	redirects := map[string]string{
		"cdn.test/a.jpg%3Ftr=w-1%26h=2": "ik.example/photo.jpg%3Ftr=w-1%2Ch-2",
	}
	out, err := rewriteHTML(src, origin, origin, redirects)
	require.NoError(t, err)
	htmlOut := string(out)
	assert.Contains(t, htmlOut, `src="/logo.png%3Fw=10%26q=75"`)
	assert.Contains(t, htmlOut, `src="/ik.example/photo.jpg%3Ftr=w-1%2Ch-2"`)
	assert.NotContains(t, htmlOut, "&amp;w=")
	assert.NotContains(t, htmlOut, "cdn.test")
}

func TestSizeFilterRemovesLargeNonMedia(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ok.html"), []byte("tiny"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "big.bin"), make([]byte, 3*1024*1024), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "photo.png"), make([]byte, 3*1024*1024), 0644))

	err := applySizeFilters(dir, ZimOptions{AnyMaxMB: 10, NotMediaMaxMB: 2})
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "ok.html"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "big.bin"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(dir, "photo.png"))
	assert.NoError(t, err)
}

func TestCrawlDepthAndRequisites(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><link rel="stylesheet" href="/style.css"></head><body><a href="/page.html">p</a><img src="/logo.png"></body></html>`))
	})
	mux.HandleFunc("/page.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><a href="/too-deep.html">no</a></body></html>`))
	})
	mux.HandleFunc("/too-deep.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>too deep</body></html>`))
	})
	mux.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write([]byte(`body { background: url(/bg.png); }`))
	})
	mux.HandleFunc("/logo.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png"))
	})
	mux.HandleFunc("/bg.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("bg"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	start, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	dir := t.TempDir()
	c := newCrawler(crawlConfig{
		Start:      start,
		WorkingDir: dir,
		MaxDepth:   1,
		Wait:       0,
	})
	require.NoError(t, c.Run())

	assert.FileExists(t, filepath.Join(dir, "index.html"))
	assert.FileExists(t, filepath.Join(dir, "page.html"))
	assert.FileExists(t, filepath.Join(dir, "style.css"))
	assert.FileExists(t, filepath.Join(dir, "logo.png"))
	_, err = os.Stat(filepath.Join(dir, "bg.png"))
	assert.True(t, os.IsNotExist(err), "css background images should not be crawled")
	_, err = os.Stat(filepath.Join(dir, "too-deep.html"))
	assert.True(t, os.IsNotExist(err))
}

func TestCrawlSkipsOffsiteAndLinkedImages(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>
<img src="/on-page.png">
<a href="/gallery.jpg">photo</a>
<iframe src="/ad.html"></iframe>
<link rel="preload" href="/preload.png" as="image">
</body></html>`))
	})
	mux.HandleFunc("/on-page.png", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/gallery.jpg", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("nope"))
	})
	mux.HandleFunc("/ad.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><img src="/ad.png"></body></html>`))
	})
	mux.HandleFunc("/ad.png", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ad"))
	})
	mux.HandleFunc("/preload.png", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("preload"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	start, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	dir := t.TempDir()
	c := newCrawler(crawlConfig{
		Start:            start,
		WorkingDir:       dir,
		MaxDepth:         1,
		Wait:             0,
		NoOverreachMedia: true,
	})
	require.NoError(t, c.Run())

	assert.FileExists(t, filepath.Join(dir, "on-page.png"))
	_, err = os.Stat(filepath.Join(dir, "gallery.jpg"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(dir, "ad.html"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(dir, "ad.png"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(dir, "preload.png"))
	assert.True(t, os.IsNotExist(err))
}

func TestCrawlRewritesRedirectedImage(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><img src="/api/pic?image=true&width=10"></body></html>`))
	})
	mux.HandleFunc("/api/pic", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/photos/a.jpg?tr=w-10%2Ch-20", http.StatusFound)
	})
	mux.HandleFunc("/photos/a.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("jpeg"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	start, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	dir := t.TempDir()
	c := newCrawler(crawlConfig{Start: start, WorkingDir: dir, MaxDepth: 0, Wait: 0})
	require.NoError(t, c.Run())
	require.NoError(t, rewriteTree(dir, start, c.redirects))

	assert.FileExists(t, filepath.Join(dir, "photos", "a.jpg%3Ftr=w-10%2Ch-20"))
	body, err := os.ReadFile(filepath.Join(dir, "index.html"))
	require.NoError(t, err)
	assert.Contains(t, string(body), `src="/photos/a.jpg%3Ftr=w-10%2Ch-20"`)
	assert.NotContains(t, string(body), "/api/pic")
}

func TestPackedCrawlResolvesOffsiteImage(t *testing.T) {
	images := http.NewServeMux()
	images.HandleFunc("/api/pic", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/photos/a.jpg?tr=w-10%2Ch-20", http.StatusFound)
	})
	images.HandleFunc("/photos/a.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("jpeg-bytes"))
	})
	images.HandleFunc("/logo.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("logo-bytes"))
	})
	images.HandleFunc("/clip.mp4", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write([]byte("video"))
	})
	imgSrv := httptest.NewServer(images)
	defer imgSrv.Close()

	pages := http.NewServeMux()
	pages.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>Lot</title></head><body>` +
			`<img src="http://cdn.test/api/pic?image=true&width=10">` +
			`<img src="http://cdn.test/logo.jpg">` +
			`<video src="http://cdn.test/clip.mp4"></video>` +
			`</body></html>`))
	})
	pageSrv := httptest.NewServer(pages)
	defer pageSrv.Close()

	start, err := url.Parse(pageSrv.URL + "/")
	require.NoError(t, err)
	dir := t.TempDir()
	c := newCrawler(crawlConfig{
		Start:            start,
		WorkingDir:       dir,
		MaxDepth:         0,
		Wait:             0,
		NoOverreachMedia: true,
	})
	transport := c.client.Transport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, splitErr := net.SplitHostPort(addr)
		if splitErr == nil && host == "cdn.test" {
			addr = imgSrv.Listener.Addr().String()
		}
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}
	c.client.Transport = transport

	require.NoError(t, c.Run())
	require.NoError(t, rewriteTree(dir, start, c.redirects))

	assert.FileExists(t, filepath.Join(dir, "cdn.test", "photos", "a.jpg%3Ftr=w-10%2Ch-20"))
	assert.FileExists(t, filepath.Join(dir, "cdn.test", "logo.jpg"))
	_, err = os.Stat(filepath.Join(dir, "cdn.test", "clip.mp4"))
	assert.True(t, os.IsNotExist(err))

	body, err := os.ReadFile(filepath.Join(dir, "index.html"))
	require.NoError(t, err)
	htmlOut := string(body)
	assert.Contains(t, htmlOut, `src="/cdn.test/photos/a.jpg%3Ftr=w-10%2Ch-20"`)
	assert.Contains(t, htmlOut, `src="/cdn.test/logo.jpg"`)
	assert.NotContains(t, htmlOut, "/api/pic")

	out := filepath.Join(t.TempDir(), "lot.zim")
	require.NoError(t, packZim(dir, out, zimPackMeta{
		Welcome:  "index.html",
		Language: "eng",
		Title:    "example.test",
		Name:     "lot",
	}))
	archive, err := zim.Open(out)
	require.NoError(t, err)
	defer archive.Close()

	photo, err := archive.EntryByPath("cdn.test/photos/a.jpg?tr=w-10,h-20")
	require.NoError(t, err)
	photoItem, err := photo.Item(false)
	require.NoError(t, err)
	photoData, err := photoItem.DataAll()
	require.NoError(t, err)
	assert.Equal(t, "jpeg-bytes", string(photoData))

	logo, err := archive.EntryByPath("cdn.test/logo.jpg")
	require.NoError(t, err)
	logoItem, err := logo.Item(false)
	require.NoError(t, err)
	logoData, err := logoItem.DataAll()
	require.NoError(t, err)
	assert.Equal(t, "logo-bytes", string(logoData))
}

func TestPackZimWithGozim(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html><head><title>Packed</title></head><body>hi</body></html>"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "style.css"), []byte("body{color:#000}"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pic.jpg%3Fw=1%26h=2"), []byte("jpeg"), 0644))

	out := filepath.Join(t.TempDir(), "site.zim")
	err := packZim(dir, out, zimPackMeta{
		Welcome:         "index.html",
		Illustration:    whitePNG48(),
		Title:           "example.com",
		Description:     "Packed",
		LongDescription: "Packed (created by har zim)",
		Creator:         "har",
		Publisher:       "har zim",
		Language:        "eng",
		Name:            "site",
		Source:          "example.com",
	})
	require.NoError(t, err)
	assert.FileExists(t, out)

	archive, err := zim.Open(out)
	require.NoError(t, err)
	defer archive.Close()

	title, ok := archive.Metadata("Title")
	require.True(t, ok)
	assert.Equal(t, "example.com", title)

	entry, err := archive.EntryByPath("index.html")
	require.NoError(t, err)
	item, err := entry.Item(false)
	require.NoError(t, err)
	data, err := item.DataAll()
	require.NoError(t, err)
	assert.Contains(t, string(data), "hi")

	pic, err := archive.EntryByPath("pic.jpg?w=1&h=2")
	require.NoError(t, err)
	picItem, err := pic.Item(false)
	require.NoError(t, err)
	picData, err := picItem.DataAll()
	require.NoError(t, err)
	assert.Equal(t, "jpeg", string(picData))

	assert.Equal(t, archive.EntryCount(), uint32(len(archive.TitleIndices())))
	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(raw), 96)
	titlePos := binary.LittleEndian.Uint64(raw[40:48])
	clusterPos := binary.LittleEndian.Uint64(raw[48:56])
	assert.Equal(t, uint64(archive.EntryCount())*4, clusterPos-titlePos)
	checksumPos := int(binary.LittleEndian.Uint64(raw[72:80]))
	require.Equal(t, len(raw)-16, checksumPos)
	sum := md5.Sum(raw[:checksumPos])
	assert.Equal(t, sum[:], raw[checksumPos:])
	assert.NotEqual(t, make([]byte, 16), raw[8:24])
	_, needsTerminator := mimeListNeedsTerminator(raw, int(binary.LittleEndian.Uint64(raw[56:64])))
	assert.False(t, needsTerminator)
}

func TestHandleZimSkipDownload(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html><head><title>Local</title></head><body>ok</body></html>"), 0644))

	cwd := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(cwd))
	t.Cleanup(func() { _ = os.Chdir(orig) })

	d, err := NewDownload("http://example.com/", "", true, nil, false)
	require.NoError(t, err)
	defer d.Cleanup()

	err = HandleZim(d, ZimOptions{
		URL:          "http://example.com/",
		OutputFile:   "local.zim",
		WorkingDir:   dir,
		SkipDownload: true,
		Turbo:        true,
		Language:     "eng",
	})
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(cwd, "local.zim"))
}
