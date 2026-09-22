package har

import (
	"bytes"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

var overlayHideWords = []string{
	"cookie", "banner", "disclaimer", "consent", "gdpr", "privacy", "popup",
	"adsby", "adsense", "advert", "sponsored", "adcontainer", "-ads-", "-ad-",
	"ads_", "ads-", "_ads", "leaderboard-", "ad-wrapper", "adholder",
	"adslot", "adspace", "adspot", "adv-", "boxad", "contentad", "footer-ad", "header-ad",
}

func rewriteTree(workingDir string, origin *url.URL, redirects map[string]string) error {
	return filepath.Walk(workingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, relErr := filepath.Rel(workingDir, path)
		if relErr != nil {
			return relErr
		}
		base := localFileBase(origin, filepath.ToSlash(rel))
		ext := strings.ToLower(filepath.Ext(strings.Split(info.Name(), "%3F")[0]))
		switch ext {
		case ".html", ".htm", ".php":
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rewritten, err := rewriteHTML(data, origin, base, redirects)
			if err != nil {
				return err
			}
			return os.WriteFile(path, rewritten, info.Mode())
		case ".css":
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(path, []byte(rewriteCSS(string(data), origin, base, redirects)), info.Mode())
		default:
			return nil
		}
	})
}

func localFileBase(origin *url.URL, rel string) *url.URL {
	u := *origin
	u.Fragment = ""
	entry := zimEntryPath(rel)
	path, query, ok := strings.Cut(entry, "?")
	u.Path = "/" + path
	u.RawQuery = ""
	if ok {
		u.RawQuery = query
	}
	return &u
}

func rewriteHTML(src []byte, origin, base *url.URL, redirects map[string]string) ([]byte, error) {
	doc, err := html.Parse(bytes.NewReader(src))
	if err != nil {
		return src, nil
	}

	var head *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "head" {
				head = n
			}
			if n.Data == "meta" && strings.EqualFold(attrValue(n, "http-equiv"), "refresh") {
				if n.Parent != nil {
					n.Parent.RemoveChild(n)
					return
				}
			}
			if n.Parent != nil && isClientRuntime(n) {
				n.Parent.RemoveChild(n)
				return
			}
			rewriteNodeAttrs(n, origin, base, redirects)
		}
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			walk(c)
			c = next
		}
	}
	walk(doc)
	injectOverlayCSS(doc, head)

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return src, err
	}
	return buf.Bytes(), nil
}

// StripClientRuntime removes the Next.js client runtime so Kiwix shows the
// server-rendered HTML instead of the framework error page.
func StripClientRuntime(src []byte) ([]byte, error) {
	doc, err := html.Parse(bytes.NewReader(src))
	if err != nil {
		return src, nil
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Parent != nil && isClientRuntime(n) {
			n.Parent.RemoveChild(n)
			return
		}
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			walk(c)
			c = next
		}
	}
	walk(doc)
	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return src, err
	}
	return buf.Bytes(), nil
}

func isClientRuntime(n *html.Node) bool {
	switch n.Data {
	case "script":
		src := attrValue(n, "src")
		if strings.Contains(src, "/_next/") {
			return true
		}
		return strings.Contains(nodeText(n), "__next_f")
	case "link":
		rel := strings.ToLower(attrValue(n, "rel"))
		if rel != "preload" && rel != "modulepreload" {
			return false
		}
		if strings.EqualFold(attrValue(n, "as"), "script") || rel == "modulepreload" {
			return strings.Contains(attrValue(n, "href"), "/_next/")
		}
	}
	return false
}

func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func rewriteNodeAttrs(n *html.Node, origin, base *url.URL, redirects map[string]string) {
	for i, a := range n.Attr {
		switch a.Key {
		case "href", "src", "poster":
			n.Attr[i].Val = rewriteURLString(a.Val, origin, base, redirects)
		case "srcset":
			n.Attr[i].Val = rewriteSrcset(a.Val, origin, base, redirects)
		}
	}
}

