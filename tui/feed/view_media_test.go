package feed

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/CrestNiraj12/terminalrant/domain"
)

func TestRenderSelectedMediaPreviewPanel_FeedShowsSinglePreviewWithCount(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showMediaPreview = true
	m.rants = []RantItem{{
		Rant: domain.Rant{
			ID:        "p1",
			Content:   "media",
			CreatedAt: time.Now(),
			Media: []domain.MediaAttachment{
				{Type: "image", PreviewURL: "u1", Description: "first alt text for image one"},
				{Type: "image", PreviewURL: "u2"},
				{Type: "image", PreviewURL: "u3"},
				{Type: "image", PreviewURL: "u4"},
				{Type: "image", PreviewURL: "u5"},
			},
		},
	}}
	m.cursor = 0
	m.mediaPreview[mediaPreviewBaseKey("u1")] = "p1"
	m.mediaPreview[mediaPreviewBaseKey("u2")] = "p2"
	m.mediaPreview[mediaPreviewBaseKey("u3")] = "p3"
	m.mediaPreview[mediaPreviewBaseKey("u4")] = "p4"
	m.mediaPreview[mediaPreviewBaseKey("u5")] = "p5"

	panel := m.renderSelectedMediaPreviewPanel()
	if !strings.Contains(panel, "p1") {
		t.Fatalf("expected first preview in feed panel: %q", panel)
	}
	if strings.Contains(panel, "p2") {
		t.Fatalf("expected only one preview in feed panel: %q", panel)
	}
	if !strings.Contains(panel, "+4") {
		t.Fatalf("expected remaining media count in panel: %q", panel)
	}
	if !strings.Contains(panel, "alt:") {
		t.Fatalf("expected alt text above feed preview: %q", panel)
	}
}

func TestRenderSelectedMediaPreviewPanel_DetailShowsAllPreviews(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showMediaPreview = true
	m.showDetail = true
	m.rants = []RantItem{{
		Rant: domain.Rant{
			ID:        "p1",
			Content:   "media",
			CreatedAt: time.Now(),
			Media: []domain.MediaAttachment{
				{Type: "image", PreviewURL: "u1", Description: "first detailed alt"},
				{Type: "image", PreviewURL: "u2"},
				{Type: "image", PreviewURL: "u3"},
				{Type: "image", PreviewURL: "u4"},
				{Type: "image", PreviewURL: "u5"},
			},
		},
	}}
	m.cursor = 0
	m.mediaPreview[mediaPreviewBaseKey("u1")] = "p1"
	m.mediaPreview[mediaPreviewBaseKey("u2")] = "p2"
	m.mediaPreview[mediaPreviewBaseKey("u3")] = "p3"
	m.mediaPreview[mediaPreviewBaseKey("u4")] = "p4"
	m.mediaPreview[mediaPreviewBaseKey("u5")] = "p5"

	panel := m.renderSelectedMediaPreviewPanel()
	for _, p := range []string{"p1", "p2", "p3", "p4", "p5"} {
		if !strings.Contains(panel, p) {
			t.Fatalf("expected detail panel to include %s: %q", p, panel)
		}
	}
	if strings.Contains(panel, "+") {
		t.Fatalf("expected no feed overflow marker in detail panel: %q", panel)
	}
	if !strings.Contains(panel, "alt:") {
		t.Fatalf("expected alt labels above detail previews: %q", panel)
	}
}

func TestPreviewSizing_UsesFixedMediaDimensions(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showDetail = true
	m.width = 88

	asciiW, asciiH, tileW, tileH := m.previewSizing(4)
	if asciiW != previewASCIIWidth || asciiH != previewASCIIHeight {
		t.Fatalf("expected fixed ascii dimensions, got %dx%d", asciiW, asciiH)
	}
	if tileW != previewTileWidth || tileH != previewTileHeight {
		t.Fatalf("expected fixed tile dimensions, got %dx%d", tileW, tileH)
	}
}

func TestRenderSelectedMediaPreviewPanel_SingleUsesBasePreview(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showMediaPreview = true
	m.rants = []RantItem{{
		Rant: domain.Rant{
			ID:        "p1",
			Content:   "media",
			CreatedAt: time.Now(),
			Media:     []domain.MediaAttachment{{Type: "image", PreviewURL: "u1"}},
		},
	}}
	m.cursor = 0
	m.mediaPreview[mediaPreviewBaseKey("u1")] = "BASE_PREVIEW"

	panel := m.renderSelectedMediaPreviewPanel()
	if !strings.Contains(panel, "BASE_PREVIEW") {
		t.Fatalf("expected base preview in single-image panel")
	}
}

func TestPreviewURLCandidates_DedupesAndStripsQuery(t *testing.T) {
	out := previewURLCandidates(
		"https://cdn.example/avatar.webp?x=1&y=2",
		"https://cdn.example/avatar.webp?x=1&y=2",
	)
	if len(out) != 2 {
		t.Fatalf("expected 2 unique candidates, got %d: %#v", len(out), out)
	}
	if out[0] != "https://cdn.example/avatar.webp?x=1&y=2" {
		t.Fatalf("unexpected first candidate: %q", out[0])
	}
	if out[1] != "https://cdn.example/avatar.webp" {
		t.Fatalf("unexpected stripped candidate: %q", out[1])
	}
}

