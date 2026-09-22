package har

import (
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const zimUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.41 Safari/537.36"

type crawlConfig struct {
	Start            *url.URL
	WorkingDir       string
	MaxDepth         int
	Wait             time.Duration
	Reject           []string
	NoOverreachMedia bool
	OverreachAny     bool
}

type queueItem struct {
	URL       *url.URL
	Depth     int
	Kind      linkKind
	Requisite bool
}

type linkKind int

const (
	linkPage linkKind = iota
	linkAsset
	linkMedia
)

type crawler struct {
	cfg       crawlConfig
	origin    *url.URL
	client    *http.Client
	visited   map[string]struct{}
	redirects map[string]string
}

func newCrawler(cfg crawlConfig) *crawler {
	origin := *cfg.Start
	origin.Path = ""
	origin.RawQuery = ""
	origin.Fragment = ""
	return &crawler{
		cfg:    cfg,
		origin: &origin,
		client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		},
		visited:   make(map[string]struct{}),
		redirects: make(map[string]string),
	}
}

func (c *crawler) Run() error {
	logger := GetLogger()
	queue := []queueItem{{URL: c.cfg.Start, Depth: 0, Kind: linkPage}}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		key := canonicalURL(item.URL)
		if _, seen := c.visited[key]; seen {
			continue
		}
		if c.rejected(item.URL) {
			continue
		}
		c.visited[key] = struct{}{}

		same := c.sameOrigin(item.URL)
		if !same {
			if item.Kind == linkPage && !c.cfg.OverreachAny {
				continue
			}
			// Images embedded on a crawled page are saved even when they are
			// hosted elsewhere. Other off-site media stays behind the flag.
			if item.Kind == linkMedia && c.cfg.NoOverreachMedia && !item.Requisite {
				continue
			}
		}

		if c.cfg.Wait > 0 {
			time.Sleep(c.cfg.Wait)
		}

		body, contentType, finalURL, err := c.fetch(item.URL)
		if err != nil {
			logger.WithError(err).Info("skip ", item.URL)
			continue
		}
		rel := storagePath(finalURL, c.sameOrigin(finalURL))
		if requested := storagePath(item.URL, c.sameOrigin(item.URL)); requested != rel {
			c.redirects[requested] = rel
		}
		dest := filepath.Join(c.cfg.WorkingDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, body, 0644); err != nil {
			return err
		}
		logger.Info("fetched ", finalURL.String())

		// Only HTML pages we intentionally crawled may discover more URLs.
		// Do not spider error pages, iframe documents, or other HTML fetched as assets.
		base := finalURL
		var discovered []discoveredLink
		switch {
		case item.Kind == linkPage && isHTML(contentType, rel):
			discovered = extractHTMLLinks(body, base)
		case isCSS(contentType, rel):
			discovered = extractCSSLinks(string(body), base)
		}

		for _, link := range discovered {
			nextDepth := item.Depth
			if link.Kind == linkPage {
				if !c.sameOrigin(link.URL) {
					if c.cfg.OverreachAny {
						queue = append(queue, queueItem{URL: link.URL, Depth: item.Depth, Kind: linkAsset})
					}
					continue
				}
				nextDepth = item.Depth + 1
				if nextDepth > c.cfg.MaxDepth {
					continue
				}
			}
			queue = append(queue, queueItem{URL: link.URL, Depth: nextDepth, Kind: link.Kind, Requisite: link.Requisite})
		}
	}
	return nil
}

func (c *crawler) fetch(u *url.URL) ([]byte, string, *url.URL, error) {
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", u, err
	}
	req.Header.Set("User-Agent", zimUserAgent)
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Referer", c.origin.String())

	var resp *http.Response
	var lastErr error
	for attempt := 0; attempt < 6; attempt++ {
		resp, lastErr = c.client.Do(req)
		if lastErr == nil && resp != nil && resp.StatusCode < 400 {
			break
		}
		if resp != nil {
			if lastErr == nil {
				lastErr = fmt.Errorf("GET %s: status %d", u, resp.StatusCode)
			}
			resp.Body.Close()
			resp = nil
		} else if lastErr != nil {
			lastErr = fmt.Errorf("GET %s: %w", u, lastErr)
		}
		time.Sleep(200 * time.Millisecond)
	}
	if lastErr != nil || resp == nil {
		return nil, "", u, lastErr
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", u, err
	}
	final := resp.Request.URL
	return body, resp.Header.Get("Content-Type"), final, nil
}

