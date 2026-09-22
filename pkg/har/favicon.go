package har

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"

	_ "image/gif"
	_ "image/jpeg"
)

func makeIllustration(workingDir string) ([]byte, error) {
	src := findFavicon(workingDir)
	dest := filepath.Join(workingDir, "zim_favicon.png")

	if src != "" {
		if converted := convertWithImageMagick(src, dest); converted {
			if data, err := os.ReadFile(dest); err == nil && len(data) > 0 {
				return data, nil
			}
		}
		if data, err := resizeImageFile(src, 48, 48); err == nil {
			_ = os.WriteFile(dest, data, 0644)
			return data, nil
		}
	}

	data := whitePNG48()
	_ = os.WriteFile(dest, data, 0644)
	return data, nil
}

func findFavicon(workingDir string) string {
	matches, _ := filepath.Glob(filepath.Join(workingDir, "favicon*"))
	for i := len(matches) - 1; i >= 0; i-- {
		info, err := os.Stat(matches[i])
		if err == nil && !info.IsDir() {
			return matches[i]
		}
	}
	return ""
}

func convertWithImageMagick(src, dest string) bool {
	for _, bin := range []string{"magick", "convert"} {
		if _, err := exec.LookPath(bin); err != nil {
			continue
		}
		var cmd *exec.Cmd
		if bin == "magick" {
			cmd = exec.Command(bin, src+"[0]", "-resize", "48x48", dest)
		} else {
			cmd = exec.Command(bin, src+"[0]", "-define", "icon:auto-resize=48", dest)
		}
		if err := cmd.Run(); err == nil {
			if info, err := os.Stat(dest); err == nil && info.Size() > 0 {
				return true
			}
		}
	}
	return false
}

func resizeImageFile(path string, w, h int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sx := sb.Min.X + x*sb.Dx()/w
			sy := sb.Min.Y + y*sb.Dy()/h
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func whitePNG48() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 48, 48))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
