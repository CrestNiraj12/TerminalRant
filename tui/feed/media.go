package feed

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"net/url"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CrestNiraj12/terminalrant/domain"
)

const (
	previewASCIIWidth  = 17
	previewASCIIHeight = 14
	previewTileWidth   = previewASCIIWidth * 2
	previewTileHeight  = previewASCIIHeight
	previewPaneWidth   = 72
)

func (m *Model) ensureMediaPreviewCmd() tea.Cmd {
	if !m.showMediaPreview {
		return nil
	}
	r := m.getSelectedRant()
	if m.showDetail {
		if m.focusedRant != nil {
			r = *m.focusedRant
		} else if len(m.rants) > 0 && m.cursor >= 0 && m.cursor < len(m.rants) {
			r = m.rants[m.cursor].Rant
		}
	}
	targets := mediaPreviewTargets(r.Media)
	if len(targets) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(targets))
	for _, target := range targets {
		baseKey := mediaPreviewBaseKey(target.URL)
		if _, ok := m.mediaPreview[baseKey]; ok {
			continue
		}
		if m.mediaLoading[baseKey] {
			continue
		}
		m.mediaLoading[baseKey] = true
		cmds = append(cmds, fetchMediaPreview(target.URL, target.FallbackURL, baseKey, previewASCIIWidth, previewASCIIHeight, target.Animated))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func mediaPreviewURLs(media []domain.MediaAttachment) []string {
	targets := mediaPreviewTargets(media)
	out := make([]string, 0, len(targets))
	for _, t := range targets {
		out = append(out, t.URL)
	}
	return out
}

type mediaPreviewTarget struct {
	URL         string
	FallbackURL string
	Animated    bool
	Description string
}

func mediaPreviewTargets(media []domain.MediaAttachment) []mediaPreviewTarget {
	out := make([]mediaPreviewTarget, 0, len(media))
	seen := make(map[string]struct{}, len(media))
	for _, m := range media {
		t := strings.ToLower(strings.TrimSpace(m.Type))
		animated := false
		url := ""
		switch t {
		case "video", "gifv":
			animated = true
			// Prefer the original media URL for actual video frame extraction.
			url = strings.TrimSpace(m.URL)
			if url == "" {
				url = strings.TrimSpace(m.PreviewURL)
			}
			fallback := strings.TrimSpace(m.PreviewURL)
			if fallback == url {
				fallback = ""
			}
			if url == "" {
				continue
			}
			if _, ok := seen[url]; ok {
				continue
			}
			seen[url] = struct{}{}
			out = append(out, mediaPreviewTarget{
				URL:         url,
				FallbackURL: fallback,
				Animated:    true,
				Description: strings.TrimSpace(m.Description),
			})
			continue
		case "image":
			url = strings.TrimSpace(m.PreviewURL)
			if url == "" {
				url = strings.TrimSpace(m.URL)
			}
			animated = looksLikeGIF(url)
		default:
			continue
		}
		if url == "" {
			continue
		}
		if _, ok := seen[url]; ok {
			continue
		}
		seen[url] = struct{}{}
		out = append(out, mediaPreviewTarget{
			URL:         url,
			Animated:    animated,
			Description: strings.TrimSpace(m.Description),
		})
	}
	return out
}

func looksLikeGIF(rawURL string) bool {
	u, err := urlParse(rawURL)
	if err != nil {
		return strings.Contains(strings.ToLower(rawURL), ".gif")
	}
	return strings.EqualFold(path.Ext(u.Path), ".gif")
}

var urlParse = func(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL)
}

func (m *Model) advanceMediaFrames() {
	for key, frames := range m.mediaFrames {
		if len(frames) <= 1 {
			continue
		}
		m.mediaFrameIndex[key] = (m.mediaFrameIndex[key] + 1) % len(frames)
		m.mediaPreview[key] = frames[m.mediaFrameIndex[key]]
	}
}

var (
	ffmpegCheckOnce sync.Once
	ffmpegAvailable bool
)

func hasFFmpeg() bool {
	ffmpegCheckOnce.Do(func() {
		_, err := exec.LookPath("ffmpeg")
		ffmpegAvailable = err == nil
	})
	return ffmpegAvailable
}

func renderANSIFramesFromGIF(data []byte, w, h int, maxFrames int) ([]string, error) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if len(g.Image) <= 1 {
		return nil, fmt.Errorf("not animated")
	}
	if maxFrames <= 0 {
		maxFrames = 8
	}
	n := min(len(g.Image), maxFrames)
	frames := make([]string, 0, n)
	for i := range n {
		frames = append(frames, renderANSIThumbnail(g.Image[i], w, h))
	}
	return frames, nil
}

func renderANSIFramesFromVideo(url string, w, h int, maxFrames int) ([]string, error) {
	if !hasFFmpeg() {
		return nil, fmt.Errorf("ffmpeg unavailable")
	}
	if maxFrames <= 0 {
		maxFrames = 8
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	filter := fmt.Sprintf("fps=4,scale=%d:%d:flags=lanczos", max(w*2, 16), max(h*2, 8))
	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-i", url,
		"-vf", filter,
		"-frames:v", fmt.Sprintf("%d", maxFrames),
		"-f", "gif",
		"-",
	)
	data, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return renderANSIFramesFromGIF(data, w, h, maxFrames)
}