func TestEnsureMediaPreviewCmd_SingleImageQueuesBaseOnly(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	m.showMediaPreview = true
	m.rants = []RantItem{{
		Rant: domain.Rant{
			ID:        "p1",
			Content:   "media",
			CreatedAt: time.Now(),
			Media:     []domain.MediaAttachment{{Type: "image", PreviewURL: "u1"}},
		},
	}}
	m.cursor = 0

	cmd := m.ensureMediaPreviewCmd()
	if cmd == nil {
		t.Fatalf("expected preview fetch command")
	}
	if !m.mediaLoading[mediaPreviewBaseKey("u1")] {
		t.Fatalf("expected base preview request queued")
	}
	if len(m.mediaLoading) != 1 {
		t.Fatalf("expected exactly one preview request queued, got %d", len(m.mediaLoading))
	}
}

func TestRenderMediaDetail_RendersItems(t *testing.T) {
	out := renderMediaDetail([]domain.MediaAttachment{
		{Type: "image", Width: 10, Height: 20, Description: "desc", URL: "https://x"},
		{Type: "video", Width: 30, Height: 40, URL: "https://y"},
	})
	if !strings.Contains(out, "Media (2)") || !strings.Contains(out, "image 10x20") || !strings.Contains(out, "video 30x40") {
		t.Fatalf("unexpected media detail rendering: %q", out)
	}
	if strings.Contains(out, "desc") {
		t.Fatalf("expected alt description to be omitted from media info list: %q", out)
	}
}

func TestWrapAndTruncate_LimitsToThreeLines(t *testing.T) {
	text := "one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen"
	lines := wrapAndTruncate(text, 10, 3)
	if len(lines) != 3 {
		t.Fatalf("expected exactly three lines, got %d", len(lines))
	}
	if !strings.HasSuffix(lines[2], "...") {
		t.Fatalf("expected truncated final line with ellipsis: %#v", lines)
	}
}

func TestRenderANSIThumbnail_HighResolutionSamplesMorePixels(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x * 8), G: uint8(y * 8), B: 100, A: 255})
		}
	}

	base := renderANSIThumbnail(img, 12, 6)
	high := renderANSIThumbnail(img, 24, 12)
	basePixels := strings.Count(base, "\x1b[48;2;")
	highPixels := strings.Count(high, "\x1b[48;2;")

	if basePixels != 72 {
		t.Fatalf("expected 72 sampled pixels for base preview, got %d", basePixels)
	}
	if highPixels != 288 {
		t.Fatalf("expected 288 sampled pixels for high preview, got %d", highPixels)
	}
	if highPixels <= basePixels {
		t.Fatalf("expected high-resolution preview to sample more pixels (%d <= %d)", highPixels, basePixels)
	}
}

func TestRenderANSIThumbnail_PreservesAspectRatioInsideFixedBounds(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 16)) // 4:1 landscape
	for y := 0; y < 16; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.NRGBA{R: 220, G: 80, B: 20, A: 255})
		}
	}

	thumb := renderANSIThumbnail(img, 12, 8)
	bgPixels := strings.Count(thumb, "\x1b[48;2;12;12;12m")
	if bgPixels == 0 {
		t.Fatalf("expected letterboxing background pixels for preserved aspect ratio")
	}
}

func TestMediaPreviewTargets_VideoUsesSourceWithFallback(t *testing.T) {
	targets := mediaPreviewTargets([]domain.MediaAttachment{
		{Type: "video", URL: "https://cdn/video.mp4", PreviewURL: "https://cdn/preview.jpg"},
		{Type: "image", URL: "https://cdn/image.jpg", PreviewURL: "https://cdn/image-preview.jpg"},
		{Type: "gifv", URL: "https://cdn/anim.gif", PreviewURL: "https://cdn/anim-preview.jpg"},
	})
	if len(targets) != 3 {
		t.Fatalf("unexpected targets count: %d", len(targets))
	}
	if targets[0].URL != "https://cdn/video.mp4" || targets[0].FallbackURL != "https://cdn/preview.jpg" || !targets[0].Animated {
		t.Fatalf("unexpected video target: %#v", targets[0])
	}
	if targets[1].URL != "https://cdn/image-preview.jpg" || targets[1].Animated {
		t.Fatalf("unexpected image target: %#v", targets[1])
	}
	if targets[2].URL != "https://cdn/anim.gif" || !targets[2].Animated {
		t.Fatalf("unexpected gifv target: %#v", targets[2])
	}
}

func TestAdvanceMediaFrames_CyclesPreview(t *testing.T) {
	m := New(stubTimeline{}, stubAccount{}, "terminalrant", "terminalrant")
	key := mediaPreviewBaseKey("https://cdn/video.mp4")
	m.mediaFrames[key] = []string{"f1", "f2", "f3"}
	m.mediaFrameIndex[key] = 0
	m.mediaPreview[key] = "f1"

	m.advanceMediaFrames()
	if m.mediaPreview[key] != "f2" {
		t.Fatalf("expected frame 2 after first tick, got %q", m.mediaPreview[key])
	}
	m.advanceMediaFrames()
	if m.mediaPreview[key] != "f3" {
		t.Fatalf("expected frame 3 after second tick, got %q", m.mediaPreview[key])
	}
	m.advanceMediaFrames()
	if m.mediaPreview[key] != "f1" {
		t.Fatalf("expected wrap to frame 1, got %q", m.mediaPreview[key])
	}
}