func rewriteSrcset(srcset string, origin, base *url.URL, redirects map[string]string) string {
	parts := strings.Split(srcset, ",")
	for i, part := range parts {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) == 0 {
			continue
		}
		fields[0] = rewriteURLString(fields[0], origin, base, redirects)
		parts[i] = strings.Join(fields, " ")
	}
	return strings.Join(parts, ", ")
}

func rewriteCSS(css string, origin, base *url.URL, redirects map[string]string) string {
	var b strings.Builder
	rest := css
	for {
		idx := strings.Index(strings.ToLower(rest), "url(")
		if idx < 0 {
			b.WriteString(rest)
			break
		}
		b.WriteString(rest[:idx])
		after := rest[idx+4:]
		end := strings.IndexByte(after, ')')
		if end < 0 {
			b.WriteString("url(")
			b.WriteString(after)
			break
		}
		raw := strings.TrimSpace(after[:end])
		quote := ""
		if strings.HasPrefix(raw, `"`) || strings.HasPrefix(raw, `'`) {
			quote = raw[:1]
			raw = strings.Trim(raw, `"'`)
		}
		rewritten := rewriteURLString(raw, origin, base, redirects)
		b.WriteString("url(")
		b.WriteString(quote)
		b.WriteString(rewritten)
		b.WriteString(quote)
		b.WriteByte(')')
		rest = after[end+1:]
	}
	return b.String()
}

func rewriteURLString(raw string, origin, base *url.URL, redirects map[string]string) string {
	val := strings.TrimSpace(raw)
	if val == "" || strings.HasPrefix(val, "data:") || strings.HasPrefix(val, "javascript:") ||
		strings.HasPrefix(val, "mailto:") || strings.HasPrefix(val, "#") {
		return raw
	}
	ref, err := url.Parse(val)
	if err != nil {
		return encodeLocalQuery(val)
	}
	resolved := base.ResolveReference(ref)
	var path string
	if normalizeHost(resolved.Hostname()) != normalizeHost(origin.Hostname()) {
		if resolved.Scheme != "http" && resolved.Scheme != "https" {
			return raw
		}
		path = storagePath(resolved, false)
	} else {
		path = storagePath(resolved, true)
	}
	if alias, ok := redirects[path]; ok {
		path = alias
	}
	return zimHref(zimEntryPath(path))
}

func encodeLocalQuery(val string) string {
	if strings.Contains(val, ".css?") {
		return strings.Split(val, "?")[0]
	}
	return strings.Replace(val, "?", "%3F", 1)
}

func injectOverlayCSS(doc, head *html.Node) {
	if head == nil {
		htmlNode := findElement(doc, "html")
		if htmlNode == nil {
			return
		}
		head = &html.Node{Type: html.ElementNode, Data: "head"}
		htmlNode.InsertBefore(head, htmlNode.FirstChild)
	}
	style := &html.Node{
		Type: html.ElementNode,
		Data: "style",
		Attr: []html.Attribute{{Key: "type", Val: "text/css"}},
	}
	style.AppendChild(&html.Node{Type: html.TextNode, Data: overlayHideCSS()})
	head.InsertBefore(style, head.FirstChild)
}

func overlayHideCSS() string {
	var b strings.Builder
	b.WriteString(`[class*="__useless__"]`)
	for _, word := range overlayHideWords {
		b.WriteString(`, [id*="`)
		b.WriteString(word)
		b.WriteString(`"], [class*="`)
		b.WriteString(word)
		b.WriteString(`"]`)
	}
	b.WriteString(` { display: none !important; } body { overflow: auto !important; }`)
	return b.String()
}

func findElement(n *html.Node, name string) *html.Node {
	if n.Type == html.ElementNode && n.Data == name {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findElement(c, name); found != nil {
			return found
		}
	}
	return nil
}

func htmlTitle(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return ""
	}
	var title string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if title != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			title = strings.TrimSpace(n.FirstChild.Data)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return title
}