func (c *crawler) sameOrigin(u *url.URL) bool {
	return normalizeHost(u.Hostname()) == normalizeHost(c.origin.Hostname())
}

func (c *crawler) rejected(u *url.URL) bool {
	ext := strings.ToLower(filepath.Ext(u.Path))
	if ext == "" {
		return false
	}
	for _, reject := range c.cfg.Reject {
		if ext == reject {
			return true
		}
	}
	return false
}

func normalizeHost(host string) string {
	host = strings.ToLower(host)
	return strings.TrimPrefix(host, "www.")
}

func canonicalURL(u *url.URL) string {
	cp := *u
	cp.Fragment = ""
	return cp.String()
}

func storagePath(u *url.URL, sameOrigin bool) string {
	p := strings.TrimPrefix(u.Path, "/")
	ext := strings.ToLower(filepath.Ext(u.Path))
	if p == "" || strings.HasSuffix(u.Path, "/") {
		p = strings.TrimSuffix(p, "/")
		if p == "" {
			p = "index.html"
		} else {
			p = p + "/index.html"
		}
	} else if ext == "" {
		p = p + "/index.html"
	}
	if ext == ".css" {
		if !sameOrigin {
			p = filepath.ToSlash(filepath.Join(u.Hostname(), p))
		}
		return p
	}
	if u.RawQuery != "" {
		// Keep the query in the filename, but hide separators the filesystem
		// and a later Kiwix lookup would treat specially.
		q := strings.ReplaceAll(u.RawQuery, "/", "%2F")
		q = strings.ReplaceAll(q, "&", "%26")
		p = p + "%3F" + q
	}
	if !sameOrigin {
		p = filepath.ToSlash(filepath.Join(u.Hostname(), p))
	}
	return p
}

func isHTML(contentType, rel string) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(strings.Split(rel, "%3F")[0]))
	return ext == ".html" || ext == ".htm" || ext == ".php"
}

func isCSS(contentType, rel string) bool {
	if strings.Contains(strings.ToLower(contentType), "text/css") {
		return true
	}
	return strings.ToLower(filepath.Ext(strings.Split(rel, "%3F")[0])) == ".css"
}

type discoveredLink struct {
	URL       *url.URL
	Kind      linkKind
	Requisite bool
}

func extractHTMLLinks(body []byte, base *url.URL) []discoveredLink {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil
	}
	var links []discoveredLink
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "a":
				if u := attrURL(n, "href", base); u != nil {
					if classifyHref(u) != linkPage {
						break
					}
					links = append(links, discoveredLink{URL: u, Kind: linkPage})
				}
			case "img", "script", "source", "track", "video", "audio":
				requisite := n.Data == "img" || ((n.Data == "source" || n.Data == "track") && !insideVideoOrAudio(n))
				if u := attrURL(n, "src", base); u != nil {
					links = append(links, discoveredLink{URL: u, Kind: classifyAsset(u), Requisite: requisite})
				}
				for _, u := range srcsetURLs(attrValue(n, "srcset"), base) {
					links = append(links, discoveredLink{URL: u, Kind: classifyAsset(u), Requisite: requisite})
				}
			case "link":
				rel := strings.ToLower(attrValue(n, "rel"))
				if !strings.Contains(rel, "stylesheet") && !strings.Contains(rel, "icon") {
					break
				}
				if u := attrURL(n, "href", base); u != nil {
					links = append(links, discoveredLink{URL: u, Kind: classifyAsset(u)})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return links
}

func extractCSSLinks(css string, base *url.URL) []discoveredLink {
	var links []discoveredLink
	rest := css
	for {
		i := strings.Index(strings.ToLower(rest), "url(")
		if i < 0 {
			break
		}
		after := rest[i+4:]
		end := strings.IndexByte(after, ')')
		if end < 0 {
			break
		}
		raw := strings.TrimSpace(after[:end])
		raw = strings.Trim(raw, `"'`)
		rest = after[end+1:]
		if raw == "" || strings.HasPrefix(raw, "data:") {
			continue
		}
		ref, err := url.Parse(raw)
		if err != nil {
			continue
		}
		u := base.ResolveReference(ref)
		u.Fragment = ""
		// Site-wide stylesheets reference many theme images that are not on this page.
		// Only follow further CSS (@import), not url() images/fonts.
		if strings.ToLower(filepath.Ext(u.Path)) != ".css" {
			continue
		}
		links = append(links, discoveredLink{URL: u, Kind: classifyAsset(u)})
	}
	return links
}

