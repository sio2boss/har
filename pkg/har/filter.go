package har

import (
	"os"
	"path/filepath"
	"strings"
)

func applySizeFilters(workingDir string, opts ZimOptions) error {
	anyMax := int64(opts.AnyMaxMB) * 1024 * 1024
	notMediaMax := int64(opts.NotMediaMaxMB) * 1024 * 1024
	pictureMax := int64(opts.PictureMaxMB) * 1024 * 1024
	documentMax := int64(opts.DocumentMaxMB) * 1024 * 1024
	musicMax := int64(opts.MusicMaxMB) * 1024 * 1024
	videoMax := int64(opts.VideoMaxMB) * 1024 * 1024

	return filepath.Walk(workingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		size := info.Size()
		ext := strings.ToLower(filepath.Ext(strings.Split(info.Name(), "%3F")[0]))
		if anyMax > 0 && size > anyMax {
			return os.Remove(path)
		}
		switch {
		case isPictureExt(ext):
			if pictureMax > 0 && size > pictureMax {
				return os.Remove(path)
			}
		case isDocumentExt(ext):
			if documentMax > 0 && size > documentMax {
				return os.Remove(path)
			}
		case isMusicExt(ext):
			if musicMax > 0 && size > musicMax {
				return os.Remove(path)
			}
		case isVideoExt(ext):
			if videoMax > 0 && size > videoMax {
				return os.Remove(path)
			}
		default:
			if notMediaMax > 0 && size > notMediaMax {
				return os.Remove(path)
			}
		}
		return nil
	})
}

func isPictureExt(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".bmp", ".ico":
		return true
	default:
		return false
	}
}

func isDocumentExt(ext string) bool {
	switch ext {
	case ".pdf", ".epub", ".pdb", ".xls", ".xlsx", ".doc", ".docx", ".odt", ".ods", ".odp", ".ppt", ".pptx":
		return true
	default:
		return false
	}
}

func isMusicExt(ext string) bool {
	switch ext {
	case ".aif", ".ogg", ".wav", ".aac", ".mp3", ".flac", ".wma", ".amr", ".fla", ".ac3", ".au", ".mka":
		return true
	default:
		return false
	}
}

func isVideoExt(ext string) bool {
	switch ext {
	case ".3gp", ".avi", ".flv", ".h264", ".mov", ".mpeg", ".mpg", ".swf", ".wmv", ".mkv", ".mp4", ".divx", ".f4v", ".ogv", ".webm":
		return true
	default:
		return false
	}
}
