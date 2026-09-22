package har

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cookiengineer/gozim/archive/zim"
)

// ZimOptions configures har zim (wget-2-zim flag parity, no Xapian index).
type ZimOptions struct {
	URL              string
	OutputFile       string
	MaxDepth         int
	AnyMaxMB         int
	NotMediaMaxMB    int
	PictureMaxMB     int
	DocumentMaxMB    int
	MusicMaxMB       int
	VideoMaxMB       int
	IncludeZip       bool
	IncludeExe       bool
	IncludeAny       bool
	NoOverreachMedia bool
	OverreachMedia   bool
	OverreachAny     bool
	Turbo            bool
	SkipDownload     bool
	WorkingDir       string
	Timestamp        bool
	Creator          string
	Publisher        string
	Description      string
	LongDescription  string
	Language         string
}

func (o *ZimOptions) applyDefaults() {
	if o.MaxDepth < 0 {
		o.MaxDepth = 1
	}
	if o.AnyMaxMB <= 0 {
		o.AnyMaxMB = 128
	}
	if o.NotMediaMaxMB <= 0 {
		o.NotMediaMaxMB = 2
	}
	if o.Creator == "" {
		o.Creator = "https://github.com/sio2boss/har"
	}
	if o.Publisher == "" {
		o.Publisher = "har zim"
	}
	if o.Language == "" {
		o.Language = "eng"
	}
}

// HandleZim crawls URL and packs a Kiwix-openable .zim using cookiengineer/gozim.
func HandleZim(d *Download, opts ZimOptions) error {
	logger := GetLogger()
	opts.applyDefaults()

	start, err := url.Parse(opts.URL)
	if err != nil || start.Scheme == "" || start.Host == "" {
		return fmt.Errorf("invalid URL: %s", opts.URL)
	}

	workingDir := opts.WorkingDir
	ownWorkingDir := false
	if workingDir == "" {
		workingDir = filepath.Join(d.TempDir, "site")
		ownWorkingDir = true
	}
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		return fmt.Errorf("working directory: %w", err)
	}

	var redirects map[string]string
	if !opts.SkipDownload {
		wait := 400 * time.Millisecond
		if opts.Turbo {
			wait = 0
		}
		crawler := newCrawler(crawlConfig{
			Start:            start,
			WorkingDir:       workingDir,
			MaxDepth:         opts.MaxDepth,
			Wait:             wait,
			Reject:           rejectExtensions(opts.IncludeZip, opts.IncludeExe, opts.IncludeAny),
			NoOverreachMedia: opts.NoOverreachMedia || (!opts.OverreachMedia && !opts.OverreachAny),
			OverreachAny:     opts.OverreachAny,
		})
		if err := crawler.Run(); err != nil {
			return err
		}
		redirects = crawler.redirects
	} else if entries, err := os.ReadDir(workingDir); err != nil || len(entries) == 0 {
		return fmt.Errorf("--skip-download requires an existing working directory with files")
	}

	if err := rewriteTree(workingDir, start, redirects); err != nil {
		return err
	}
	if err := applySizeFilters(workingDir, opts); err != nil {
		return err
	}

	welcome, err := chooseWelcome(workingDir, start)
	if err != nil {
		return err
	}

	illustration, err := makeIllustration(workingDir)
	if err != nil {
		logger.WithError(err).Info("favicon conversion failed, using a generated icon")
		illustration = whitePNG48()
	}

	description := opts.Description
	if description == "" {
		description = htmlTitle(filepath.Join(workingDir, welcome))
		if description == "" {
			description = start.Host
		}
	}
	longDescription := opts.LongDescription
	if longDescription == "" {
		longDescription = description + " (created by har zim)"
	}

	output := opts.OutputFile
	if output == "" {
		output = start.Hostname()
	}
	output = strings.TrimSuffix(output, ".zim")
	if opts.Timestamp {
		output = output + "_" + time.Now().Format("20060102_150405")
	}
	output = output + ".zim"

	name := strings.TrimSuffix(filepath.Base(output), ".zim")
	if err := packZim(workingDir, output, zimPackMeta{
		Welcome:         welcome,
		Illustration:    illustration,
		Title:           start.Hostname(),
		Description:     description,
		LongDescription: longDescription,
		Creator:         opts.Creator,
		Publisher:       opts.Publisher,
		Language:        opts.Language,
		Name:            name,
		Source:          start.Hostname(),
	}); err != nil {
		return err
	}

	logger.Info("created ", output)
	if ownWorkingDir {
		return nil
	}
	logger.Info("working directory left at ", workingDir)
	return nil
}