func fetchMediaPreview(url, fallbackURL, key string, w, h int, animated bool) tea.Cmd {
	return func() tea.Msg {
		// For video/gif media, try animated ASCII first.
		if animated {
			if frames, err := renderANSIFramesFromVideo(url, w, h, 8); err == nil && len(frames) > 0 {
				return MediaPreviewLoadedMsg{Key: key, Preview: frames[0], Frames: frames}
			}
		}

		preview, frames, err := loadStaticMediaPreview(url, w, h, animated)
		if err == nil {
			return MediaPreviewLoadedMsg{Key: key, Preview: preview, Frames: frames}
		}
		if fallbackURL != "" {
			preview, frames, ferr := loadStaticMediaPreview(fallbackURL, w, h, false)
			if ferr == nil {
				return MediaPreviewLoadedMsg{Key: key, Preview: preview, Frames: frames}
			}
		}
		return MediaPreviewLoadedMsg{Key: key, Err: err}
	}
}

func loadStaticMediaPreview(url string, w, h int, allowGIFAnimation bool) (string, []string, error) {
	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("preview status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return "", nil, err
	}
	if allowGIFAnimation {
		if frames, err := renderANSIFramesFromGIF(data, w, h, 8); err == nil && len(frames) > 0 {
			return frames[0], frames, nil
		}
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", nil, err
	}
	return renderANSIThumbnail(img, w, h), nil, nil
}

func mediaOpenURLs(media []domain.MediaAttachment) []string {
	out := make([]string, 0, len(media))
	seen := make(map[string]struct{}, len(media))
	for _, m := range media {
		url := strings.TrimSpace(m.URL)
		if url == "" {
			url = strings.TrimSpace(m.PreviewURL)
		}
		if url == "" {
			continue
		}
		if _, ok := seen[url]; ok {
			continue
		}
		seen[url] = struct{}{}
		out = append(out, url)
	}
	return out
}

func mediaPreviewBaseKey(url string) string {
	return "base|" + url
}

func renderANSIThumbnail(img image.Image, w, h int) string {
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return ""
	}
	if w < 4 {
		w = 4
	}
	if h < 2 {
		h = 2
	}

	drawW := w
	drawH := int(math.Round(float64(b.Dy()) * float64(drawW) / float64(b.Dx())))
	if drawH > h {
		drawH = h
		drawW = int(math.Round(float64(b.Dx()) * float64(drawH) / float64(b.Dy())))
	}
	if drawW < 1 {
		drawW = 1
	}
	if drawH < 1 {
		drawH = 1
	}
	offsetX := (w - drawW) / 2
	offsetY := (h - drawH) / 2
	bg := color.NRGBA{R: 12, G: 12, B: 12, A: 255}

	var out strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := bg
			if x >= offsetX && x < offsetX+drawW && y >= offsetY && y < offsetY+drawH {
				cx := x - offsetX
				cy := y - offsetY
				c = sampleAveragedColor(img, b, cx, cy, drawW, drawH)
			}
			fmt.Fprintf(&out, "\x1b[48;2;%d;%d;%dm  \x1b[0m", c.R, c.G, c.B)
		}
		if y < h-1 {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

func sampleAveragedColor(img image.Image, b image.Rectangle, cellX, cellY, cellsW, cellsH int) color.NRGBA {
	if cellsW <= 0 || cellsH <= 0 {
		return color.NRGBA{}
	}
	fx0 := float64(cellX) / float64(cellsW)
	fx1 := float64(cellX+1) / float64(cellsW)
	fy0 := float64(cellY) / float64(cellsH)
	fy1 := float64(cellY+1) / float64(cellsH)

	points := [4][2]float64{
		{fx0*0.75 + fx1*0.25, fy0*0.75 + fy1*0.25},
		{fx0*0.25 + fx1*0.75, fy0*0.75 + fy1*0.25},
		{fx0*0.75 + fx1*0.25, fy0*0.25 + fy1*0.75},
		{fx0*0.25 + fx1*0.75, fy0*0.25 + fy1*0.75},
	}

	var sumR, sumG, sumB int
	for _, p := range points {
		sx := b.Min.X + int(p[0]*float64(b.Dx()))
		sy := b.Min.Y + int(p[1]*float64(b.Dy()))
		if sx >= b.Max.X {
			sx = b.Max.X - 1
		}
		if sy >= b.Max.Y {
			sy = b.Max.Y - 1
		}
		c := color.NRGBAModel.Convert(img.At(sx, sy)).(color.NRGBA)
		sumR += int(c.R)
		sumG += int(c.G)
		sumB += int(c.B)
	}
	return color.NRGBA{
		R: uint8(sumR / len(points)),
		G: uint8(sumG / len(points)),
		B: uint8(sumB / len(points)),
		A: 255,
	}
}