func attrValue(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func attrURL(n *html.Node, key string, base *url.URL) *url.URL {
	val := strings.TrimSpace(attrValue(n, key))
	if val == "" || strings.HasPrefix(val, "data:") || strings.HasPrefix(val, "javascript:") || strings.HasPrefix(val, "mailto:") {
		return nil
	}
	ref, err := url.Parse(val)
	if err != nil {
		return nil
	}
	u := base.ResolveReference(ref)
	u.Fragment = ""
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil
	}
	return u
}

func srcsetURLs(srcset string, base *url.URL) []*url.URL {
	var out []*url.URL
	for _, part := range strings.Split(srcset, ",") {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) == 0 {
			continue
		}
		ref, err := url.Parse(fields[0])
		if err != nil {
			continue
		}
		u := base.ResolveReference(ref)
		u.Fragment = ""
		out = append(out, u)
	}
	return out
}

func insideVideoOrAudio(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && (p.Data == "video" || p.Data == "audio") {
			return true
		}
	}
	return false
}

func classifyHref(u *url.URL) linkKind {
	ext := strings.ToLower(filepath.Ext(u.Path))
	if ext == "" || ext == ".html" || ext == ".htm" || ext == ".php" {
		return linkPage
	}
	if isMediaExt(ext) {
		return linkMedia
	}
	return linkAsset
}

func classifyAsset(u *url.URL) linkKind {
	ext := strings.ToLower(filepath.Ext(u.Path))
	if isMediaExt(ext) {
		return linkMedia
	}
	return linkAsset
}

func isMediaExt(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".bmp", ".ico",
		".webm", ".ogg", ".mp3", ".aac", ".wav", ".mpeg", ".mpg", ".flac",
		".fla", ".flv", ".ac3", ".au", ".mka", ".swf", ".mp4", ".f4v",
		".ogv", ".avi", ".wmv", ".mkv", ".divx", ".aif", ".wma",
		".epub", ".pdf", ".pdb", ".xls", ".xlsx", ".doc", ".docx", ".ppt", ".pptx",
		".odt", ".ods", ".odp":
		return true
	default:
		return false
	}
}

func rejectExtensions(includeZip, includeExe, includeAny bool) []string {
	if includeAny {
		return nil
	}
	reject := []string{
		".img", ".dsk", ".nrg", ".iso", ".cue", ".daa", ".ass", ".ipa",
		".ace", ".toast", ".vcd", ".vol", ".bak", ".cab", ".tmp",
	}
	if !includeZip {
		reject = append(reject, ".lz", ".gz", ".zip", ".rar", ".7z", ".tar", ".xz", ".bz2")
	}
	if !includeExe {
		reject = append(reject, ".exe", ".deb", ".rpm", ".dmg", ".bin", ".msi", ".apk")
	}
	return reject
}

// zimEntryPath is the path Kiwix looks up after decoding the request URL.
// A query is stored with a literal '?' so the decoded path matches.
func zimEntryPath(disk string) string {
	disk = filepath.ToSlash(disk)
	decoded, err := url.PathUnescape(disk)
	if err != nil {
		return strings.ReplaceAll(disk, "%3F", "?")
	}
	return decoded
}

// zimHref is the link to put in HTML. '?' and '&' stay percent-encoded so the
// browser does not split them off before Kiwix decodes the path.
func zimHref(entry string) string {
	path, query, ok := strings.Cut(entry, "?")
	if !ok {
		return "/" + path
	}
	var b strings.Builder
	b.Grow(len(entry) + 8)
	b.WriteByte('/')
	b.WriteString(path)
	b.WriteString("%3F")
	for _, r := range query {
		switch r {
		case '&':
			b.WriteString("%26")
		case '/':
			b.WriteString("%2F")
		case '?':
			b.WriteString("%3F")
		case ',':
			b.WriteString("%2C")
		case ' ':
			b.WriteString("%20")
		case '#':
			b.WriteString("%23")
		case '%':
			b.WriteString("%25")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func mimeFromPath(rel string) string {
	clean := rel
	if i := strings.IndexAny(clean, "?"); i >= 0 {
		clean = clean[:i]
	}
	clean = strings.Split(clean, "%3F")[0]
	ext := strings.ToLower(filepath.Ext(clean))
	if t := mime.TypeByExtension(ext); t != "" {
		return t
	}
	switch ext {
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "text/javascript"
	case ".json":
		return "application/json"
	case ".svg":
		return "image/svg+xml"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".php":
		return "text/html"
	default:
		return "application/octet-stream"
	}
}