type zimPackMeta struct {
	Welcome         string
	Illustration    []byte
	Title           string
	Description     string
	LongDescription string
	Creator         string
	Publisher       string
	Language        string
	Name            string
	Source          string
}

func packZim(workingDir, output string, meta zimPackMeta) error {
	writer := zim.NewWriter().
		SetCompression(zim.CompressionZstd).
		SetClusterSize(2*zim.Megabyte).
		SetIndexing(false, meta.Language).
		SetMainPath(zimEntryPath(meta.Welcome))

	if err := writer.Create(output); err != nil {
		return fmt.Errorf("create zim: %w", err)
	}

	if err := writer.AddMetadata("Title", meta.Title); err != nil {
		return err
	}
	if err := writer.AddMetadata("Language", meta.Language); err != nil {
		return err
	}
	if err := writer.AddMetadata("Date", time.Now().Format("2006-01-02")); err != nil {
		return err
	}
	if err := writer.AddMetadata("Description", meta.Description); err != nil {
		return err
	}
	if err := writer.AddMetadata("LongDescription", meta.LongDescription); err != nil {
		return err
	}
	if err := writer.AddMetadata("Creator", meta.Creator); err != nil {
		return err
	}
	if err := writer.AddMetadata("Publisher", meta.Publisher); err != nil {
		return err
	}
	if err := writer.AddMetadata("Name", meta.Name); err != nil {
		return err
	}
	if err := writer.AddMetadata("Source", meta.Source); err != nil {
		return err
	}
	if err := writer.AddMetadata("Scraper", "har zim"); err != nil {
		return err
	}
	if len(meta.Illustration) > 0 {
		if err := writer.AddIllustration(48, meta.Illustration); err != nil {
			return err
		}
	}

	err := filepath.Walk(workingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(workingDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "zim_favicon.png" {
			return nil
		}
		entryPath := zimEntryPath(rel)
		mimeType := mimeFromPath(entryPath)
		title := entryTitle(path, rel)
		item, err := zim.NewFileItem(entryPath, mimeType, title, path)
		if err != nil {
			return err
		}
		return writer.AddItem(item)
	})
	if err != nil {
		return err
	}

	if err := writer.Finish(); err != nil {
		return fmt.Errorf("finish zim: %w", err)
	}
	if err := RepairZim(output); err != nil {
		return fmt.Errorf("finalize zim: %w", err)
	}
	return nil
}

func chooseWelcome(workingDir string, start *url.URL) (string, error) {
	candidate := storagePath(start, true)
	if _, err := os.Stat(filepath.Join(workingDir, filepath.FromSlash(candidate))); err == nil {
		return candidate, nil
	}
	for _, name := range []string{"index.html", "index.htm", "index.php"} {
		if _, err := os.Stat(filepath.Join(workingDir, name)); err == nil {
			return name, nil
		}
	}
	var found string
	_ = filepath.Walk(workingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || found != "" {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".html" || ext == ".htm" || ext == ".php" {
			rel, relErr := filepath.Rel(workingDir, path)
			if relErr == nil {
				found = filepath.ToSlash(rel)
			}
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("no HTML welcome page found in %s", workingDir)
	}
	return found, nil
}

func entryTitle(absPath, rel string) string {
	if title := htmlTitle(absPath); title != "" {
		return title
	}
	base := filepath.Base(rel)
	if base == "." || base == string(filepath.Separator) {
		return rel
	}
	return base
}
